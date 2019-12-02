package commands

import (
	"errors"
	"fmt"
	"github.com/beevik/ntp"
	"time"
)

func checkNTPTime() error {
	ntpTime, err := ntp.Time("pool.ntp.org")
	if err != nil {
		err := errors.New(fmt.Sprintf("Acquire ntp time error %s", err))
		return err
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