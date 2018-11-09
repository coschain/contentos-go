package timer

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/node"
	log "github.com/inconshreveable/log15"
	"time"
)

type Timer struct {
	currentTime time.Time
	ticker      *time.Ticker
	ctx         *node.ServiceContext
	interval    int
}

func New(ctx *node.ServiceContext, config service_configs.TimerConfig) (*Timer, error) {
	interval := config.Interval
	return &Timer{ctx: ctx, interval: interval}, nil
}

func (t *Timer) getPrinter() (iservices.IPrinter, error) {
	s, err := t.ctx.Service("printer")
	if err != nil {
		log.Error(fmt.Sprintf("Service serviceTimer error : %v", err))
		return nil, err
	}
	printer := s.(iservices.IPrinter)
	return printer, nil
}

func (t *Timer) Start(node* node.Node) error {
	printer, err := t.getPrinter()
	if err != nil {
		return err
	}
	t.ticker = time.NewTicker(time.Duration(t.interval) * time.Millisecond)
	go func() {
		for range t.ticker.C {
			log.Info(printer.GetCurrent())
			t.currentTime = time.Now()
		}
	}()
	return nil
}

func (t *Timer) Stop() error {
	t.ticker.Stop()
	return nil
}

func (t *Timer) GetCurrent() string {
	return t.currentTime.String()
}
