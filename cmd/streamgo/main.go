package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/romantomjak/shoutcast"
	"gopkg.in/yaml.v2"

	"github.com/zachfi/zkit/pkg/tracing"

	"github.com/zachfi/streamgo/app"
)

const appName = "streamgo"

// Version is set via build flag -ldflags -X main.Version
var (
	Version  string
	Branch   string
	Revision string
)

func init() {
	version.Version = Version
	version.Branch = Branch
	version.Revision = Revision
	prometheus.MustRegister(version.NewCollector(appName))
}

func main() {
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config file", "err", err)
		os.Exit(1)
	}

	shutdownTracer, err := tracing.InstallOpenTelemetryTracer(&cfg.Tracing, logger, appName, Version)
	if err != nil {
		slog.Error("error initialising tracer", "err", err)
		os.Exit(1)
	}
	defer shutdownTracer()

	a, err := app.New(*cfg, *logger)
	if err != nil {
		slog.Error("failed to create", "app", appName, "err", err)
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		slog.Error("error running", "app", appName, "err", err)
		os.Exit(1)
	}

	url := "http://ice6.somafm.com/groovesalad-256-mp3"

	stream, err := shoutcast.Open(url)
	if err != nil {
		slog.Error("error opening stream", "err", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var f *os.File
	var ctx context.Context
	var cancel context.CancelFunc

	bufferMutex := &sync.Mutex{}

	cw := NewChannelWriter()

	wg.Add(1)
	go func() {
		slog.Info("starting copy")
		_, copyErr := io.Copy(cw, stream)
		if copyErr != nil {
			slog.Error("error copying stream to buffer", "err", copyErr)
		}
	}()

	fileName := ""

	stream.MetadataCallbackFunc = func(m *shoutcast.Metadata) {
		slog.Info("Now listening to", "title", m.StreamTitle)

		time.Sleep(5 * time.Second)
		if f != nil {
			f.Close()
		}

		err := os.MkdirAll(stream.Name, os.ModePerm)
		if err != nil {
			slog.Error("error creating stream directory", "err", err)
		}

		name := path.Join(stream.Name, m.StreamTitle+".mp3")

		if name != fileName {
			fileName = name

			f, err = os.Create(name)
			if err != nil {
				slog.Error("error creating file", "err", err)
			}

			if cancel != nil {
				cancel()
			}

			ctx, cancel = context.WithCancel(context.Background())
			slog.Debug("starting new writer")
			go writeToFile(ctx, cw.dataChan, bufferMutex, f)
		}
	}

	wg.Wait()
}

func writeToFile(ctx context.Context, dataChan chan []byte, bufferMutex *sync.Mutex, f *os.File) {
	var err error

	for {
		select {
		case <-ctx.Done():
			// Context canceled, stop writing to the file.
			slog.Info("Context canceled, stopping writing to file.")
			return

		case b := <-dataChan:
			bufferMutex.Lock()
			_, err = f.Write(b)
			if err != nil {
				slog.Error("error writing to file", "err", err)
				return
			}
			bufferMutex.Unlock()
		}
	}
}

type ChannelWriter struct {
	sync.Mutex
	dataChan chan []byte
	closed   bool
}

func NewChannelWriter() *ChannelWriter {
	return &ChannelWriter{
		dataChan: make(chan []byte, 10240), // Buffer size can be adjusted as needed
	}
}

func (cw *ChannelWriter) Write(p []byte) (n int, err error) {
	cw.Lock()
	defer cw.Unlock()

	if cw.closed {
		return 0, io.ErrClosedPipe
	}

	cw.dataChan <- p

	return len(p), nil
}

func (cw *ChannelWriter) Close() error {
	cw.Lock()
	defer cw.Unlock()

	if !cw.closed {
		close(cw.dataChan)
		cw.closed = true
	}

	return nil
}

func loadConfig() (*app.Config, error) {
	const (
		configFileOption = "config.file"
	)

	var configFile string

	args := os.Args[1:]
	config := &app.Config{}

	// first get the config file
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&configFile, configFileOption, "", "")

	// Try to find -config.file & -config.expand-env flags. As Parsing stops on the first error, eg. unknown flag,
	// we simply try remaining parameters until we find config flag, or there are no params left.
	// (ContinueOnError just means that flag.Parse doesn't call panic or os.Exit, but it returns error, which we ignore)
	for len(args) > 0 {
		_ = fs.Parse(args)
		args = args[1:]
	}

	// load config defaults and register flags
	config.RegisterFlagsAndApplyDefaults("", flag.CommandLine)

	// overlay with config file if provided
	if configFile != "" {
		buff, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read configFile %s: %w", configFile, err)
		}

		err = yaml.UnmarshalStrict(buff, config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse configFile %s: %w", configFile, err)
		}
	}

	// overlay with cli
	flagext.IgnoredFlag(flag.CommandLine, configFileOption, "Configuration file to load")
	flag.Parse()

	return config, nil
}
