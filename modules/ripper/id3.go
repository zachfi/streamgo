package ripper

import "io"

// writeID3v2Tag writes a minimal ID3v2.4 tag containing a single TIT2 (title)
// frame to w. The title is encoded as UTF-8 (text encoding byte 0x03).
func writeID3v2Tag(w io.Writer, title string) error {
	// Frame content: encoding byte (0x03 = UTF-8) + title bytes
	titleBytes := []byte(title)
	frameContent := make([]byte, 1+len(titleBytes))
	frameContent[0] = 0x03 // UTF-8
	copy(frameContent[1:], titleBytes)

	// TIT2 frame: 4-byte ID + 4-byte big-endian size + 2-byte flags + content
	frameSize := len(frameContent)
	frame := make([]byte, 10+frameSize)
	copy(frame[0:4], "TIT2")
	frame[4] = byte(frameSize >> 24)
	frame[5] = byte(frameSize >> 16)
	frame[6] = byte(frameSize >> 8)
	frame[7] = byte(frameSize)
	// flags: 0x00 0x00
	copy(frame[10:], frameContent)

	// ID3v2.4 tag header: "ID3" + version (0x04 0x00) + flags (0x00) + synchsafe size
	tagContentSize := len(frame)
	header := [10]byte{}
	copy(header[0:3], "ID3")
	header[3] = 0x04 // version 2.4
	// header[4] = revision 0x00, header[5] = flags 0x00 (already zero)
	// Encode tag content size as synchsafe integer (7 bits per byte)
	header[6] = byte((tagContentSize >> 21) & 0x7F)
	header[7] = byte((tagContentSize >> 14) & 0x7F)
	header[8] = byte((tagContentSize >> 7) & 0x7F)
	header[9] = byte(tagContentSize & 0x7F)

	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(frame)
	return err
}
