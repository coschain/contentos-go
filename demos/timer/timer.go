package timer

import (
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p"
	"time"
)

type Timer struct {
	currentTime time.Time
	ticker      *time.Ticker
}

func New(ctx *node.ServiceContext) (*Timer, error) {
	return &Timer{}, nil
}

func (t *Timer) Start(server *p2p.Server) error {
	t.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		for _ = range t.ticker.C {
			t.currentTime = time.Now()
		}
	}()
	return nil
}

func (t *Timer) Stop() error {
	t.ticker.Stop()
	return nil
}

func (t *Timer) Current() string {
	return t.currentTime.String()
}
