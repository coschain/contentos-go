package test

import (
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"sort"
	"strings"
	"time"
)

type Components struct {
	ConsensusSvc iservices.IConsensus
	P2pSvc       iservices.IP2P
}

type Monitor struct {
	compo              map[string]*Components
	validators         map[string]bool
	validatorList      *widgets.List
	vX1, vY1, vX2, vY2 int

	nonValidatorList       *widgets.List
	nvX1, nvY1, nvX2, nvY2 int

	chainInfoList      *widgets.List
	bX1, bY1, bX2, bY2 int

	marginStep       *widgets.Plot
	confirmationTime *widgets.Plot
	ci *CommitInfo

	firstBlock common.ISignedBlock
}

func NewMonitor(c []*Components) *Monitor {
	m := &Monitor{
		compo:         make(map[string]*Components),
		validators:    make(map[string]bool),
		validatorList: widgets.NewList(),
		vX1:           0,
		vY1:           5,
		vX2:           75,

		nonValidatorList: widgets.NewList(),
		nvX1:             0,
		nvX2:             75,

		chainInfoList: widgets.NewList(),
		bX1:           80,
		bY1:           5,
		bX2:           110,
		bY2:           10,

		marginStep: widgets.NewPlot(),
		ci: NewCommitInfo(),
	}

	for i := 0; i < len(c); i++ {
		m.compo[c[i].ConsensusSvc.GetName()] = c[i]
	}

	return m
}

func (m *Monitor) Run() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	bs, err := m.compo["initminer"].ConsensusSvc.FetchBlocks(1, 1)
	if err != nil {
		log.Fatal(err)
	}
	m.firstBlock = bs[0]

	m.compo["initminer1"].ConsensusSvc.SetHook("commit", m.ci.commitHook)
	m.compo["initminer1"].ConsensusSvc.SetHook("generate_block", m.ci.generateBlockHook)
	m.compo["initminer1"].ConsensusSvc.SetHook("switch_fork", m.ci.switchFork)
	m.compo["initminer1"].ConsensusSvc.SetHook("branches", m.ci.branches)

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
	v := m.compo["initminer"].ConsensusSvc.ActiveValidators()
	for i := range v {
		m.validators[v[i]] = true
	}
	m.validatorList.Title = "validators"
	m.vY2 = m.vY1 + len(m.validators) + 3
	m.validatorList.SetRect(m.vX1, m.vY1, m.vX2, m.vY2)
	m.validatorList.TextStyle.Fg = ui.ColorYellow
	m.validatorList.Rows = m.getInfo(m.validators)
}

func (m *Monitor) drawNonValidators() {
	nonV := make(map[string]bool)
	for k := range m.compo {
		if m.validators[k] == true {
			continue
		}
		nonV[k] = true
	}
	m.nonValidatorList.Title = "non-validators"
	m.nvY1 = m.vY2 + 3
	m.nvY2 = m.nvY1 + len(m.compo) - len(m.validators) + 3
	m.nonValidatorList.SetRect(m.nvX1, m.nvY1, m.nvX2, m.nvY2)
	m.nonValidatorList.TextStyle.Fg = ui.ColorYellow
	m.nonValidatorList.Rows = m.getInfo(nonV)
}

func (m *Monitor) drawNodeList() {
	m.validators = make(map[string]bool)
	m.drawValidators()
	m.drawNonValidators()
}

func (m *Monitor) drawChainInfo() {
	// latency
	// block count
	m.chainInfoList.Title = "Info"
	m.chainInfoList.SetRect(m.bX1, m.bY1, m.bX2, m.bY2)
	m.chainInfoList.TextStyle.Fg = ui.ColorCyan
	info := make([]string, 0, 3)
	info = append(info, fmt.Sprintf("Latency: %dms", m.compo["initminer"].P2pSvc.GetMockLatency()))
	cs := m.compo["initminer"].ConsensusSvc
	head, _ := cs.FetchBlock(cs.GetHeadBlockId())
	info = append(info, fmt.Sprintf("Total blocks: %d", cs.GetHeadBlockId().BlockNum()))
	info = append(info, fmt.Sprintf("Expected blocks: %d", head.Timestamp()-m.firstBlock.Timestamp()+1))
	m.chainInfoList.Rows = info
}

func (m *Monitor) drawMarginStep() {
	m.marginStep.Title = "margin step"
	m.marginStep.Data = make([][]float64, 1)
	m.marginStep.Data[0] = m.ci.MarginStepInfo()
	m.marginStep.SetRect(80, 10, 110, 20)
	m.marginStep.AxesColor = ui.ColorWhite
	m.marginStep.LineColors[0] = ui.ColorYellow
}

func (m *Monitor) drawConfirmationTime() {

}

func (m *Monitor) draw() {
	m.drawNodeList()
	m.drawChainInfo()
	m.drawMarginStep()
	m.drawConfirmationTime()

	ui.Clear()
	ui.Render(m.validatorList, m.nonValidatorList, m.chainInfoList, m.marginStep)
}

func formattedLine(c *Components) string {
	peers := c.P2pSvc.GetNodeNeighbours()
	ps := strings.Split(peers, ",")
	return fmt.Sprintf("%12s %12d %12d %12d",
		c.ConsensusSvc.GetName(),
		c.ConsensusSvc.GetHeadBlockId().BlockNum(),
		c.ConsensusSvc.GetLIB().BlockNum(),
		len(ps),
	)
}

func (m *Monitor) getInfo(names map[string]bool) []string {
	info := make([]string, len(names)+1)
	info[0] = fmt.Sprintf("%12s %12s %12s %12s", "NodeName", "HeadBlockNum", "LastCommitted", "ConnectedPeers")
	i := 0
	ns := make([]string, len(names))
	for name := range names {
		ns[i] = name
		i++
	}
	sort.Strings(ns)

	for i = range ns {
		info[i+1] = formattedLine(m.compo[ns[i]])
	}
	return info
}
