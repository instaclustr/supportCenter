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

func CollectLogs(host string) {
	log := lcLogger.WithFields(logrus.Fields{
		"prefix": "LC " + host,
	})

	log.Info("Logs collector started")

}
