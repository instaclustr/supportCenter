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
	"time"
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
	GCPath     string   `yaml:"gc-path"`
	DataPath   []string `yaml:"data-path"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
}

type CollectingSettings struct {
	Configs       []string `yaml:"configs"`
	Logs          []string `yaml:"logs"`
	GCLogPatterns []string `yaml:"gc-log-patterns"`
}

func NodeCollectorDefaultSettings() *NodeCollectorSettings {
	return &NodeCollectorSettings{
		Cassandra: CassandraSettings{
			ConfigPath: "/etc/cassandra",
			LogPath:    "/var/log/cassandra",
			GCPath:     "/var/log/cassandra",
			DataPath: []string{
				"/var/lib/cassandra/data",
			},
			Username: "",
			Password: "",
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
			GCLogPatterns: []string{
				"gc*",
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

	InfoTaskCount := 4
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

	go func() {
		defer wg.Done()

		log.Info("Collecting system info...")
		err = collector.collectSystemInfo(agent)
		if err != nil {
			log.Error(err)
		}
		log.Info("Collecting system info completed.")
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
		err = agent.ReceiveFile(src, dest, nil)
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
		err = agent.ReceiveFile(src, dest, func(copied int64, size int64, remaining time.Duration) {
			collector.log.Info("Downloading '", name, "' log file ",
				HumanSize(float64(copied)), " of ", HumanSize(float64(size)),
				" (remaining ", remaining.Round(time.Second), ") ...")
		})
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

	entries, err := agent.ListDirectory(collector.Settings.Cassandra.GCPath)
	if err != nil {
		return errors.New("Failed to check GC log directory (" + err.Error() + ")")
	}

	for _, entry := range entries {
		if entry.IdDir {
			continue
		}

		for _, pattern := range collector.Settings.Collecting.GCLogPatterns {
			filename := filepath.Base(entry.Path)
			match, err := filepath.Match(pattern, filename)
			if err != nil {
				collector.log.Warn("Failed to check GC log '" + entry.Path + " pattern  '" + pattern +
					"' matching ' (" + err.Error() + ")")
			}

			if match == true {
				err := agent.ReceiveFile(entry.Path, dest, func(copied int64, size int64, remaining time.Duration) {
					collector.log.Info("Downloading '", filename, "' GC log file ",
						HumanSize(float64(copied)), " of ", HumanSize(float64(size)),
						" (remaining ", remaining.Round(time.Second), ") ...")
				})
				if err != nil {
					collector.log.Warn("Failed to receive GC log file (" + err.Error() + ")")
				}
			}
		}
	}

	return nil
}

func (collector *NodeCollector) collectNodeToolInfo(agent SSHCollectingAgent) error {
	commands := [...]string{
		"info",
		"version",
		"status",
		"tpstats",
		"compactionstats -H",
		"gossipinfo",
		"cfstats -H",
		"ring",
	}

	path, err := collector.makeFolder(agent.GetHost(), "info")
	if err != nil {
		return err
	}

	for _, command := range commands {
		var args = strings.Builder{}
		args.WriteString("nodetool ")
		if len(collector.Settings.Cassandra.Username) > 0 {
			fmt.Fprintf(&args, "-u '%s' ", collector.Settings.Cassandra.Username)
		}
		if len(collector.Settings.Cassandra.Password) > 0 {
			fmt.Fprintf(&args, "-pw '%s' ", collector.Settings.Cassandra.Password)
		}
		args.WriteString(command)
		sout, _, err := agent.ExecuteCommand(args.String())
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

func (collector *NodeCollector) collectSystemInfo(agent SSHCollectingAgent) error {
	commands := [...]string{
		"ulimit -a",
		"free -m",
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

func (collector *NodeCollector) makeFolder(host string, name string) (string, error) {
	path := filepath.Join(collector.Path, host, name)
	err := collector.AppFs.MkdirAll(path, os.ModePerm)
	if err != nil {
		return "", errors.New("Failed to create '" + name + "' folder (" + err.Error() + ")")
	}

	return path, nil
}
