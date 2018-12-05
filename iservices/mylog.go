package iservices

import "github.com/sirupsen/logrus"

var LogServerName = "mylog"

type ILog interface {
	GetLog() *logrus.Logger
}