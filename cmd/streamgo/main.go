package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
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
