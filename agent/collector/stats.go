package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hnakamur/go-scp"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

/*
Constants
*/
const prometheusSnapshotSuccess = "success"
const prometheusSnapshotFolder = "snapshots"
const prometheusCreateSnapshotTemplate = "curl -s -XPOST http://localhost:%d/api/v1/admin/tsdb/snapshot"
const prometheusRemoveSnapshotTemplate = "rm -rf %s"

/*
Settings
*/
type StatsCollectorSettings struct {
	Prometheus PrometheusSettings `yaml:"prometheus"`
}

type PrometheusSettings struct {
	Port     int16  `yaml:"port"`
	DataPath string `yaml:"data-path"`
}

func StatsCollectorDefaultSettings() *StatsCollectorSettings {
	return &StatsCollectorSettings{
		Prometheus: PrometheusSettings{
			Port:     9090,
			DataPath: "/var/data",
		},
	}
}

/*
Collector
*/
type StatsCollector struct {
	Settings *StatsCollectorSettings
	Log      *logrus.Logger
	Path     string
}

func (collector *StatsCollector) Collect(agent *SSHAgent) error {
	log := collector.Log.WithFields(logrus.Fields{
		"prefix": "SC " + agent.host,
	})
	log.Info("Stats collecting started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Creating snapshot...")
	snapshot, err := collector.createSnapshot(agent)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Creating snapshot  OK")
	log.Info("Snapshot name: ", snapshot)

	src := filepath.Join(collector.Settings.Prometheus.DataPath, prometheusSnapshotFolder, snapshot)
	dest := filepath.Join(collector.Path, agent.host, "/snapshot")

	log.Info("Downloading snapshot...")
	err = collector.downloadSnapshot(agent, src, dest)
	if err != nil {

		log.Error(err)
		return err
	}
	log.Info("Downloading snapshot  OK")

	log.Info("Cleanup snapshot...")
	err = collector.removeSnapshot(agent, src)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Cleanup snapshot  OK")

	log.Info("Stats collecting completed")
	return nil
}

func (collector *StatsCollector) createSnapshot(agent *SSHAgent) (string, error) {
	createSnapshotCommand := fmt.Sprintf(prometheusCreateSnapshotTemplate, collector.Settings.Prometheus.Port)
	sout, serr, err := agent.ExecuteCommand(createSnapshotCommand)
	if err != nil {
		return "", err
	}
	if serr.Len() > 0 {
		return "", errors.New("Wrong output on creating snapshot: " + serr.String())
	}

	type PrometheusSnapshotResponse struct {
		Status string
		Data   struct {
			Name string
		}
	}

	var response PrometheusSnapshotResponse
	err = json.Unmarshal(sout.Bytes(), &response)
	if err != nil {
		return "", errors.New("Failed to unmarshal snapshot command output (" + err.Error() + ")")
	}

	if response.Status != prometheusSnapshotSuccess {
		return "", errors.New("Failed to create prometheus snapshot (status: " + response.Status + ")")
	}

	return response.Data.Name, nil
}

func (collector *StatsCollector) downloadSnapshot(agent *SSHAgent, src string, dest string) error {
	scpAgent := scp.NewSCP(agent.client)
	err := scpAgent.ReceiveDir(src, dest, nil)

	if err != nil {
		return errors.New("Failed to receive snapshot (" + err.Error() + ")")
	}

	return nil
}

func (collector *StatsCollector) removeSnapshot(agent *SSHAgent, path string) error {
	_, serr, err := agent.ExecuteCommand(fmt.Sprintf(prometheusRemoveSnapshotTemplate, path))

	if err != nil {
		return errors.New("Failed to remove snapshot (" + err.Error() + ")")
	}

	if serr.Len() > 0 {
		return errors.New("Wrong output on creating snapshot: " + serr.String())
	}

	return nil
}
