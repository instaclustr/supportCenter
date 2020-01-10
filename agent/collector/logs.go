package collector

import (
	"errors"
	"github.com/hnakamur/go-scp"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

/*
Constants
*/
const cassandraGCLogFolderName = "logs"

/*
Settings
*/
type LogsCollectorSettings struct {
	Cassandra  CassandraSettings  `yaml:"cassandra"`
	Collecting CollectingSettings `yaml:"collecting"`
}

type CassandraSettings struct {
	ConfigPath string `yaml:"config-path"`
	LogPath    string `yaml:"log-path"`
	HomePath   string `yaml:"home-path"`
}

type CollectingSettings struct {
	Configs []string `yaml:"configs"`
	Logs    []string `yaml:"logs"`
}

func LogsCollectorDefaultSettings() *LogsCollectorSettings {
	return &LogsCollectorSettings{
		Cassandra: CassandraSettings{
			ConfigPath: "/etc/cassandra",
			LogPath:    "/var/log/cassandra",
			HomePath:   "/var/lib/cassandra",
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
type LogsCollector struct {
	Settings *LogsCollectorSettings
	Log      *logrus.Logger
}

func (collector *LogsCollector) Collect(agent *SSHAgent) error {
	log := collector.Log.WithFields(logrus.Fields{
		"prefix": "LC " + agent.host,
	})
	log.Info("Logs collecting started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Downloading configuration files...")
	err = collector.downloadConfigurationFiles(agent, log)
	if err != nil {
		log.Error(err)
	}
	log.Info("Downloading configuration files  OK")

	log.Info("Downloading log files...")
	err = collector.downloadLogFiles(agent, log)
	if err != nil {
		log.Error(err)
	}
	log.Info("Downloading log files  OK")

	log.Info("Downloading gc log files...")
	err = collector.downloadGCLogFiles(agent, log)
	if err != nil {
		log.Error(err)
	}
	log.Info("Downloading gc log files  OK")

	log.Info("Logs collecting completed")
	return nil
}

func (collector *LogsCollector) downloadConfigurationFiles(agent *SSHAgent, log *logrus.Entry) error {
	// TODO add timestamp
	dest := filepath.Join("./data", agent.host, "config")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for configs (" + dest + ")")
	}

	for _, name := range collector.Settings.Collecting.Configs {
		src := filepath.Join(collector.Settings.Cassandra.ConfigPath, name)
		scpAgent := scp.NewSCP(agent.client)
		err = scpAgent.ReceiveFile(src, dest)
		if err != nil {
			log.Warn("Failed to receive config file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *LogsCollector) downloadLogFiles(agent *SSHAgent, log *logrus.Entry) error {
	// TODO add timestamp
	dest := filepath.Join("./data", agent.host, "log")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for logs (" + dest + ")")
	}

	for _, name := range collector.Settings.Collecting.Logs {
		src := filepath.Join(collector.Settings.Cassandra.LogPath, name)
		scpAgent := scp.NewSCP(agent.client)
		err = scpAgent.ReceiveFile(src, dest)
		if err != nil {
			log.Warn("Failed to receive log file '" + src + "' (" + err.Error() + ")")
		}
	}

	return nil
}

func (collector *LogsCollector) downloadGCLogFiles(agent *SSHAgent, log *logrus.Entry) error {
	// TODO add timestamp
	dest := filepath.Join("./data", agent.host, "gc")
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return errors.New("Failed to create folder for logs (" + dest + ")")
	}

	src := filepath.Join(collector.Settings.Cassandra.HomePath, cassandraGCLogFolderName)

	scpAgent := scp.NewSCP(agent.client)
	err = scpAgent.ReceiveDir(src, dest, func(parentDir string, info os.FileInfo) (b bool, err error) {
		log.Info("copy ", parentDir)
		return true, nil
	})
	if err != nil {
		log.Warn("Failed to receive gc log files (" + err.Error() + ")")
	}

	return nil
}
