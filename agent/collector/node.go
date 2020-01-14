package collector

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hnakamur/go-scp"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

/*
Constants
*/
const cassandraGCLogFolderName = "logs"
const perm os.FileMode = 0755

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

	log *logrus.Entry
}

func (collector *NodeCollector) Collect(agent *SSHAgent) error {
	log := collector.Logger.WithFields(logrus.Fields{
		"prefix": "NC " + agent.host,
	})
	collector.log = log
	log.Info("Node collector started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Collecting nodetool info...")
	err = collector.collectNodeToolInfo(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting nodetool info completed.")

	// TODO Hint "sudo apt install sysstat"
	log.Info("Collecting IO stats...")
	err = collector.collectIOStats(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting IO stats completed.")

	log.Info("Collecting disc info...")
	err = collector.collectDiscInfo(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting disc info completed.")

	log.Info("Collecting configuration files...")
	err = collector.downloadConfigurationFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting configuration files completed.")

	log.Info("Collecting log files...")
	err = collector.downloadLogFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting log files completed.")

	log.Info("Collecting gc log files...")
	err = collector.downloadGCLogFiles(agent)
	if err != nil {
		log.Error(err)
	}
	log.Info("Collecting gc log files completed.")

	log.Info("Node collector completed")
	return nil
}

func (collector *NodeCollector) downloadConfigurationFiles(agent *SSHAgent) error {
	dest := filepath.Join(collector.Path, agent.host, "config")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for configs (" + dest + ")")
	}

	for _, name := range collector.Settings.Collecting.Configs {
		src := filepath.Join(collector.Settings.Cassandra.ConfigPath, name)
		scpAgent := scp.NewSCP(agent.client)
		err = scpAgent.ReceiveFile(src, dest)
		if err != nil {
			collector.log.Warn("Failed to receive config file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *NodeCollector) downloadLogFiles(agent *SSHAgent) error {
	dest := filepath.Join(collector.Path, agent.host, "logs")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for logs (" + dest + ")")
	}

	for _, name := range collector.Settings.Collecting.Logs {
		src := filepath.Join(collector.Settings.Cassandra.LogPath, name)
		scpAgent := scp.NewSCP(agent.client)
		err = scpAgent.ReceiveFile(src, dest)
		if err != nil {
			collector.log.Warn("Failed to receive log file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *NodeCollector) downloadGCLogFiles(agent *SSHAgent) error {
	dest := filepath.Join(collector.Path, agent.host, "gc_logs")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for logs (" + dest + ")")
	}

	src := filepath.Join(collector.Settings.Cassandra.HomePath, cassandraGCLogFolderName)

	scpAgent := scp.NewSCP(agent.client)
	err = scpAgent.ReceiveDir(src, dest, func(parentDir string, info os.FileInfo) (b bool, err error) {
		// TODO generate gc logs
		collector.log.Info("copy ", parentDir)
		return true, nil
	})
	if err != nil {
		collector.log.Warn("Failed to receive gc log files (" + err.Error() + ")")
	}

	return nil
}

func (collector *NodeCollector) collectNodeToolInfo(agent *SSHAgent) error {
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

	path, err := getInfoFolder(collector.Path, agent.host)
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
		err = ioutil.WriteFile(filepath.Join(path, fileName), sout.Bytes(), perm)
		if err != nil {
			collector.log.Error("Failed to save '" + command + "' data (" + err.Error() + ")")
			continue
		}
	}

	return nil
}

// TODO Investigate (Process exited with status 124)
func (collector *NodeCollector) collectIOStats(agent *SSHAgent) error {
	const command = "eval timeout -sHUP 60s iostat -x -m -t -y -z 30 < /dev/null"

	path, err := getInfoFolder(collector.Path, agent.host)
	if err != nil {
		return err
	}

	sout, serr, err := agent.ExecuteCommand(command)
	collector.log.Warn(sout)
	collector.log.Warn(serr)
	if err != nil {
		return errors.New("Failed to execute '" + command + "' (" + err.Error() + ")")
	}

	err = ioutil.WriteFile(filepath.Join(path, "io_stat.info"), sout.Bytes(), perm)
	if err != nil {
		return errors.New("Failed to save iostate info (" + err.Error() + ")")
	}

	return nil
}

func (collector *NodeCollector) collectDiscInfo(agent *SSHAgent) error {
	commands := [...]string{
		"df -h",
		"du -h",
	}

	path, err := getInfoFolder(collector.Path, agent.host)
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

	err = ioutil.WriteFile(filepath.Join(path, "disk.info"), report.Bytes(), perm)
	if err != nil {
		return errors.New("Failed to save disk info (" + err.Error() + ")")
	}

	return nil
}

func getInfoFolder(root string, host string) (string, error) {
	path := filepath.Join(root, host, "info")
	err := os.MkdirAll(path, perm)
	if err != nil {
		return "", errors.New("Failed to create info folder (" + err.Error() + ")")
	}

	return path, nil
}
