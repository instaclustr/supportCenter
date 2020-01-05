package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
)

var (
	host = flag.String("host", "", "Target hostname")
	user = flag.String("u", "", "User name")
	port = flag.Int("p", 22, "Port")
)

func validateCommandLineArguments() {
	if *host == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *user == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	log.Println("Instaclustr Agent +")

	flag.Parse()
	validateCommandLineArguments()

	fmt.Println("Target host is: ", *host)

	agent := &SSHAgent{
		addr: fmt.Sprintf("%s:%d", *host, *port),
	}
	agent.config = &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password("qweasd!"),
		},
		// TODO Ask if we need to check host keys
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	Connect(agent)

	ExecuteCommand(agent, "uname -a")
}
