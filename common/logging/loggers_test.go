package logging

import (
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLoggers(t *testing.T) {
	Init("logs", "debug", 0)

	CLog().Debugf("format debug clog msg [%s]", "test clog msg")
	CLog().WithFields(logrus.Fields{"name": "clog_test", "type": "clog"}).Debugf("format debug clog msg [%s]", "test clog msg")

	VLog().Debugf("format debug vlog msg [%s]", "test vlog msg")
	VLog().WithFields(logrus.Fields{"name": "vlog_test", "type": "vlog"}).Debugf("format debug vlog msg [%s]", "test vlog msg")

}
