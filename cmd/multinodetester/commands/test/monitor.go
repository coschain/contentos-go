package test

import (
	"fmt"
	"github.com/coschain/contentos-go/consensus"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"time"
)

func Monitor(nodes []*node.Node) {
	csss := make([]iservices.IConsensus, 0, len(nodes))
	for i := 0; i < len(nodes); i++ {
		c, err := nodes[i].Service(iservices.ConsensusServerName)
		if err != nil {
			panic(err)
		}
		css := c.(iservices.IConsensus)
		csss = append(csss, css)
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	l := widgets.NewList()
	l.Title = "List"
	l.SetRect(0, 5, 145, 50)
	l.TextStyle.Fg = ui.ColorYellow

	draw := func(info []string) {
		l.Rows = info
		ui.Render(l)
	}

	draw(getInfo(csss))
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(2 * time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			}
		case <-ticker:
			draw(getInfo(csss))
		}
	}
}

func formattedLine(css iservices.IConsensus) string {
	c := css.(*consensus.SABFT)
	return fmt.Sprintf("%12s %12d %12d", c.Name, c.GetHeadBlockId().BlockNum(), c.GetLIB().BlockNum())
}

func getInfo(csss []iservices.IConsensus) []string {
	info := make([]string, len(csss)+1)
	info[0] = fmt.Sprintf("%12s %12s %12s", "NodeName", "HeadBlockNum", "LastCommitted")
	for i := 0; i < len(csss); i++ {
		info[i+1] = formattedLine(csss[i])
	}
	return info
}
