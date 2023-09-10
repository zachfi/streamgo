package app

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/server"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/zachfi/zkit/pkg/tracing"

	"github.com/zachfi/streamgo/modules/ripper"
)

type Config struct {
	Target  string         `yaml:"target"`
	Tracing tracing.Config `yaml:"tracing,omitempty"`
	Server  server.Config  `yaml:"server,omitempty"`
	Ripper  ripper.Config  `yaml:"ripper,omitempty"`
}

// LoadConfig receives a file path for a configuration to load.
func LoadConfig(file string) (Config, error) {
	filename, _ := filepath.Abs(file)

	config := Config{}
	err := loadYamlFile(filename, &config)
	if err != nil {
		return config, errors.Wrap(err, "failed to load yaml file")
	}

	return config, nil
}

// loadYamlFile unmarshals a YAML file into the received interface{} or returns an error.
func loadYamlFile(filename string, d interface{}) error {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, d)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	flagext.DefaultValues(&c.Server)
	f.IntVar(&c.Server.HTTPListenPort, "server.http-listen-port", 3030, "HTTP server listen port.")
	f.IntVar(&c.Server.GRPCListenPort, "server.grpc-listen-port", 9090, "gRPC server listen port.")

	c.Tracing.RegisterFlagsAndApplyDefaults("tracing", f)
	c.Ripper.RegisterFlagsAndApplyDefaults("ripper", f)
}
