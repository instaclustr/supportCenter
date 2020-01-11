package collector

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
)

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
		return nil, nil, errors.New("SSH agent: Failed to run command '" + cmd + "' on '" + agent.host + "'. (" + err.Error() + ")")
	}

	return &outBuffer, &errBuffer, nil
}
