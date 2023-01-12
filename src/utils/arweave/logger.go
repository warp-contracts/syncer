package arweave

import (
	"syncer/src/utils/logger"

	"github.com/sirupsen/logrus"
)

// Transforms all logs to debug
type Logger struct {
	log *logrus.Entry
}

func NewLogger() (self *Logger) {
	self = new(Logger)
	self.log = logger.NewSublogger("arweave-resty")
	return
}

func (self *Logger) Errorf(format string, v ...interface{}) {
	self.log.Tracef(format, v...)
}
func (self *Logger) Warnf(format string, v ...interface{}) {
	self.log.Tracef(format, v...)
}
func (self *Logger) Debugf(format string, v ...interface{}) {
	self.log.Tracef(format, v...)
}
