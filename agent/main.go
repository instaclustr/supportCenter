package main

import (
	"agent/collector"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"os"
	"path/filepath"
	"time"
)

const timestampPattern = "20060102T150405"
const collectingRootFolder = "data"
const knownHostsPath = "/.ssh/known_hosts"

var (
	user              = flag.String("l", "", "User to log in as on the remote machine")
	port              = flag.Int("p", 22, "Port to connect to on the remote host")
	disableKnownHosts = flag.Bool("disable_known_hosts", false, "Skip loading the user’s known-hosts file")
)

var log = logrus.New()

func init() {
	log.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

func main() {
	log.Info("Instaclustr Agent")

	var mcTargets HostList
	flag.Var(&mcTargets, "mc", "Metrics collecting hostname")

	var lcTargets HostList
	flag.Var(&lcTargets, "lc", "Log collecting hostnames")

	flag.Parse()
	validateCommandLineArguments()

	settings := &Settings{
		Logs:    *collector.LogsCollectorDefaultSettings(),
		Metrics: *collector.MetricsCollectorDefaultSettings(),
	}
	settingsPath := "settings.yml"
	exists, _ := Exists(settingsPath)
	if exists == true {
		log.Info("Loading settings from ", settingsPath, "...")
		err := settings.Load(settingsPath)
		if err != nil {
			log.Warn(err)
		}
	}

	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if !(*disableKnownHosts) {
		path := filepath.Join(os.Getenv("HOME"), knownHostsPath)
		log.Info("Loading known host '", path, "'...")
		callback, err := knownhosts.New(path)
		if err != nil {
			log.Error("Filed to load known hosts (" + err.Error() + ")")
		}

		hostKeyCallback = callback
	}

	collectingTimestamp := time.Now().UTC().Format(timestampPattern)
	collectingFolder := filepath.Join(".", collectingRootFolder, collectingTimestamp)
	log.Info("Collecting timestamp: ", collectingTimestamp)

	metricsCollector := collector.MetricsCollector{
		Settings: &settings.Metrics,
		Log:      log,
		Path:     collectingFolder,
	}

	logsCollector := collector.LogsCollector{
		Settings: &settings.Logs,
		Log:      log,
		Path:     collectingFolder,
	}

	log.Info("Metrics collecting hosts are: ", mcTargets.String())
	log.Info("Log collecting hosts are: ", lcTargets.String())

	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			// TODO ask password or search private key
			ssh.Password("qweasd!"),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Second * 2,
	}

	completed := make(chan bool, len(mcTargets.hosts)+len(lcTargets.hosts))

	for _, host := range mcTargets.hosts {
		go func(host string) {
			agent := &collector.SSHAgent{}
			agent.SetTarget(host, *port)
			agent.SetConfig(sshConfig)

			err := metricsCollector.Collect(agent)
			if err != nil {
				log.Error("Failed to collect logs from " + host)
			}

			completed <- true
		}(host)
	}

	for _, host := range lcTargets.hosts {
		go func(host string) {
			agent := &collector.SSHAgent{}
			agent.SetTarget(host, *port)
			agent.SetConfig(sshConfig)

			err := logsCollector.Collect(agent)
			if err != nil {
				log.Error("Failed to collect logs from " + host)
			}

			completed <- true
		}(host)
	}

	// TODO Add timeout maybe
	for i := 0; i < len(mcTargets.hosts)+len(lcTargets.hosts); i++ {
		<-completed
	}

	log.Info("Compressing collected data (", collectingFolder, ")...")
	tarball := filepath.Join(collectingRootFolder, fmt.Sprint(collectingTimestamp, "-data.zip"))
	err := Zip(collectingFolder, tarball)
	if err != nil {
		log.Error("Failed to compress collected data (", err, ")")
	} else {
		log.Info("Compressing collected data  OK")
	}

	log.Info("Tarball: ", tarball)
}
