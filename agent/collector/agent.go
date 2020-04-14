package collector

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hnakamur/go-scp"
	"golang.org/x/crypto/ssh"
)

type SSHCollectingAgent interface {
	SetTarget(host string, port int)
	SetConfig(config *ssh.ClientConfig)

	GetHost() string

	Connect() error
	ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error)

	ReceiveFile(src, dest string) error
	ReceiveDir(src, dest string, acceptFn scp.AcceptFunc) error
}

type SSHAgent struct {
	host string
	addr string

	config *ssh.ClientConfig
	client *ssh.Client
}

func (agent *SSHAgent) SetTarget(host string, port int) {
	agent.host = host
	agent.addr = fmt.Sprintf("%s:%d", host, port)
}

func (agent *SSHAgent) SetConfig(config *ssh.ClientConfig) {
	agent.config = config
}

func (agent *SSHAgent) GetHost() string {
	return agent.host
}

func (agent *SSHAgent) Connect() error {
	client, err := ssh.Dial("tcp", agent.addr, agent.config)
	if err != nil {
		return errors.New("SSH agent: Failed to establish connection to remote host '" + agent.host + "' (" + err.Error() + ")")
	}

	agent.client = client

	return nil
}

func (agent *SSHAgent) ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error) {
	session, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, errors.New("SSH agent: Failed to create SSH session to '" + agent.host + "'")
	}
	defer session.Close()

	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer
	err = session.Run(cmd)
	if err != nil {
		return &outBuffer, &errBuffer, errors.New("SSH agent: Failed to run command '" + cmd + "' on '" + agent.host + "'. (" + err.Error() + ")")
	}

	return &outBuffer, &errBuffer, nil
}

func (agent *SSHAgent) ReceiveFile(src, dest string) error {
	scpAgent := scp.NewSCP(agent.client)
	return scpAgent.ReceiveFile(src, dest)
}

func (agent *SSHAgent) ReceiveDir(src, dest string, acceptFn scp.AcceptFunc) error {
	scpAgent := scp.NewSCP(agent.client)
	return scpAgent.ReceiveDir(src, dest, acceptFn)
}
