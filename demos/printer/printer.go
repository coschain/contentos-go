package printer

import (
	"github.com/coschain/contentos-go/demos/timer"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	log "github.com/inconshreveable/log15"
	"time"
)

type Printer struct {
	timer  *timer.Timer
	ticker *time.Ticker
}

func New(ctx *node.ServiceContext) (*Printer, error) {
	var serviceTimer *timer.Timer
	if err := ctx.Service(&serviceTimer); err != nil {
		log.Error("Service serviceTimer error : %v", err)
	}
	return &Printer{timer: serviceTimer}, nil
}

func (t *Printer) Start(server *p2p.Server) error {
	t.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for _ = range t.ticker.C {
			log.Info(t.timer.Current())
		}
	}()
	return nil
}

func (t *Printer) Stop() error {
	t.ticker.Stop()
	return nil
}
