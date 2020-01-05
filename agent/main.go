package main

import (
	"flag"
	"fmt"
	"github.com/instaclustr/supportCenter/agent/collector"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh"
	"os"
)

var (
	host = flag.String("host", "", "Target hostname")
	user = flag.String("u", "", "User name")
	port = flag.Int("p", 22, "Port")
)

var log = logrus.New()

func init() {
	log.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

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
	log.Info("Instaclustr Agent")

	flag.Parse()
	validateCommandLineArguments()

	log.Info("Target host is: ", *host)

	done := make(chan bool, 2)
	go func() {
		collector.CollectLogs(*host)
		done <- true
	}()

	go func() {
		collector.CollectStats("asdasd")
		done <- true
	}()

	<-done
	<-done

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
