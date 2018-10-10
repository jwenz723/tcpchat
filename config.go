package main

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/sirupsen/logrus"
)

// Config defines a struct to match a configuration yaml file.
type Config struct {
	HTTPAddress 		string 		`yaml:"HTTPAddress"`
	HTTPPort 			int 		`yaml:"HTTPPort"`
	LogDirectory 		string 		`yaml:"LogDirectory"`
	LogJSON 			bool		`yaml:"LogJSON"`
	LogLevel 			string 		`yaml:"LogLevel"`
	TCPAddress 			string 		`yaml:"TCPAddress"`
	TCPPort 			int 		`yaml:"TCPPort"`
}

// NewConfig will create a new Config instance from the specified yaml file
func NewConfig(yamlFile string) (*Config, error) {
	config := Config{}
	source, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(source, &config)
	if err != nil {
		return nil, err
	}

	// Ensure a proper LogLevel was provided
	if config.LogLevel == "" {
		config.LogLevel = "info"
	} else {
		_, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			return nil, err
		}
	}

	// Set a default port for the HTTP listener
	if config.HTTPPort == 0 {
		config.HTTPPort = 8080
	}

	// Set a default port for the TCP listener
	if config.TCPPort == 0 {
		config.TCPPort = 6000
	}

	return &config, nil
}