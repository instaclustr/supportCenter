package collector

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
)

type SSHAgent struct {
	addr string

	config *ssh.ClientConfig
	client *ssh.Client
}

func (agent *SSHAgent) SetTarget(host string, port int) {
	agent.addr = fmt.Sprintf("%s:%d", host, port)
}

func (agent *SSHAgent) SetConfig(config *ssh.ClientConfig) {
	agent.config = config
}

func (agent *SSHAgent) Connect() error {
	client, err := ssh.Dial("tcp", agent.addr, agent.config)
	if err != nil {
		return errors.New("SSHAgent: Failed to establish connection to remote SSH '" + agent.addr + "'")
	}

	agent.client = client

	return nil
}

func (agent *SSHAgent) ExecuteCommand(cmd string) (*bytes.Buffer, *bytes.Buffer, error) {
	session, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, errors.New("SSHAgent: Failed to create SSH session to '" + agent.addr + "'")
	}
	defer session.Close()

	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer
	err = session.Run(cmd)
	if err != nil {
		return nil, nil, errors.New("SSHAgent: Failed to run command '" + cmd + "' csHost: " + agent.addr)
	}

	return &outBuffer, &errBuffer, nil
}
