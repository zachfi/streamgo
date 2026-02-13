package ripper

// findMP3FrameSync finds the position of the first valid MP3 frame sync word.
// MP3 frame sync is: 0xFF followed by 0xE or 0xF in the high nibble (bits 8-11 = 1111)
// More precisely: 0xFF followed by a byte where bits 4-7 are 1110 or 1111
// Returns -1 if not found.
func findMP3FrameSync(data []byte) int {
	for i := 0; i < len(data)-1; i++ {
		// MP3 frame sync: 0xFF followed by 0xE or 0xF in high nibble
		// Check: first byte is 0xFF, second byte has 0xE or 0xF in bits 4-7
		if data[i] == 0xFF && (data[i+1]&0xF0) >= 0xE0 {
			// Additional check: bits 4-7 should be 1110 or 1111
			// This means (data[i+1] & 0xF0) should be 0xE0 or 0xF0
			if (data[i+1] & 0xF0) == 0xE0 || (data[i+1]&0xF0) == 0xF0 {
				return i
			}
		}
	}
	return -1
}
