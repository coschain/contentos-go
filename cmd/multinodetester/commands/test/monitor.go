package test

import (
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"sort"
	"strings"
	"time"
)

type components struct {
	consensusSvc iservices.IConsensus
	p2pSvc       iservices.IP2P
}

type Monitor struct {
	compo              map[string]*components
	validators         map[string]bool
	validatorList      *widgets.List
	vX1, vY1, vX2, vY2 int

	nonValidatorList       *widgets.List
	nvX1, nvY1, nvX2, nvY2 int

	chainInfoList      *widgets.List
	bX1, bY1, bX2, bY2 int

	firstBlock common.ISignedBlock
}

func NewMonitor(nodes []*node.Node) *Monitor {
	m := &Monitor{
		compo:         make(map[string]*components),
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
		bY2:           15,
	}

	for i := 0; i < len(nodes); i++ {
		c, err := nodes[i].Service(iservices.ConsensusServerName)
		if err != nil {
			panic(err)
		}
		css := c.(iservices.IConsensus)

		p, err := nodes[i].Service(iservices.P2PServerName)
		if err != nil {
			panic(err)
		}
		p2p := p.(iservices.IP2P)
		m.compo[css.GetName()] = &components{
			consensusSvc: css,
			p2pSvc:       p2p,
		}
	}

	return m
}

func (m *Monitor) Run() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	bs, err := m.compo["initminer"].consensusSvc.FetchBlocks(1, 1)
	if err != nil {
		log.Fatal(err)
	}
	m.firstBlock = bs[0]

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
	v := m.compo["initminer"].consensusSvc.ActiveValidators()
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
	m.nvY1 = m.vY2 + 5
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
	info = append(info, "Latency: 1500ms")
	cs := m.compo["initminer"].consensusSvc
	head, _ := cs.FetchBlock(cs.GetHeadBlockId())
	info = append(info, fmt.Sprintf("Total blocks: %d", cs.GetHeadBlockId().BlockNum()))
	info = append(info, fmt.Sprintf("Expected blocks: %d", head.Timestamp()-m.firstBlock.Timestamp()+1))
	m.chainInfoList.Rows = info
}

func (m *Monitor) draw() {
	m.drawNodeList()
	m.drawChainInfo()

	ui.Clear()
	ui.Render(m.validatorList, m.nonValidatorList, m.chainInfoList)
}

func formattedLine(c *components) string {
	peers := c.p2pSvc.GetNodeNeighbours()
	ps := strings.Split(peers, ",")
	return fmt.Sprintf("%12s %12d %12d %12d",
		c.consensusSvc.GetName(),
		c.consensusSvc.GetHeadBlockId().BlockNum(),
		c.consensusSvc.GetLIB().BlockNum(),
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
