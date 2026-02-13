// Package shoutcast provides ICY/Shoutcast stream reading with metadata stripping and playlist resolution.
//
// It is a fork of github.com/romantomjak/shoutcast, extended for stream recording:
//   - Playlist resolution: .pls and .m3u URLs are resolved to the actual stream URL
//   - Correct metadata stripping: ICY metadata blocks are read and skipped so only audio bytes are returned
//   - No client timeout on the stream so long-running recording is supported
package shoutcast
