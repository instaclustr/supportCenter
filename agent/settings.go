package main

import (
	"agent/collector"
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const defaultAgentHomePath = "~/.instaclustr/supportcenter"
const defaultProfileContainerName = "DEFAULT"

type AgentSettings struct {
	CollectedDataPath string `yaml:"collected-data-path"`
}

func AgentDefaultSettings() *AgentSettings {
	return &AgentSettings{
		CollectedDataPath: "~/.instaclustr/supportcenter/DATA",
	}
}

type TargetSettings struct {
	Nodes   []string `yaml:"nodes"`
	Metrics []string `yaml:"metrics"`
}

func TargetDefaultSettings() *TargetSettings {
	return &TargetSettings{
		Nodes:   []string{},
		Metrics: []string{},
	}
}

type Settings struct {
	Agent   AgentSettings                      `yaml:"agent"`
	Node    collector.NodeCollectorSettings    `yaml:"node"`
	Metrics collector.MetricsCollectorSettings `yaml:"metrics"`
	Target  TargetSettings                     `yaml:"target"`
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

func (settings *Settings) Save(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return errors.New("Failed to save settings file (" + err.Error() + ")")
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	err = encoder.Encode(settings)
	if err != nil {
		return errors.New("Failed to marshal settings file (" + err.Error() + ")")
	}

	return nil
}

func SearchSettingsPath(configPath string) string {

	// Config file defined
	if len(configPath) > 0 {
		return configPath
	}

	// Default profile
	profilePath := Expand(filepath.Join(defaultAgentHomePath, defaultProfileContainerName))
	exists, _ := Exists(profilePath)
	if exists == true {
		data, err := ioutil.ReadFile(profilePath)
		if err == nil {
			configName := strings.TrimSpace(string(data))
			if len(configName) > 0 {
				return filepath.Join(defaultAgentHomePath, configName)
			}
		}
	}

	// Default settings in the working dir
	return "settings.yml"
}
