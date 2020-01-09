package collector

import (
	"github.com/sirupsen/logrus"
)

/*
Settings
*/
type LogsCollectorSettings struct {
}

func LogsCollectorDefaultSettings() *LogsCollectorSettings {
	return &LogsCollectorSettings{}
}

/*
Collector
*/
type LogsCollector struct {
	Settings *LogsCollectorSettings
	Log      *logrus.Logger
}

func (collector *LogsCollector) Collect(agent *SSHAgent) error {
	log := collector.Log.WithFields(logrus.Fields{
		"prefix": "LC " + agent.host,
	})
	log.Info("Logs collecting started")

	err := agent.Connect()
	if err != nil {
		log.Error(err)
		return err
	}

	sout, serr, err := agent.ExecuteCommand("uname -a")
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info(sout.String())
	log.Info(serr.String())

	log.Info("Logs collecting completed")
	return nil
}
