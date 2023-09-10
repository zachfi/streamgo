package ripper

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/grafana/dskit/services"
	"github.com/romantomjak/shoutcast"
)

type Ripper struct {
	services.Service
	cfg    *Config
	logger *slog.Logger
	stream *shoutcast.Stream
	w      *ChannelWriter
}

var module = "ripper"

// New creates and returns a new.
func New(cfg Config, logger slog.Logger) (*Ripper, error) {
	r := &Ripper{
		cfg:    &cfg,
		logger: logger.With("module", module),
	}

	r.Service = services.NewBasicService(r.starting, r.running, r.stopping)

	return r, nil
}

func (r *Ripper) starting(ctx context.Context) error {
	stream, err := shoutcast.Open(r.cfg.URL)
	if err != nil {
		r.logger.Error("error opening stream", "err", err)
		return err
	}

	r.stream = stream

	return nil
}

func (r *Ripper) running(ctx context.Context) error {
	var f *os.File
	var wCtx context.Context
	var cancel context.CancelFunc

	bufferMutex := &sync.Mutex{}

	cw := NewChannelWriter()
	r.w = cw

	go func() {
		r.logger.Info("starting copy")
		b, copyErr := io.Copy(cw, r.stream)
		if copyErr != nil {
			r.logger.Error("error copying stream to buffer", "err", copyErr, "written", ByteCountIEC(b))
			return
		}
	}()

	fileName := ""

	r.stream.MetadataCallbackFunc = func(m *shoutcast.Metadata) {
		r.logger.Info("now listening to", "title", m.StreamTitle)

		time.Sleep(5 * time.Second)
		if f != nil {
			f.Close()
		}

		var name string
		if r.cfg.Dir != "" {
			name = path.Join(r.cfg.Dir, r.stream.Name, m.StreamTitle+".mp3")
		} else {
			name = path.Join(r.stream.Name, m.StreamTitle+".mp3")
		}

		err := os.MkdirAll(path.Dir(name), os.ModePerm)
		if err != nil {
			r.logger.Error("error creating stream directory", "err", err)
		}

		if name != fileName {
			fileName = name

			f, err = os.Create(name)
			if err != nil {
				r.logger.Error("error creating file", "err", err)
			}

			if cancel != nil {
				cancel()
			}

			wCtx, cancel = context.WithCancel(ctx)
			r.logger.Debug("starting new writer")
			go r.writeToFile(wCtx, cw.dataChan, bufferMutex, f)
		}
	}

	if cancel != nil {
		cancel()
	}
	<-ctx.Done()
	return nil
}

func (r *Ripper) stopping(_ error) error {
	r.logger.Info("stopping")

	var errs []error
	var err error

	err = r.w.Close()
	if err != nil {
		errs = append(errs, err)
	}

	err = r.stream.Close()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (r *Ripper) writeToFile(ctx context.Context, dataChan chan []byte, bufferMutex *sync.Mutex, f *os.File) {
	var err error

	for {
		select {
		case <-ctx.Done():
			// Context canceled, stop writing to the file.
			r.logger.Info("context canceled, closing file")
			if f != nil {
				f.Close()
			}
			return
		case b := <-dataChan:
			bufferMutex.Lock()
			_, err = f.Write(b)
			if err != nil {
				r.logger.Error("error writing to file", "err", err)
				bufferMutex.Unlock()
				return
			}
			bufferMutex.Unlock()
		}
	}
}
