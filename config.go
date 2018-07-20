package main

import (
	"io/ioutil"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

// UnmarshalYAML overrides what happens when the yaml.Unmarshal function is executed on the Config type
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type RawConfig Config
	if err := unmarshal((*RawConfig)(c)); err != nil {
		return err
	}

	// Set a default LogDirectory
	if c.LogDirectory == "" {
		c.LogDirectory = "logs"
	}

	// Ensure a proper LogLevel was provided
	if c.LogLevel == "" {
		c.LogLevel = "info"
	} else {
		_, err := logrus.ParseLevel(c.LogLevel)
		if err != nil {
			return err
		}
	}

	// Set a default port for the HTTP listener
	if c.HTTPPort == 0 {
		c.HTTPPort = 8080
	}

	// Set a default port for the TCP listener
	if c.TCPPort == 0 {
		c.TCPPort = 6000
	}

	return nil
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

	return &config, nil
}