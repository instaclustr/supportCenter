package main

import (
	"agent/collector"
	"flag"
	"fmt"
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const timestampPattern = "20060102T150405"
const knownHostsPath = "/.ssh/known_hosts"
const defaultPrivateKeyPath = "/.ssh/id_rsa"

var (
	user              = flag.String("l", "", "User to log in as on the remote machine")
	port              = flag.Int("p", 22, "Port to connect to on the remote host")
	disableKnownHosts = flag.Bool("disable_known_hosts", false, "Skip loading the user’s known-hosts file")
	mcTimeRangeFrom   = flag.String("mc-from", "", "Datetime (RFC3339 format, 2006-01-02T15:04:05Z07:00) to fetch metrics from some time point. (Default 1970-01-01 00:00:00 +0000 UTC)")
	mcTimeRangeTo     = flag.String("mc-to", "", "Datetime (RFC3339 format, 2006-01-02T15:04:05Z07:00) to fetch metrics to some time point. (Default current datetime)")
	configPath        = flag.String("config", "", "The path to the configuration file")

	mcTargets   StringList
	ncTargets   StringList
	privateKeys StringList

	collectingTimestamp = time.Now().UTC().Format(timestampPattern)

	mcTimestampFrom = time.Unix(0, 0).UTC()
	mcTimestampTo   = time.Now().UTC()
)

var log = logrus.New()

func init() {
	log.Formatter = &prefixed.TextFormatter{
		ForceFormatting: true,
		ForceColors:     true,
		FullTimestamp:   true,
	}
}

func init() {
	flag.Var(&mcTargets, "mc", "Metrics collecting hostname")
	flag.Var(&ncTargets, "nc", "Node collecting hostnames")
	flag.Var(&privateKeys, "pk", "List of files from which the identification keys (private key) for public key authentication are read, in addition to default one (Default [HOME]/.ssh/id_rsa)")
}

func main() {
	// Init file logging
	agentLogPath := filepath.Join(".", "agent.log")
	agentLogFile, err := os.Create(agentLogPath)
	if err != nil {
		log.Fatalf("Failed to open agent log file %s for output: %s", agentLogPath, err)
	}
	defer agentLogFile.Close()
	log.SetOutput(io.MultiWriter(colorable.NewColorableStdout(), agentLogFile))

	log.Info("Instaclustr Agent")

	initCommandLineParameters()

	// Settings
	settings := &Settings{
		Agent:   *AgentDefaultSettings(),
		Node:    *collector.NodeCollectorDefaultSettings(),
		Metrics: *collector.MetricsCollectorDefaultSettings(),
		Target:  *TargetDefaultSettings(),
	}

	settingsPath := Expand(SearchSettingsPath(*configPath))
	exists, _ := Exists(settingsPath)
	if exists == true {
		log.Info("Loading settings from '", settingsPath, "'...")
		err := settings.Load(settingsPath)
		if err != nil {
			log.Warn(err)
		}
	} else {
		log.Warn("The settings file '", settingsPath, "' does not exists")
	}

	// SSH Settings
	knownHostsKeyCallback := loadKnownHostsKey()
	privateKeySigners := loadPrivateKeySigners()
	agentForwardingSigners := loadAgentForwardingSigners()

	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(append(privateKeySigners, agentForwardingSigners...)...),
		},
		HostKeyCallback: knownHostsKeyCallback,
		Timeout:         time.Second * 2,
	}

	collectingRootFolder := Expand(settings.Agent.CollectedDataPath)

	// Collecting
	collectingPath := filepath.Join(collectingRootFolder, collectingTimestamp)
	if os.MkdirAll(collectingPath, os.ModePerm) != nil {
		log.Warn("Failed to create collecting folder '" + collectingPath + "'")
	}

	log.Info("Collecting timestamp: ", collectingTimestamp)

	metricsCollector := collector.MetricsCollector{
		Settings:      &settings.Metrics,
		Logger:        log,
		Path:          filepath.Join(collectingPath, "metrics"),
		TimestampFrom: mcTimestampFrom,
		TimestampTo:   mcTimestampTo,
	}

	nodesCollector := collector.NodeCollector{
		Settings: &settings.Node,
		Logger:   log,
		Path:     filepath.Join(collectingPath, "nodes"),
		AppFs:    afero.NewOsFs(),
	}

	metricsTargets := JoinToSet(settings.Target.Metrics, mcTargets.items)
	nodeTargets := JoinToSet(settings.Target.Nodes, ncTargets.items)

	if len(metricsTargets) > 1 {
		metricsTargets = metricsTargets[:1]
	}
	log.Info("Metrics collecting hosts are: ", metricsTargets)
	log.Info("Metrics collecting time span: ", mcTimestampFrom.UTC(), " ... ", mcTimestampTo.UTC())
	log.Info("Node collecting hosts are: ", nodeTargets)

	taskCount := len(metricsTargets) + len(nodeTargets)

	var wg sync.WaitGroup
	wg.Add(taskCount)

	for _, host := range metricsTargets {
		go func(host string) {
			defer wg.Done()

			sshAgent := &collector.SSHAgent{}
			sshAgent.SetTarget(host, *port)
			sshAgent.SetConfig(sshConfig)

			err := metricsCollector.Collect(sshAgent)
			if err != nil {
				log.Error("Failed to collect metrics on '" + host + "'")
			}
		}(host)
	}

	for _, host := range nodeTargets {
		go func(host string) {
			defer wg.Done()

			sshAgent := &collector.SSHAgent{}
			sshAgent.SetTarget(host, *port)
			sshAgent.SetConfig(sshConfig)

			err := nodesCollector.Collect(sshAgent)
			if err != nil {
				log.Error("Failed to collect node on '" + host + "'")
			}
		}(host)
	}

	wg.Wait()

	// Compressing tarball
	log.Info("Compressing collected data (", collectingPath, ")...")

	// - terminate file logging
	log.SetOutput(os.Stdout)
	agentLogFile.Sync()
	agentLogFile.Close()
	err = CopyFile(agentLogPath, filepath.Join(collectingPath, "agent.log"))
	if err != nil {
		log.Warn("Failed to copy agent log to collecting folder: " + err.Error())
	}

	tarball := filepath.Join(collectingRootFolder, fmt.Sprint(collectingTimestamp, "-data.zip"))
	err = Zip(collectingPath, tarball)
	if err != nil {
		log.Error("Failed to compress collected data (", err, ")")
	} else {
		log.Info("Compressing collected data  OK")
	}

	log.Info("Tarball: ", tarball)
}

func loadKnownHostsKey() ssh.HostKeyCallback {
	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if !(*disableKnownHosts) {
		hostsFilePath := filepath.Join(os.Getenv("HOME"), knownHostsPath)

		log.Info("Loading known host '", hostsFilePath, "'...")
		callback, err := knownhosts.New(hostsFilePath)
		if err != nil {
			log.Error("Filed to load known hosts (" + err.Error() + ")")
			os.Exit(1)
		}

		hostKeyCallback = callback
	}
	return hostKeyCallback
}

func loadPrivateKeySigners() []ssh.Signer {
	var signers []ssh.Signer

	keys := []string{
		filepath.Join(os.Getenv("HOME"), defaultPrivateKeyPath),
		// TODO Does somebody use DSA?
		//filepath.Join(os.Getenv("HOME"), "/.ssh/id_dsa"),
	}

	keys = append(keys, privateKeys.items...)

	for _, keyPath := range keys {
		log.Info("Loading private key '", keyPath, "'...")

		key, err := ioutil.ReadFile(keyPath)
		if err != nil {
			log.Warn("Failed to read key '" + keyPath + "' (" + err.Error() + ")")
			continue
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			log.Error("Failed to parse private key '" + keyPath + "' (" + err.Error() + ")")
			continue
		}

		signers = append(signers, signer)
	}

	return signers
}

func loadAgentForwardingSigners() []ssh.Signer {
	socket := os.Getenv("SSH_AUTH_SOCK")

	conn, err := net.Dial("unix", socket)
	if err == nil {
		agentClient := agent.NewClient(conn)

		signers, err := agentClient.Signers()
		if err == nil {
			return signers
		} else {
			log.Warnf("Failed to provide agent forwarded signers: %v", err)
		}
	} else {
		log.Warnf("Failed to open SSH_AUTH_SOCK: %v", err)
	}

	return []ssh.Signer{}
}
