package collector

import (
	"encoding/json"
	"errors"
	"github.com/hnakamur/go-scp"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

/*
Constants
*/
const PrometheusSnapshotSuccess = "success"

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

var scLogger = logrus.New()

func init() {
	scLogger.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

func CollectStats(agent *SSHAgent) error {
	log := scLogger.WithFields(logrus.Fields{
		"prefix": "SC " + agent.host,
	})
	log.Info("Stats collecting started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Creating snapshot...")
	snapshot, err := createSnapshot(agent)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Creating snapshot  OK")
	log.Info("Prometheus snapshot name: ", snapshot)

	// TODO move to settings
	prometheusPath := "/home/serhii/prometheus-2.15.1.linux-amd64"
	snapshotPath := prometheusPath + "/data/snapshots/" + snapshot
	// TODO add timestamp
	destinationPath := "./data/" + agent.host + "/snapshot"

	log.Info("Downloading snapshot...")
	err = downloadSnapshot(agent, snapshotPath, destinationPath)
	if err != nil {

		log.Error(err)
		return err
	}
	log.Info("Downloading snapshot  OK")

	log.Info("Cleanup snapshot...")
	err = removeSnapshot(agent, snapshotPath)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("Cleanup snapshot  OK")

	log.Info("Stats collecting completed")
	return nil
}

func createSnapshot(agent *SSHAgent) (string, error) {
	// TODO move port to options
	sout, serr, err := agent.ExecuteCommand("curl -s -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot")
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

	if response.Status != PrometheusSnapshotSuccess {
		return "", errors.New("Failed to create prometheus snapshot (status: " + response.Status + ")")
	}

	return response.Data.Name, nil
}

func downloadSnapshot(agent *SSHAgent, src string, dest string) error {
	scpAgent := scp.NewSCP(agent.client)
	err := scpAgent.ReceiveDir(src, dest, nil)

	if err != nil {
		return errors.New("Failed to receive snapshot (" + err.Error() + ")")
	}

	return nil
}

func removeSnapshot(agent *SSHAgent, path string) error {
	_, serr, err := agent.ExecuteCommand("rm -rf " + path)

	if err != nil {
		return errors.New("Failed to remove snapshot (" + err.Error() + ")")
	}

	if serr.Len() > 0 {
		return errors.New("Wrong output on creating snapshot: " + serr.String())
	}

	return nil
}
