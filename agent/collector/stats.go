package collector

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const PrometheusSnapshotSuccess = "success"

var scLogger = logrus.New()

func init() {
	scLogger.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

func CollectStats(agent *SSHAgent) error {
	log := scLogger.WithFields(logrus.Fields{
		"prefix": "SC " + agent.addr,
	})
	log.Info("Stats collecting started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	snapshot, err := createSnapshot(agent)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Prometheus snapshot name: ", snapshot)

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
