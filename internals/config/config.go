package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Database struct {
		ConnectionString string `yaml:"connection_string"`
	} `yaml:"database"`

	MasterServer struct {
		Enabled  bool   `yaml:"enabled"`
		Endpoint string `yaml:"endpoint"`
		AuthKey  string `yaml:"auth_key"`
	} `yaml:"master_server"`

	Scanner struct {
		NumWorkers     int           `yaml:"num_workers"`
		RequestTimeout time.Duration `yaml:"request_timeout"`
		Exts           []string      `yaml:"exts"`
	} `yaml:"scanner"`
}

func LoadConfig(filepath string) (*Config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}
