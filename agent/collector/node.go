package collector

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

/*
Constants
*/
const cassandraGCLogFolderName = "logs"

/*
Settings
*/
type NodeCollectorSettings struct {
	Cassandra  CassandraSettings  `yaml:"cassandra"`
	Collecting CollectingSettings `yaml:"collecting"`
}

type CassandraSettings struct {
	ConfigPath string   `yaml:"config-path"`
	LogPath    string   `yaml:"log-path"`
	HomePath   string   `yaml:"home-path"`
	DataPath   []string `yaml:"data-path"`
}

type CollectingSettings struct {
	Configs []string `yaml:"configs"`
	Logs    []string `yaml:"logs"`
}

func NodeCollectorDefaultSettings() *NodeCollectorSettings {
	return &NodeCollectorSettings{
		Cassandra: CassandraSettings{
			ConfigPath: "/etc/cassandra",
			LogPath:    "/var/log/cassandra",
			HomePath:   "/var/lib/cassandra",
			DataPath: []string{
				"/var/lib/cassandra/data",
			},
		},
		Collecting: CollectingSettings{
			Configs: []string{
				"cassandra.yaml",
				"cassandra-env.sh",
				"jvm.options",
				"logback.xml",
			},
			Logs: []string{
				"system.log",
			},
		},
	}
}

/*
Collector
*/
type NodeCollector struct {
	Settings *NodeCollectorSettings
	Logger   *logrus.Logger
	Path     string

	AppFs afero.Fs

	log *logrus.Entry
}

func (collector *NodeCollector) Collect(agent SSHCollectingAgent) error {

	log := collector.Logger.WithFields(logrus.Fields{
		"prefix": "NC " + agent.GetHost(),
	})
	collector.log = log
	log.Info("Node collector started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	InfoTaskCount := 3
	var wg sync.WaitGroup
	wg.Add(InfoTaskCount)

	go func() {
		defer wg.Done()

		log.Info("Collecting nodetool info...")
		err = collector.collectNodeToolInfo(agent)
		if err != nil {
			log.Error(err)
		}
		log.Info("Collecting nodetool info completed.")
	}()

	go func() {
		defer wg.Done()

		// TODO Hint "sudo apt install sysstat"
		log.Info("Collecting IO stats...")
		err = collector.collectIOStats(agent)
		if err != nil {
			log.Error(err)
		}
		log.Info("Collecting IO stats completed.")
	}()

	go func() {
		defer wg.Done()

		log.Info("Collecting disc info...")
		err = collector.collectDiscInfo(agent)
		if err != nil {
			log.Error(err)
		}
		log.Info("Collecting disc info completed.")
	}()

	log.Info("Collecting configuration files...")
	err = collector.collectConfigurationFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting configuration files completed.")

	log.Info("Collecting log files...")
	err = collector.collectLogFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting log files completed.")

	log.Info("Collecting gc log files...")
	err = collector.collectGCLogFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting gc log files completed.")

	wg.Wait()

	log.Info("Node collector completed")
	return nil
}

func (collector *NodeCollector) collectConfigurationFiles(agent SSHCollectingAgent) error {
	dest, err := collector.makeFolder(agent.GetHost(), "config")
	if err != nil {
		return err
	}

	for _, name := range collector.Settings.Collecting.Configs {
		src := filepath.Join(collector.Settings.Cassandra.ConfigPath, name)
		err = agent.ReceiveFile(src, dest)
		if err != nil {
			collector.log.Warn("Failed to receive config file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *NodeCollector) collectLogFiles(agent SSHCollectingAgent) error {
	dest, err := collector.makeFolder(agent.GetHost(), "logs")
	if err != nil {
		return err
	}

	for _, name := range collector.Settings.Collecting.Logs {
		src := filepath.Join(collector.Settings.Cassandra.LogPath, name)
		err = agent.ReceiveFile(src, dest)
		if err != nil {
			collector.log.Warn("Failed to receive log file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *NodeCollector) collectGCLogFiles(agent SSHCollectingAgent) error {
	dest, err := collector.makeFolder(agent.GetHost(), "gc_logs")
	if err != nil {
		return err
	}

	src := filepath.Join(collector.Settings.Cassandra.HomePath, cassandraGCLogFolderName)

	err = agent.ReceiveDir(src, dest, func(parentDir string, info os.FileInfo) (b bool, err error) {
		// TODO generate gc logs
		collector.log.Info("copy ", parentDir)
		return true, nil
	})
	if err != nil {
		collector.log.Warn("Failed to receive gc log files (" + err.Error() + ")")
	}

	return nil
}

func (collector *NodeCollector) collectNodeToolInfo(agent SSHCollectingAgent) error {
	commands := [...]string{
		"nodetool info",
		"nodetool version",
		"nodetool status",
		"nodetool tpstats",
		"nodetool compactionstats -H",
		"nodetool gossipinfo",
		"nodetool cfstats -H",
		"nodetool ring",
	}

	path, err := collector.makeFolder(agent.GetHost(), "info")
	if err != nil {
		return err
	}

	for _, command := range commands {
		sout, _, err := agent.ExecuteCommand(command)
		if err != nil {
			collector.log.Error("Failed to execute '" + command + "' (" + err.Error() + ")")
			continue
		}

		fileName := strings.ReplaceAll(command, " ", "_") + ".info"
		err = afero.WriteFile(collector.AppFs, filepath.Join(path, fileName), sout.Bytes(), os.ModePerm)
		if err != nil {
			collector.log.Error("Failed to save '" + command + "' data (" + err.Error() + ")")
			continue
		}
	}

	return nil
}

func (collector *NodeCollector) collectIOStats(agent SSHCollectingAgent) error {
	const command = "eval timeout -sHUP 60s iostat -x -m -t -y -z 30 < /dev/null"

	path, err := collector.makeFolder(agent.GetHost(), "info")
	if err != nil {
		return err
	}

	sout, _, err := agent.ExecuteCommand(command)
	if err != nil {
		// TODO Check if returned 124 status code
		//return errors.New("Failed to execute '" + command + "' (" + err.Error() + ")")
	}

	err = afero.WriteFile(collector.AppFs, filepath.Join(path, "io_stat.info"), sout.Bytes(), os.ModePerm)
	if err != nil {
		return errors.New("Failed to save iostate info (" + err.Error() + ")")
	}

	return nil
}

func (collector *NodeCollector) collectDiscInfo(agent SSHCollectingAgent) error {
	commands := [...]string{
		"df -h",
		"du -h",
	}

	path, err := collector.makeFolder(agent.GetHost(), "info")
	if err != nil {
		return err
	}

	var report bytes.Buffer

	for _, command := range commands {
		for _, dataPath := range collector.Settings.Cassandra.DataPath {
			command := fmt.Sprintf("%s %s", command, dataPath)

			sout, _, err := agent.ExecuteCommand(command)
			if err != nil {
				collector.log.Error("Failed to execute '" + command + "' (" + err.Error() + ")")
				continue
			}

			report.WriteString(command)
			report.WriteString(sout.String())
			report.WriteString("\n")
		}
	}

	err = afero.WriteFile(collector.AppFs, filepath.Join(path, "disk.info"), report.Bytes(), os.ModePerm)
	if err != nil {
		return errors.New("Failed to save disk info (" + err.Error() + ")")
	}

	return nil
}

func (collector *NodeCollector) makeFolder(host string, name string) (string, error) {
	path := filepath.Join(collector.Path, host, name)
	err := collector.AppFs.MkdirAll(path, os.ModePerm)
	if err != nil {
		return "", errors.New("Failed to create '" + name + "' folder (" + err.Error() + ")")
	}

	return path, nil
}
