package collector

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var lcLogger = logrus.New()

func init() {
	lcLogger.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

func CollectLogs(agent *SSHAgent) error {
	log := lcLogger.WithFields(logrus.Fields{
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
