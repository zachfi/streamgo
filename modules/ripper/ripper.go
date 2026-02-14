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
	cfg    *Config
	logger *slog.Logger
	stream *shoutcast.Stream
	w      *ChannelWriter
	copyWg sync.WaitGroup // signals when the io.Copy goroutine has exited
}

var module = "ripper"

// New creates and returns a new.
func New(cfg Config, logger slog.Logger) (*Ripper, error) {
	if cfg.WriteBufferSize == 0 {
		cfg.WriteBufferSize = defaultWriteBufferSize
	}
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

			dir := path.Dir(name)
			tmpF, err := os.CreateTemp(dir, "*.mp3.tmp")
			if err != nil {
				r.logger.Error("error creating temp file", "err", err)
				return
			}
			f = tmpF

			wCtx, cancel = context.WithCancel(ctx)
			writerDone = make(chan struct{})
			done := writerDone
			r.logger.Debug("starting new writer")
			go func() {
				defer close(done)
				r.writeToFile(wCtx, cw.dataChan, bufferMutex, f, name)
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

// commitTempFile renames tempPath to destPath only if dest doesn't exist or
// the temp file is larger (so a previous crash doesn't overwrite a good recording).
func (r *Ripper) commitTempFile(tempPath, destPath string) {
	tempInfo, err := os.Stat(tempPath)
	if err != nil {
		r.logger.Error("error stating temp file", "err", err, "path", tempPath)
		_ = os.Remove(tempPath)
		return
	}
	destInfo, err := os.Stat(destPath)
	if err != nil {
		if !os.IsNotExist(err) {
			r.logger.Error("error stating dest file", "err", err, "path", destPath)
			_ = os.Remove(tempPath)
			return
		}
		// Dest doesn't exist; use the temp file.
		if err := os.Rename(tempPath, destPath); err != nil {
			r.logger.Error("error renaming temp to dest", "err", err, "temp", tempPath, "dest", destPath)
			_ = os.Remove(tempPath)
			return
		}
		r.logger.Debug("saved new recording", "path", destPath)
		return
	}
	if tempInfo.Size() > destInfo.Size() {
		if err := os.Rename(tempPath, destPath); err != nil {
			r.logger.Error("error renaming temp to dest", "err", err, "temp", tempPath, "dest", destPath)
			_ = os.Remove(tempPath)
			return
		}
		r.logger.Debug("overwrote with longer recording", "path", destPath, "size", tempInfo.Size())
	} else {
		_ = os.Remove(tempPath)
		r.logger.Debug("discarded shorter recording", "path", destPath, "temp_size", tempInfo.Size(), "existing_size", destInfo.Size())
	}
}

// minWriteBufSize and maxWriteBufSize clamp the configured write buffer to avoid
// tiny writes (no benefit) or very large buffers (memory and latency).
const (
	minWriteBufSize = 32 * 1024   // 32 KiB
	maxWriteBufSize = 4 * 1024 * 1024 // 4 MiB
)

func (r *Ripper) writeToFile(ctx context.Context, dataChan chan []byte, bufferMutex *sync.Mutex, f *os.File, destPath string) {
	writeBufSize := r.cfg.WriteBufferSize
	if writeBufSize < minWriteBufSize {
		writeBufSize = minWriteBufSize
	}
	if writeBufSize > maxWriteBufSize {
		writeBufSize = maxWriteBufSize
	}

	var err error
	firstWrite := true
	buffer := make([]byte, 0, 4096)       // Buffer to accumulate data until we find frame sync
	writeBuf := make([]byte, 0, writeBufSize) // Batch writes to reduce disk I/O

	flushWriteBuf := func() {
		if len(writeBuf) == 0 {
			return
		}
		bufferMutex.Lock()
		_, err = f.Write(writeBuf)
		bufferMutex.Unlock()
		if err != nil {
			r.logger.Error("error writing to file", "err", err)
			return
		}
		writeBuf = writeBuf[:0]
	}

	closeAndCommit := func() {
		if f == nil {
			return
		}
		tempPath := f.Name()
		// Flush any remaining buffered data (frame-sync buffer and write batch buffer)
		if len(buffer) > 0 {
			bufferMutex.Lock()
			_, _ = f.Write(buffer)
			bufferMutex.Unlock()
		}
		flushWriteBuf()
		if syncErr := f.Sync(); syncErr != nil {
			r.logger.Error("error syncing file", "err", syncErr)
		}
		if closeErr := f.Close(); closeErr != nil {
			r.logger.Error("error closing file", "err", closeErr)
		}
		r.commitTempFile(tempPath, destPath)
	}

	for {
		select {
		case <-ctx.Done():
			// Context canceled (new track started), stop writing and close file.
			r.logger.Debug("context canceled, closing file")
			closeAndCommit()
			return
		case b, ok := <-dataChan:
			if !ok {
				// Channel closed (shutdown); close file and exit
				closeAndCommit()
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
				// Normal write: batch in memory and only write when buffer is large enough
				writeBuf = append(writeBuf, b...)
				if len(writeBuf) >= writeBufSize {
					flushWriteBuf()
				}
			}
		}
	}
}
