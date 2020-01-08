package main

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type Settings struct {
	Stats `yaml:"stats"`
}

type Stats struct {
	Prometheus `yaml:"prometheus"`
}

type Prometheus struct {
	Port     int16  `yaml:"port"`
	DataPath string `yaml:"data-path"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Stats: Stats{Prometheus{
			Port:     9090,
			DataPath: "/var/data",
		}},
	}
}

func (settings *Settings) Load(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return errors.New("Failed to load settings file (" + err.Error() + ")")
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(settings)
	if err != nil {
		return errors.New("Failed to unmarshal settings file (" + err.Error() + ")")
	}

	return nil
}
