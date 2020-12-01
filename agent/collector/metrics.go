package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

/*
Constants
*/
const prometheusSnapshotSuccess = "success"
const prometheusSnapshotFolder = "snapshots"
const prometheusCreateSnapshotTemplate = "curl -s -XPOST http://localhost:%d/api/v1/admin/tsdb/snapshot"
const temporalSnapshotTarballPath = "/tmp/InstaclustrCollection.tar"
const createSnapshotTarballTemplate = "tar -cf %s -C %s ."
const snapshotMetadataFileName = "meta.json"

/*
Settings
*/
type MetricsCollectorSettings struct {
	Prometheus     PrometheusSettings `yaml:"prometheus"`
	CopyCompressed bool               `yaml:"copy_compressed"`
}

type PrometheusSettings struct {
	Port     int16  `yaml:"port"`
	DataPath string `yaml:"data-path"`
}

func MetricsCollectorDefaultSettings() *MetricsCollectorSettings {
	return &MetricsCollectorSettings{
		Prometheus: PrometheusSettings{
			Port:     9090,
			DataPath: "/var/data",
		},
		CopyCompressed: true,
	}
}

/*
Collector
*/
type MetricsCollector struct {
	Settings *MetricsCollectorSettings
	Logger   *logrus.Logger
	Path     string

	TimestampFrom time.Time
	TimestampTo   time.Time

	log *logrus.Entry
}

func (collector *MetricsCollector) Collect(agent SSHCollectingAgent) error {
	log := collector.Logger.WithFields(logrus.Fields{
		"prefix": "MC " + agent.GetHost(),
	})
	collector.log = log
	log.Info("Metrics collecting started")

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

	resourceName := "snapshot"
	src := filepath.Join(collector.Settings.Prometheus.DataPath, prometheusSnapshotFolder, snapshot)

	{
		log.Info("Lightening snapshot...")
		err := collector.lightenSnapshot(agent, src)
		if err != nil {
			log.Warn("Failed to lighten snapshot: " + err.Error())
		}
		log.Info("Lightening snapshot  OK")
	}

	if collector.Settings.CopyCompressed {
		log.Info("Creating snapshot tarball...")
		tarballErr := collector.tarballSnapshot(agent, src, temporalSnapshotTarballPath)
		if tarballErr != nil {
			log.Error(tarballErr)
		} else {
			log.Info("Creating snapshot tarball  OK")
		}

		log.Info("Cleanup snapshot...")
		err = collector.removeResource(agent, src)
		if err != nil {
			log.Error(err)
		} else {
			log.Info("Cleanup snapshot  OK")
		}

		if tarballErr != nil {
			return tarballErr
		}

		src = temporalSnapshotTarballPath
		resourceName = "snapshot tarball"
	}

	dest := filepath.Join(collector.Path, "snapshot")

	log.Info("Downloading snapshot...")
	err = collector.downloadSnapshot(agent, src, dest)
	if err != nil {
		log.Error(err)
	} else {
		log.Info("Downloading snapshot  OK")
	}

	log.Info(fmt.Sprint("Cleanup ", resourceName, "..."))
	err = collector.removeResource(agent, src)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(fmt.Sprint("Cleanup ", resourceName, "  OK"))

	log.Info("Metrics collecting completed")
	return nil
}

func (collector *MetricsCollector) createSnapshot(agent SSHCollectingAgent) (string, error) {
	createSnapshotCommand := fmt.Sprintf(prometheusCreateSnapshotTemplate, collector.Settings.Prometheus.Port)
	sout, serr, err := agent.ExecuteCommand(createSnapshotCommand)
	if err != nil {
		return "", err
	}
	if serr.Len() > 0 {
		return "", errors.New("Failed to create prometheus snapshot: " + serr.String())
	}

	type PrometheusSnapshotResponse struct {
		Status string
		Data   struct {
			Name string
		}
		Error string
	}

	var response PrometheusSnapshotResponse
	err = json.Unmarshal(sout.Bytes(), &response)
	if err != nil {
		return "", errors.New("Failed to unmarshal snapshot command output (" + err.Error() + ")")
	}

	if response.Status != prometheusSnapshotSuccess {
		return "", errors.New("Failed to create prometheus snapshot (status: " + response.Status + " '" + response.Error + "')")
	}

	return response.Data.Name, nil
}

func (collector *MetricsCollector) lightenSnapshot(agent SSHCollectingAgent, src string) error {

	blocks, err := getBlockList(agent, src)
	if err != nil {
		return err
	}

	for index, block := range blocks {
		metadata, err := getBlockMetadata(agent, block)
		if err != nil {
			collector.Logger.Warn("Ignoring block (" + block + "): " + err.Error())
			continue
		}

		if metadata.Version != 1 {
			collector.Logger.Warn("Ignoring block (", block, "): version #", metadata.Version, " unsupported")
			continue
		}

		blockMinTimestamp := time.Unix(metadata.MinTime/int64(1000), (metadata.MinTime%int64(1000))*int64(1000000)).UTC()
		blockMaxTimestamp := time.Unix(metadata.MaxTime/int64(1000), (metadata.MaxTime%int64(1000))*int64(1000000)).UTC()

		fallsIntoTheSelectedTimeRange := false
		logMessage := "will be skipped"

		if (blockMinTimestamp.After(collector.TimestampFrom) || blockMaxTimestamp.After(collector.TimestampFrom)) &&
			(blockMinTimestamp.Before(collector.TimestampTo) || blockMaxTimestamp.Before(collector.TimestampTo)) {
			fallsIntoTheSelectedTimeRange = true
			logMessage = "falls into the time span"
		}

		collector.Logger.Info("Block ", index+1, "/", len(blocks), " ", metadata.Ulid, "  ", blockMinTimestamp, " .. ", blockMaxTimestamp, ": ", logMessage)

		if !fallsIntoTheSelectedTimeRange {
			err := collector.removeResource(agent, block)
			if err != nil {
				collector.Logger.Warn("Failed to drop snapshot block: " + err.Error())
			}
		}
	}

	return nil
}

func getBlockList(agent SSHCollectingAgent, src string) ([]string, error) {

	entries, err := agent.ListDirectory(src)
	if err != nil {
		return nil, errors.New("Failed to get block list of prometheus snapshot: " + err.Error())
	}

	directories := make([]string, 0)
	for _, entry := range entries {
		if entry.IdDir {
			directories = append(directories, entry.Path)
		}
	}

	return directories, nil
}

type blockMetadata struct {
	Ulid    string
	Version int
	MinTime int64
	MaxTime int64
	Stats   struct {
		NumSamples uint64
		NumSeries  uint64
		NumChunks  uint64
	}
}

func getBlockMetadata(agent SSHCollectingAgent, path string) (*blockMetadata, error) {
	content, err := agent.GetContent(filepath.Join(path, snapshotMetadataFileName))
	if err != nil {
		return nil, errors.New("Failed to get block metadata (" + err.Error() + ")")
	}

	var metadata blockMetadata
	err = json.Unmarshal(content.Bytes(), &metadata)
	if err != nil {
		return nil, errors.New("Failed to unmarshal block metadata (" + err.Error() + ")")
	}

	return &metadata, nil
}

func (collector *MetricsCollector) tarballSnapshot(agent SSHCollectingAgent, src string, dest string) error {
	createTarballCommand := fmt.Sprintf(createSnapshotTarballTemplate, dest, src)
	_, serr, err := agent.ExecuteCommand(createTarballCommand)
	if err != nil {
		return err
	}
	if serr.Len() > 0 {
		return errors.New("Failed to create snapshot tarball: " + serr.String())
	}

	return nil
}

func (collector *MetricsCollector) downloadSnapshot(agent SSHCollectingAgent, src string, dest string) error {
	err := agent.ReceiveDir(src, dest, func(copied int64, size int64, remaining time.Duration) {
		collector.log.Info("Downloading snapshot ", HumanSize(float64(copied)), " of ", HumanSize(float64(size)),
			" (remaining ", remaining.Round(time.Second), ") ...")
	})
	if err != nil {
		return errors.New("Failed to receive snapshot (" + err.Error() + ")")
	}

	return nil
}

func (collector *MetricsCollector) removeResource(agent SSHCollectingAgent, path string) error {
	err := agent.Remove(path)
	if err != nil {
		return errors.New("Failed to remove resource '" + path + "' (" + err.Error() + ")")
	}

	return nil
}
