package collector

import (
	"bytes"
	"github.com/hnakamur/go-scp"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/ssh"
)

type mockedSSHAgentObject struct {
	mock.Mock
}

func (m *mockedSSHAgentObject) SetTarget(host string, port int) {
	panic("implement me")
}

func (m *mockedSSHAgentObject) SetConfig(config *ssh.ClientConfig) {
	panic("implement me")
}

func (m *mockedSSHAgentObject) GetHost() string {
	arguments := m.Called()
	return arguments.String(0)
}

func (m *mockedSSHAgentObject) Connect() error {
	arguments := m.Called()
	return arguments.Error(0)

}

func (m *mockedSSHAgentObject) ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error) {
	ret := m.Called(cmd)
	return ret.Get(0).(*bytes.Buffer), ret.Get(1).(*bytes.Buffer), ret.Error(2)
}

func (m *mockedSSHAgentObject) ReceiveFile(src, dest string) error {
	ret := m.Called(src, dest)
	return ret.Error(0)
}

func (m *mockedSSHAgentObject) ReceiveDir(src, dest string, acceptFn scp.AcceptFunc) error {
	ret := m.Called(src, dest, acceptFn)
	return ret.Error(0)
}
