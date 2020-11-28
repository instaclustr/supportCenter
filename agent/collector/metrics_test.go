package collector

import (
	"bytes"
	"errors"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const createSnapshotCommand = "curl -s -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot"
const createSnapshotsResponse = `
	{
	  "status": "success",
	  "data": {
		"name": "20200325T090812Z-78629a0f5f3f164f"
	  }
	}
`

const snapshotPath = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f"

var snapshotSubdirectoriesList = []string{
	"/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E444CMB0HSK01H0GSRE20NV1",
	"/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E444CNCYHACHCQPN2ERCGQPP",
	"/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E48F42Q6VHY4E8KBK02E7QE2",
	"/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E48FKDW67J37AEQ0N2S0ZBCZ",
}

const snapshotMeta1Path = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E444CMB0HSK01H0GSRE20NV1/meta.json"
const snapshotMeta1Content = `
	{
		"ulid": "01E444CMB0HSK01H0GSRE20NV1",
		"minTime": 1584957600000,
		"maxTime": 1584964800000,
		"stats": {
			"numSamples": 9352682,
			"numSeries": 7411,
			"numChunks": 74110
		},
		"compaction": {
			"level": 1,
			"sources": [
				"01E444CMB0HSK01H0GSRE20NV1"
			]
		},
		"version": 1
	}
`

const snapshotMeta2Path = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E444CNCYHACHCQPN2ERCGQPP/meta.json"
const snapshotMeta2Content = `
	{
		"ulid": "01E444CNCYHACHCQPN2ERCGQPP",
		"minTime": 1584964800000,
		"maxTime": 1584972000000,
		"stats": {
			"numSamples": 4698574,
			"numSeries": 7411,
			"numChunks": 44466
		},
		"compaction": {
			"level": 1,
			"sources": [
				"01E444CNCYHACHCQPN2ERCGQPP"
			]
		},
		"version": 1
	}
`

const snapshotMeta3Path = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E48F42Q6VHY4E8KBK02E7QE2/meta.json"
const snapshotMeta3Content = `
	{
		"ulid": "01E48F42Q6VHY4E8KBK02E7QE2",
		"minTime": 1584979200000,
		"maxTime": 1584986400000,
		"stats": {
			"numSamples": 20572936,
			"numSeries": 7411,
			"numChunks": 177864
		},
		"compaction": {
			"level": 1,
			"sources": [
				"01E48F42Q6VHY4E8KBK02E7QE2"
			]
		},
		"version": 1
	}
`

const snapshotMeta4Path = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f/01E48FKDW67J37AEQ0N2S0ZBCZ/meta.json"
const snapshotMeta4Content = `
	{
		"ulid": "01E48H0HDGMZRJD9BDF91VQ01Y",
		"minTime": 1585123200000,
		"maxTime": 1585129210806,
		"stats": {
			"numSamples": 14402529,
			"numSeries": 7263,
			"numChunks": 123471
		},
		"compaction": {
			"level": 1,
			"sources": [
				"01E48H0HDGMZRJD9BDF91VQ01Y"
			]
		},
		"version": 1
	}
`

const createTarballCommand = "tar -cf /tmp/InstaclustrCollection.tar -C /var/data/snapshots/20200325T090812Z-78629a0f5f3f164f ."
const removeSnapshotPath = "/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f"
const removeTarballPath = "/tmp/InstaclustrCollection.tar"

func TestMetricsCollector_Collect(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.On("GetHost").Return("metrics-test-host-1")
	mockedSSHAgent.On("Connect").Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", createSnapshotCommand).
		Return(bytes.NewBufferString(createSnapshotsResponse), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("ListDirectory", snapshotPath).
		Return(snapshotSubdirectoriesList, nil)

	mockedSSHAgent.
		On("GetContent", snapshotMeta1Path).
		Return(bytes.NewBufferString(snapshotMeta1Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta2Path).
		Return(bytes.NewBufferString(snapshotMeta2Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta3Path).
		Return(bytes.NewBufferString(snapshotMeta3Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta4Path).
		Return(bytes.NewBufferString(snapshotMeta4Content), nil)

	mockedSSHAgent.
		On("ExecuteCommand", createTarballCommand).
		Return(bytes.NewBufferString(""), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("Remove", removeSnapshotPath).
		Return(nil)

	mockedSSHAgent.
		On("ReceiveDir",
			"/tmp/InstaclustrCollection.tar", "/some/metrics/path/snapshot").
		Return(nil)

	mockedSSHAgent.
		On("Remove", removeTarballPath).
		Return(nil)

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_CollectOnCompressionDisabled(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.On("GetHost").Return("metrics-test-host-1")
	mockedSSHAgent.On("Connect").Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", createSnapshotCommand).
		Return(bytes.NewBufferString(createSnapshotsResponse), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("ListDirectory", snapshotPath).
		Return(snapshotSubdirectoriesList, nil)

	mockedSSHAgent.
		On("GetContent", snapshotMeta1Path).
		Return(bytes.NewBufferString(snapshotMeta1Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta2Path).
		Return(bytes.NewBufferString(snapshotMeta2Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta3Path).
		Return(bytes.NewBufferString(snapshotMeta3Content), nil)
	mockedSSHAgent.
		On("GetContent", snapshotMeta4Path).
		Return(bytes.NewBufferString(snapshotMeta4Content), nil)

	mockedSSHAgent.
		On("Remove", removeSnapshotPath).
		Return(nil)

	mockedSSHAgent.
		On("ReceiveDir",
			"/var/data/snapshots/20200325T090812Z-78629a0f5f3f164f", "/some/metrics/path/snapshot").
		Return(nil)

	logger, hook := test.NewNullLogger()

	metricsCollectorSettings := MetricsCollectorDefaultSettings()
	metricsCollectorSettings.CopyCompressed = false
	collector := MetricsCollector{
		Settings:      metricsCollectorSettings,
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_Collect_OnFailedToConnect(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("metrics-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(errors.New("SSH agent: Failed to establish connection to remote host 'Remote test' (some error)"))

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "SSH agent: Failed to establish connection to remote host 'Remote test' (some error)")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_Collect_OnFailedToCreateSnapshotByError(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("metrics-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", createSnapshotCommand).
		Return(bytes.NewBufferString(""), bytes.NewBufferString(""), errors.New("some test error"))

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "some test error")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_Collect_OnFailedToCreateSnapshotByStdoutMessage(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("metrics-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand",
			createSnapshotCommand).
		Return(bytes.NewBufferString(""), bytes.NewBufferString("we can not do that"), nil)

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "Failed to create prometheus snapshot: we can not do that")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_Collect_OnFailedToCreateSnapshotByInvalidResponse(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("metrics-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", createSnapshotCommand).
		Return(bytes.NewBufferString(`{ "xxx": "blablabla", sdfgsdf gsdfgsdfg } `), bytes.NewBufferString(""), nil)

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "Failed to unmarshal snapshot command output (invalid character 's' looking for beginning of object key string)")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestMetricsCollector_Collect_OnFailedToCreateSnapshotByInvalidStatus(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("metrics-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", createSnapshotCommand).
		Return(bytes.NewBufferString(`{ "xxx": "blablabla" } `), bytes.NewBufferString(""), nil)

	logger, hook := test.NewNullLogger()

	collector := MetricsCollector{
		Settings:      MetricsCollectorDefaultSettings(),
		Logger:        logger,
		Path:          "/some/metrics/path",
		TimestampFrom: time.Unix(0, 0).UTC(),
		TimestampTo:   time.Now().UTC(),
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "Failed to create prometheus snapshot (status:  '')")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}
