package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Docker struct {
		Domain struct {
			Endpoint string `yaml:"endpoint,omitempty"`
			Port     uint   `yaml:"port,omitempty"`
		} `yaml:"domain,omitempty"`
		TLS struct {
			Enabled  bool   `yaml:"enabled,omitempty"`
			CertFile string `yaml:"certFile,omitempty"`
			CertKey  string `yaml:"certKey,omitempty"`
			CAFile   string `yaml:"caFile,omitempty"`
		} `yaml:"tls,omitempty"`
		AUTH struct {
			AuthKey string `yaml:"aKey,omitempty"`
			SignKey string `yaml:"sKey,omitempty"`
		} `yaml:"auth,omitempty"`
	} `yaml:"docker,omitempty"`
}

func NewConfig(configPath string) (*Config, error) {

	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal(f, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}
