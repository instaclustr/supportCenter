package collector

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var scLogger = logrus.New()

func init() {
	scLogger.Formatter = &prefixed.TextFormatter{
		FullTimestamp: true,
	}
}

func CollectStats(agent *SSHAgent) error {
	log := scLogger.WithFields(logrus.Fields{
		"prefix": "SC " + agent.addr,
	})
	log.Info("Stats collecting started")

	log.Info("Stats collecting completed")
	return nil
}
