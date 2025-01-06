package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Database struct {
		Dialect          string `yaml:"dialect"`
		ConnectionString string `yaml:"connection_string"`
	} `yaml:"database"`

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
