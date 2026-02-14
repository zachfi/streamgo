package ripper

import (
	"flag"

	"github.com/zachfi/zkit/pkg/util"
)

// Write buffer sizing guidance (write-buffer-size):
// - SSD wear: fewer, larger writes reduce I/O overhead; 256KiB–1MiB is a good range.
// - NFS: larger buffers amortize round-trip cost; 512KiB–1MiB often performs better than 256KiB.
// - Upper bound: config is clamped to 4MiB to limit memory and avoid huge single writes.
const (
	defaultWriteBufferSize = 256 * 1024 // 256 KiB
)

type Config struct {
	URL              string `yaml:"url,omitempty"`
	Dir              string `yaml:"dir,omitempty"`
	WriteBufferSize  int    `yaml:"write-buffer-size,omitempty"` // bytes to buffer before writing (reduces write frequency)
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.URL, util.PrefixConfig(prefix, "url"), "", "The URL from which to stream")
	f.StringVar(&cfg.Dir, util.PrefixConfig(prefix, "dir"), "", "The directory to save the data")
	f.IntVar(&cfg.WriteBufferSize, util.PrefixConfig(prefix, "write-buffer-size"), defaultWriteBufferSize,
		"Bytes to buffer in memory before writing to disk (default 256KiB). Larger values reduce write frequency (helps SSD longevity and NFS). Reasonable range: 256KiB-1MiB.")
}
