package plugins

import (
	"fmt"
	service_configs "github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestDailyStatisticService_All(t *testing.T) {
	databaseConfig := &service_configs.DatabaseConfig{Driver: "mysql", User: "contentos", Password: "123456", Db: "contentosdb"}
	logger := logrus.New()
	service, err := NewDailyStatisticService(nil, databaseConfig, logger)
	if err != nil {
		fmt.Println(err)
	}
	err = service.Start(nil)
	if err != nil {
		fmt.Println()
	}
	//row1, err := service.make("photogrid", "2019-08-31")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(row1)
	rows := service.DailyStatsSince(30, "photogrid")
	fmt.Println(rows)
	rows2 := service.DailyStatsSince(30, "photogrid")
	fmt.Println(rows2)
	service.cron()

	err = service.Stop()
	if err != nil {
		fmt.Println(err)
	}
}

