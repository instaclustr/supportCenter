package main

import (
	"agent/collector"
	"flag"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh"
	"path/filepath"
	"time"
)

const timestampPattern = "20060102T150405"
const collectingFolder = "data"

var (
	user = flag.String("l", "", "User to log in as on the remote machine")
	port = flag.Int("p", 22, "Port to connect to on the remote csHost")
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

	collectingTimestamp := time.Now().UTC().Format(timestampPattern)
	log.Info("Collecting timestamp: ", collectingTimestamp)

	metricsCollector := collector.MetricsCollector{
		Settings: &settings.Metrics,
		Log:      log,
		Path:     filepath.Join(".", collectingFolder, collectingTimestamp),
	}

	logsCollector := collector.LogsCollector{
		Settings: &settings.Logs,
		Log:      log,
		Path:     filepath.Join(".", collectingFolder, collectingTimestamp),
	}

	log.Info("Metrics collecting hosts are: ", mcTargets.String())
	log.Info("Log collecting hosts are: ", lcTargets.String())

	completed := make(chan bool, len(mcTargets.hosts)+len(lcTargets.hosts))

	for _, host := range mcTargets.hosts {
		go func(host string) {
			agent := &collector.SSHAgent{}
			agent.SetTarget(host, *port)
			agent.SetConfig(&ssh.ClientConfig{
				User: *user,
				Auth: []ssh.AuthMethod{
					// TODO ask password or search private key
					ssh.Password("qweasd!"),
				},
				// TODO Ask if we need to check host keys
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			})

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
			agent.SetConfig(&ssh.ClientConfig{
				User: *user,
				Auth: []ssh.AuthMethod{
					// TODO ask password or search private key
					ssh.Password("qweasd!"),
				},
				// TODO Ask if we need to check host keys
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			})

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
}
