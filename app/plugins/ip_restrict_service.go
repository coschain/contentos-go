package plugins

import (
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type TpsCounter struct {
	LastUpdate int64  // UTC seconds elapsed
	Count uint32
}

type IpRestrictService struct {
	node.Service
	blackList map[string]bool // ip
	whiteList map[string]bool // ip
	monitorList map[string]*TpsCounter // ip -> (request count / second)
	lock                   sync.RWMutex
	ctx *node.ServiceContext
	log *logrus.Logger
	requestThreshold uint32
}

func NewIpRestrictService(ctx *node.ServiceContext, log *logrus.Logger) (*IpRestrictService,error) {
	ipsrv := &IpRestrictService{
		ctx:ctx,
		log:log,
		monitorList:make(map[string]*TpsCounter),
	}
	return ipsrv,nil
}

func (s *IpRestrictService) Start(node *node.Node) error {
	s.LoadConfig()
	return nil
}

func (s *IpRestrictService) Stop() error {
	return nil
}

func (s *IpRestrictService) Reload(config *node.Config) error {
	s.ctx.ResetConfig(config)
	s.LoadConfig()
	return nil
}

func (s *IpRestrictService) LoadConfig() {
	s.lock.Lock()
	defer s.lock.Unlock()
	// reset map
	s.blackList = make(map[string]bool)
	s.whiteList = make(map[string]bool)

	// init white list
	for _, ip := range s.ctx.Config().IpWhiteList {
		s.whiteList[ip] = true
	}
	// init black list
	for _, ip := range s.ctx.Config().IpBlackList {
		s.blackList[ip] = true
	}

	s.requestThreshold = s.ctx.Config().RequestThreshold
}

func (s *IpRestrictService) AddToWhiteList(ip string) {
	s.whiteList[ip] = true
}

func (s *IpRestrictService) AddToBlackList(ip string) {
	s.blackList[ip] = true
}

func (s *IpRestrictService) IsValidIp(ip string) bool {
	parsedIp := net.ParseIP(ip)
	return parsedIp != nil
}

func (s *IpRestrictService) UpdateMonitor(ip string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.monitorList[ip]; !ok {
		s.monitorList[ip] = &TpsCounter{LastUpdate:time.Now().Unix(),Count:1}
	} else {
		if time.Now().Unix() - s.monitorList[ip].LastUpdate >= 1 {
			s.monitorList[ip].Count = 1
		} else {
			s.monitorList[ip].Count++
		}
		s.monitorList[ip].LastUpdate = time.Now().Unix()
	}
}

func (s *IpRestrictService) HitWhiteList(ip string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// white list
	if _,ok := s.whiteList[ip];ok {
		return true
	}

	return false
}

func (s *IpRestrictService) HitBlackList(ip string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// black list
	if _,ok := s.blackList[ip];ok {
		return true
	}
	return false
}

func (s *IpRestrictService) HitMonitorList(ip string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// dynamic change list
	if tc,ok := s.monitorList[ip];ok {
		if tc.Count > s.requestThreshold {
			return true
		}
	}
	return false
}
