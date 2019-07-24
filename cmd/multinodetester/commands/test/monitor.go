package test

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"time"
)

type Monitor struct {
	consensus  map[string]iservices.IConsensus
	validators map[string]bool
	validatorList *widgets.List
	vX1, vY1, vX2, vY2 int

	nonValidatorList *widgets.List
	nvX1, nvY1, nvX2, nvY2 int
}

func NewMonitor(nodes []*node.Node) *Monitor {
	m := &Monitor{
		consensus: make(map[string]iservices.IConsensus),
		validators: make(map[string]bool),
		validatorList: widgets.NewList(),
		vX1: 0,
		vY1: 5,
		vX2: 100,

		nonValidatorList: widgets.NewList(),
		nvX1: 0,
		nvX2: 100,
	}

	for i := 0; i < len(nodes); i++ {
		c, err := nodes[i].Service(iservices.ConsensusServerName)
		if err != nil {
			panic(err)
		}
		css := c.(iservices.IConsensus)
		m.consensus[css.GetName()] = css
	}

	return m
}

func (m *Monitor) Run() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	m.draw()
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
			m.draw()
		}
	}
}

func (m *Monitor) drawValidators() {
	v := m.consensus["initminer"].ActiveValidators()
	//log.Println("dff     dddddddddddddddddddddddddddddddddddddddddd", v)
	for i := range v {
		m.validators[v[i]] = true
	}
	m.validatorList.Title = "validators"
	m.vY2 = m.vY1 + len(m.validators)+5
	m.validatorList.SetRect(m.vX1, m.vY1, m.vX2, m.vY2)
	m.validatorList.TextStyle.Fg = ui.ColorYellow
	m.validatorList.Rows = m.getInfo(m.validators)
}

func (m *Monitor) drawNonValidators() {
	nonV := make(map[string]bool)
	for k := range m.consensus {
		if m.validators[k] == true {
			continue
		}
		nonV[k] = true
	}
	m.nonValidatorList.Title = "non-validators"
	m.nvY1 = m.vY2 + 15
	m.nvY2 = m.nvY1 + len(m.consensus)-len(m.validators)
	m.nonValidatorList.SetRect(m.nvX1, m.nvY1, m.nvX2, m.nvY2)
	m.nonValidatorList.TextStyle.Fg = ui.ColorYellow
	m.nonValidatorList.Rows = m.getInfo(nonV)
}

func (m *Monitor) drawNodeList() {
	m.validators = make(map[string]bool)
	m.drawValidators()
	m.drawNonValidators()
}

func (m *Monitor) draw() {
	m.drawNodeList()

	ui.Render(m.validatorList, m.nonValidatorList)
}

func formattedLine(css iservices.IConsensus) string {
	return fmt.Sprintf("%12s %12d %12d", css.GetName(), css.GetHeadBlockId().BlockNum(), css.GetLIB().BlockNum())
}

func (m *Monitor) getInfo(names map[string]bool) []string {
	info := make([]string, len(names)+1)
	info[0] = fmt.Sprintf("%12s %12s %12s", "NodeName", "HeadBlockNum", "LastCommitted")
	i := 0
	for name := range names {
		info[i+1] = formattedLine(m.consensus[name])
		i++
	}
	return info
}
