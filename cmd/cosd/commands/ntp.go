package commands

import (
	"errors"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/sirupsen/logrus"
	"time"
)

var serverList = []string{"pool.ntp.org", "cn.pool.ntp.org"}

func checkNTPTime(log *logrus.Logger) error {
	var ntpTime time.Time
	var err error
	for i:=0;i<len(serverList);i++ {
		ntpTime, err = ntp.Time( serverList[i] )
		if err != nil {
			if i == len(serverList)-1 {
				return errors.New(fmt.Sprintf("Acquire ntp time error %s", err))
			}
		} else {
			log.Info("ntp server ", serverList[i])
			break
		}
	}
	localTime := time.Now()
	ntpTimeSec := ntpTime.Unix()
	localTimeSec := localTime.Unix()
	if ntpTimeSec - localTimeSec > 1 || ntpTimeSec - localTimeSec < -1 {
		err := errors.New(fmt.Sprintf("Gap between ntp time and local time greater than 1 second, ntp %d, local %d", ntpTimeSec, localTimeSec))
		return err
	}
	return nil
}