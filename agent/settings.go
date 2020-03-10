package main

import (
	"agent/collector"
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type AgentSettings struct {
	CollectedDataPath string `yaml:"collected-data-path"`
}

func AgentDefaultSettings() *AgentSettings {
	return &AgentSettings{
		CollectedDataPath: "~/.instaclustr/supportcenter/DATA",
	}
}

type Settings struct {
	Agent   AgentSettings                      `yaml:"agent"`
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
