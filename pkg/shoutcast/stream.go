package shoutcast

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

// MetadataCallbackFunc is the type of the function called when the stream metadata changes
type MetadataCallbackFunc func(m *Metadata)

// Stream represents an open shoutcast stream.
type Stream struct {
	// The name of the server
	Name string

	// What category the server falls under
	Genre string

	// The description of the stream
	Description string

	// Homepage of the server
	URL string

	// Bitrate of the server
	Bitrate int

	// Optional function to be executed when stream metadata changes
	MetadataCallbackFunc MetadataCallbackFunc

	// Amount of bytes to read before expecting a metadata block
	metaint int

	// Stream metadata
	metadata *Metadata

	// The number of bytes read since last metadata block
	pos int

	// The underlying data stream
	rc io.ReadCloser
}

// Open establishes a connection to a remote server.
// It automatically handles playlist files (.pls, .m3u) and resolves them to stream URLs.
func Open(url string) (*Stream, error) {
	log.Print("[INFO] Opening ", url)

	// Check if URL is a playlist and resolve it
	resolvedURL, err := resolvePlaylistURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve playlist URL: %w", err)
	}
	if resolvedURL != url {
		log.Print("[INFO] Resolved playlist to stream URL: ", resolvedURL)
		url = resolvedURL
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("accept", "*/*")
	req.Header.Add("user-agent", "iTunes/12.9.2 (Macintosh; OS X 10.14.3) AppleWebKit/606.4.5")
	req.Header.Add("icy-metadata", "1")

	// Timeout for establishing the connection.
	// We don't want for the stream to timeout while we're reading it, but
	// we do want a timeout for establishing the connection to the server.
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Dial: dialer.Dial,
		// Disable timeouts for streaming - we want to read indefinitely
		ResponseHeaderTimeout: 10 * time.Second, // Only timeout on initial connection
	}
	// No timeout on the client - we want to stream indefinitely
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	for k, v := range resp.Header {
		log.Print("[DEBUG] HTTP header ", k, ": ", v[0])
	}

	var bitrate int
	if rawBitrate := resp.Header.Get("icy-br"); rawBitrate != "" {
		bitrate, err = strconv.Atoi(rawBitrate)
		if err != nil {
			return nil, fmt.Errorf("cannot parse bitrate: %v", err)
		}
	}

	metaint, err := strconv.Atoi(resp.Header.Get("icy-metaint"))
	if err != nil {
		return nil, fmt.Errorf("cannot parse metaint: %v", err)
	}

	s := &Stream{
		Name:        resp.Header.Get("icy-name"),
		Genre:       resp.Header.Get("icy-genre"),
		Description: resp.Header.Get("icy-description"),
		URL:         resp.Header.Get("icy-url"),
		Bitrate:     bitrate,
		metaint:     metaint,
		metadata:    nil,
		pos:         0,
		rc:          resp.Body,
	}

	return s, nil
}

// Read implements the standard Read interface
func (s *Stream) Read(buf []byte) (dataLen int, err error) {
	// We need to read and process data in a way that handles metadata blocks
	// that may span across multiple Read calls. We'll use a simpler approach:
	// read audio data in chunks of metaint bytes, then skip metadata.

	cleanBuf := make([]byte, 0, len(buf))

	for len(cleanBuf) < len(buf) {
		// Calculate how many audio bytes we need
		bytesNeeded := len(buf) - len(cleanBuf)
		bytesUntilMetadata := s.metaint - s.pos

		if bytesUntilMetadata == 0 {
			// We're at a metadata boundary, extract and skip it
			// Read the metadata length byte
			var metaLenByte [1]byte
			var n int
			n, err = s.rc.Read(metaLenByte[:])
			if err != nil && err != io.EOF {
				break
			}
			if n == 0 {
				if err == nil {
					err = io.EOF
				}
				break
			}

			metaBlockLen := int(metaLenByte[0]) * 16
			if metaBlockLen > 0 {
				// Read and parse the metadata block
				metaBuf := make([]byte, metaBlockLen)
				n = 0
				for n < metaBlockLen && err == nil {
					var nn int
					nn, err = s.rc.Read(metaBuf[n:])
					n += nn
				}
				if n == metaBlockLen {
					// Parse and process metadata
					if m := NewMetadata(metaBuf); !m.Equals(s.metadata) {
						s.metadata = m
						if s.MetadataCallbackFunc != nil {
							s.MetadataCallbackFunc(s.metadata)
						}
					}
				} else if err == nil || err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
			}
			// Empty metadata block (length byte was 0), nothing more to read
			s.pos = 0
			continue
		}

		// Read audio data up to the next metadata block or until we have enough
		bytesToRead := bytesUntilMetadata
		if bytesToRead > bytesNeeded {
			bytesToRead = bytesNeeded
		}

		// Read directly into cleanBuf
		startLen := len(cleanBuf)
		cleanBuf = append(cleanBuf, make([]byte, bytesToRead)...)
		n, readErr := s.rc.Read(cleanBuf[startLen:])
		if n == 0 {
			if readErr != nil && readErr != io.EOF {
				err = readErr
			} else if err == nil {
				err = io.EOF
			}
			cleanBuf = cleanBuf[:startLen]
			break
		}
		cleanBuf = cleanBuf[:startLen+n]
		s.pos += n

		if readErr != nil && readErr != io.EOF {
			err = readErr
			break
		}

		// If we've read metaint bytes, we're at a metadata boundary - read and skip the metadata block
		if s.pos >= s.metaint {
			s.pos = 0
			// Read the metadata length byte
			var metaLenByte [1]byte
			var mn int
			mn, err = s.rc.Read(metaLenByte[:])
			if err != nil && err != io.EOF {
				break
			}
			if mn == 0 {
				if err == nil {
					err = io.EOF
				}
				break
			}
			metaBlockLen := int(metaLenByte[0]) * 16
			if metaBlockLen > 0 {
				metaBuf := make([]byte, metaBlockLen)
				mn = 0
				for mn < metaBlockLen && err == nil {
					var nn int
					nn, err = s.rc.Read(metaBuf[mn:])
					mn += nn
				}
				if mn == metaBlockLen && s.MetadataCallbackFunc != nil {
					if m := NewMetadata(metaBuf); !m.Equals(s.metadata) {
						s.metadata = m
						s.MetadataCallbackFunc(s.metadata)
					}
				} else if mn < metaBlockLen && (err == nil || err == io.EOF) {
					err = io.ErrUnexpectedEOF
				}
			}
		}
	}

	// Copy the clean buffer to the output buffer
	dataLen = len(cleanBuf)
	if dataLen > 0 {
		copy(buf, cleanBuf)
	}

	return dataLen, err
}

// Close closes the stream
func (s *Stream) Close() error {
	log.Print("[INFO] Closing ", s.URL)
	return s.rc.Close()
}
