package ripper

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/grafana/dskit/services"

	"github.com/zachfi/streamgo/pkg/shoutcast"
)

type Ripper struct {
	services.Service
	cfg     *Config
	logger  *slog.Logger
	stream  *shoutcast.Stream
	w       *ChannelWriter
	copyWg  sync.WaitGroup // signals when the io.Copy goroutine has exited
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
	var writerDone chan struct{} // closed when the current writer goroutine exits

	bufferMutex := &sync.Mutex{}

	cw := NewChannelWriter()
	r.w = cw

	r.copyWg.Add(1)
	go func() {
		defer r.copyWg.Done()
		r.logger.Info("starting copy")
		b, copyErr := io.Copy(cw, r.stream)
		if copyErr != nil && copyErr != io.EOF {
			r.logger.Error("error copying stream to buffer", "err", copyErr, "written", ByteCountIEC(b))
		}
	}()

	fileName := ""

	r.stream.MetadataCallbackFunc = func(m *shoutcast.Metadata) {
		r.logger.Info("now listening to", "title", m.StreamTitle)

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

			// Cancel previous writer, then wait for it to exit so only one goroutine
			// reads from the channel at a time (avoids splitting the stream).
			if cancel != nil {
				cancel()
			}
			if writerDone != nil {
				<-writerDone
			}

			f, err = os.Create(name)
			if err != nil {
				r.logger.Error("error creating file", "err", err)
			}

			wCtx, cancel = context.WithCancel(ctx)
			writerDone = make(chan struct{})
			done := writerDone
			r.logger.Debug("starting new writer")
			go func() {
				defer close(done)
				r.writeToFile(wCtx, cw.dataChan, bufferMutex, f)
			}()
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
	// Close stream first so io.Copy gets EOF and exits; then wait for copy
	// goroutine before closing the channel (otherwise we get "read/write on closed pipe").
	if r.stream != nil {
		if err := r.stream.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	r.copyWg.Wait()

	if r.w != nil {
		if err := r.w.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (r *Ripper) writeToFile(ctx context.Context, dataChan chan []byte, bufferMutex *sync.Mutex, f *os.File) {
	var err error
	firstWrite := true
	buffer := make([]byte, 0, 4096) // Buffer to accumulate data until we find frame sync

	for {
		select {
		case <-ctx.Done():
			// Context canceled (new track started), stop writing and close file.
			r.logger.Debug("context canceled, closing file")
			if f != nil {
				// Write any remaining buffered data
				if len(buffer) > 0 {
					bufferMutex.Lock()
					_, err = f.Write(buffer)
					bufferMutex.Unlock()
				}
				// Flush and sync before closing to ensure all data is written
				if syncErr := f.Sync(); syncErr != nil {
					r.logger.Error("error syncing file", "err", syncErr)
				}
				if closeErr := f.Close(); closeErr != nil {
					r.logger.Error("error closing file", "err", closeErr)
				}
			}
			return
		case b, ok := <-dataChan:
			if !ok {
				// Channel closed (shutdown); close file and exit
				if f != nil {
					if len(buffer) > 0 {
						bufferMutex.Lock()
						_, _ = f.Write(buffer)
						bufferMutex.Unlock()
					}
					_ = f.Sync()
					_ = f.Close()
				}
				return
			}
			if len(b) == 0 {
				continue
			}
			
			if firstWrite {
				// Find the first MP3 frame sync in the accumulated buffer + new data
				buffer = append(buffer, b...)
				framePos := findMP3FrameSync(buffer)
				if framePos >= 0 {
					// Found frame sync, write from that position
					bufferMutex.Lock()
					_, err = f.Write(buffer[framePos:])
					if err != nil {
						r.logger.Error("error writing to file", "err", err)
						bufferMutex.Unlock()
						return
					}
					bufferMutex.Unlock()
					buffer = buffer[:0] // Clear buffer
					firstWrite = false
				} else if len(buffer) > 8192 {
					// Buffer is getting large, write it anyway (might be valid MP3 without sync)
					r.logger.Warn("no MP3 frame sync found in first 8KB, writing anyway")
					bufferMutex.Lock()
					_, err = f.Write(buffer)
					if err != nil {
						r.logger.Error("error writing to file", "err", err)
						bufferMutex.Unlock()
						return
					}
					bufferMutex.Unlock()
					buffer = buffer[:0]
					firstWrite = false
				}
				// Otherwise, keep buffering
			} else {
				// Normal write after finding first frame
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
}
