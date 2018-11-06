package printer

import (
	"github.com/coschain/contentos-go/demos/itimer"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	log "github.com/inconshreveable/log15"
	"time"
)

type Printer struct {
	timer  itimer.ITimer
	ticker *time.Ticker
	ctx    *node.ServiceContext
}

func New(ctx *node.ServiceContext) (*Printer, error) {
	return &Printer{ctx: ctx}, nil
}

func (t *Printer) Start(server *p2p.Server) error {
	s, err := t.ctx.Service("timer")
	if err != nil {
		log.Error("Service serviceTimer error : %v", err)
		return err
	}
	t.timer = s.(itimer.ITimer)
	t.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for _ = range t.ticker.C {
			log.Info(t.timer.GetCurrent())
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
