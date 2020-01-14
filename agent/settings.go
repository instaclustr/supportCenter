package main

import (
	"agent/collector"
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type Settings struct {
	Node    collector.NodeCollectorSettings    `yaml:"node"`
	Metrics collector.MetricsCollectorSettings `yaml:"metrics"`
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
