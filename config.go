package main

import (
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Config defines a struct to match a configuration yaml file.
type Config struct {
	Address 			string 		`yaml:"Address"`
	LogDirectory 		string 		`yaml:"LogDirectory"`
	LogJSON 			bool		`yaml:"LogJSON"`
	LogLevel 			string 		`yaml:"LogLevel"`
	Port 				int 		`yaml:"Port"`
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
	_, err := log.ParseLevel(c.LogLevel)
	if err != nil {
		return err
	}

	// Set a default port
	//if c.Port == 0 {
	//	c.Port = 6000
	//}

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