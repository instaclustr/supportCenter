package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
)

type SSHAgent struct {
	addr string

	config *ssh.ClientConfig
	client *ssh.Client
}

func Connect(agent *SSHAgent) {
	addr := fmt.Sprintf("%s:%d", *host, *port)
	client, err := ssh.Dial("tcp", agent.addr, agent.config)
	if err != nil {
		log.Panicln("Failed to establish connection to remote SSH '"+addr+"'", err)
	}

	agent.client = client
}

func ExecuteCommand(agent *SSHAgent, cmd string) {
	session, err := agent.client.NewSession()
	if err != nil {
		log.Panicln("Failed to create SSH session to '"+agent.addr+"'", err)
	}
	defer session.Close()

	var outBuffer, errBuffer bytes.Buffer
	session.Stdout = &outBuffer
	session.Stderr = &errBuffer
	err = session.Run(cmd)
	if err != nil {
		log.Panicln("Failed to run command '"+cmd+"' host: "+agent.addr, err)
	}

	fmt.Println(">", outBuffer.String())
}
