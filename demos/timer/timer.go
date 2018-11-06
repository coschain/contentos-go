package timer

import (
	"fmt"
	"github.com/coschain/contentos-go/demos/iprinter"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	log "github.com/inconshreveable/log15"
	"time"
)

type Timer struct {
	currentTime time.Time
	ticker      *time.Ticker
	ctx         *node.ServiceContext
}

func New(ctx *node.ServiceContext) (*Timer, error) {
	return &Timer{ctx: ctx}, nil
}

func (t *Timer) getPrinter() (iprinter.IPrinter, error) {
	s, err := t.ctx.Service("printer")
	if err != nil {
		log.Error(fmt.Sprintf("Service serviceTimer error : %v", err))
		return nil, err
	}
	printer := s.(iprinter.IPrinter)
	return printer, nil
}

func (t *Timer) Start(server *p2p.Server) error {
	printer, err := t.getPrinter()
	if err != nil {
		return err
	}
	t.ticker = time.NewTicker(500 * time.Millisecond)
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
