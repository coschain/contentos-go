package printer

import (
	"fmt"
	"github.com/coschain/contentos-go/demos/itimer"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	log "github.com/inconshreveable/log15"
	"time"
)

type Printer struct {
	ticker *time.Ticker
	ctx    *node.ServiceContext
}

func New(ctx *node.ServiceContext) (*Printer, error) {
	return &Printer{ctx: ctx}, nil
}

func (t *Printer) getTimer() (itimer.ITimer, error) {
	s, err := t.ctx.Service("timer")
	if err != nil {
		log.Error(fmt.Sprintf("Service serviceTimer error : %v", err))
		return nil, err
	}
	timer := s.(itimer.ITimer)
	return timer, nil
}

func (t *Printer) Start(server *p2p.Server) error {
	timer, err := t.getTimer()
	if err != nil {
		return err
	}
	t.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for range t.ticker.C {
			log.Info(timer.GetCurrent())
		}
	}()
	return nil
}

func (t *Printer) Stop() error {
	t.ticker.Stop()
	return nil
}

func (t *Printer) GetCurrent() string {
	return "printer printer"
}
