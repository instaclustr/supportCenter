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

func CollectStats(host string) {
	log := scLogger.WithFields(logrus.Fields{
		"prefix": "SC " + host,
	})

	log.Info("Stats collector started")

}
