package ripper

import (
	"flag"

	"github.com/zachfi/zkit/pkg/util"
)

type Config struct {
	URL string `yaml:"url,omitempty"`
	Dir string `yaml:"dir,omitempty"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.URL, util.PrefixConfig(prefix, "url"), "", "The URL from which to stream")
	f.StringVar(&cfg.Dir, util.PrefixConfig(prefix, "dir"), "", "The directory to save the data")
}
