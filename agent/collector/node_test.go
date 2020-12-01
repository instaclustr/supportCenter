package collector

import (
	"bytes"
	"errors"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

const collectNodeToolInfoCommand = "nodetool info"
const collectNodeToolVersionCommand = "nodetool version"
const collectNodeToolStatusCommand = "nodetool status"
const collectNodeToolTpstatsCommand = "nodetool tpstats"
const collectNodeToolCompactionstatsCommand = "nodetool compactionstats -H"
const collectNodeToolGossipinfoCommand = "nodetool gossipinfo"
const collectNodeToolCfstatsCommand = "nodetool cfstats -H"
const collectNodeToolRingCommand = "nodetool ring"

const collectIOStatsCommand = "eval timeout -sHUP 60s iostat -x -m -t -y -z 30 < /dev/null"

const collectDiscInfo1Command = "df -h /var/lib/cassandra/data"
const collectDiscInfo2Command = "du -h /var/lib/cassandra/data"

var gcLogs = []FileInfo{
	{"/var/log/cassandra/system.log", false},
	{"/var/log/cassandra/gc.log.2", false},
	{"/var/log/cassandra/gc.log.0", false},
	{"/var/log/cassandra/gc.log.3.current", false},
	{"/var/log/cassandra/gc.log.1", false},
}

func TestNodeCollector_Collect(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.On("GetHost").Return("node-test-host-1")
	mockedSSHAgent.On("Connect").Return(nil)

	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolInfoCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolVersionCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolStatusCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolTpstatsCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolCompactionstatsCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolGossipinfoCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolCfstatsCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectNodeToolRingCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("ExecuteCommand", collectIOStatsCommand).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("ExecuteCommand", collectDiscInfo1Command).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectDiscInfo2Command).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)

	mockedSSHAgent.
		On("ReceiveFile",
			"/etc/cassandra/cassandra.yaml", "some/path/node-test-host-1/config", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/etc/cassandra/cassandra-env.sh", "some/path/node-test-host-1/config", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/etc/cassandra/jvm.options", "some/path/node-test-host-1/config", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/etc/cassandra/logback.xml", "some/path/node-test-host-1/config", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/var/log/cassandra/system.log", "some/path/node-test-host-1/logs", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)

	mockedSSHAgent.
		On("ListDirectory", "/var/log/cassandra").
		Return(gcLogs, nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/var/log/cassandra/gc.log.2", "some/path/node-test-host-1/gc_logs", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/var/log/cassandra/gc.log.0", "some/path/node-test-host-1/gc_logs", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/var/log/cassandra/gc.log.3.current", "some/path/node-test-host-1/gc_logs", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)
	mockedSSHAgent.
		On("ReceiveFile",
			"/var/log/cassandra/gc.log.1", "some/path/node-test-host-1/gc_logs", mock.AnythingOfType("collector.ProgressFunc")).
		Return(nil)

	logger, hook := test.NewNullLogger()

	collector := NodeCollector{
		Settings: NodeCollectorDefaultSettings(),
		Logger:   logger,
		Path:     "some/path",
		AppFs:    afero.NewMemMapFs(),
	}

	err := collector.Collect(mockedSSHAgent)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestNodeCollector_Collect_OnFailedToConnect(t *testing.T) {

	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("node-test-host-1")
	mockedSSHAgent.
		On("Connect").
		Return(errors.New("SSH agent: Failed to establish connection to remote host 'Remote test' (some error)"))

	logger, hook := test.NewNullLogger()

	collector := NodeCollector{
		Settings: NodeCollectorDefaultSettings(),
		Logger:   logger,
		Path:     "some/path",
	}

	err := collector.Collect(mockedSSHAgent)
	if assert.Error(t, err) {
		assert.EqualError(t, err, "SSH agent: Failed to establish connection to remote host 'Remote test' (some error)")
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}

func TestNodeCollector_collectDiscInfo(t *testing.T) {
	mockedSSHAgent := new(mockedSSHAgentObject)
	mockedSSHAgent.
		On("GetHost").
		Return("node-test-host-1")
	mockedSSHAgent.
		On("ExecuteCommand", collectDiscInfo1Command).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)
	mockedSSHAgent.
		On("ExecuteCommand", collectDiscInfo2Command).
		Return(bytes.NewBufferString("some data"), bytes.NewBufferString(""), nil)

	logger, hook := test.NewNullLogger()

	collector := NodeCollector{
		Settings: NodeCollectorDefaultSettings(),
		Logger:   logger,
		Path:     "some/path",
		AppFs:    afero.NewMemMapFs(),
	}

	err := collector.collectDiscInfo(mockedSSHAgent)
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	mockedSSHAgent.AssertExpectations(t)

	hook.Reset()
}
