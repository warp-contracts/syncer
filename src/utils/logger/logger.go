package logger

import (
	"github.com/warp-contracts/syncer/src/utils/config"

	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
}

func Init(config *config.Config) (err error) {
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		return
	}
	logger.SetLevel(level)
	logger.SetOutput(os.Stdout)

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logger.SetFormatter(formatter)

	return nil
}

func NewSublogger(tag string) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"module": "warp." + tag})
}
