package ripper

import (
	"io"
	"sync"
)

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
