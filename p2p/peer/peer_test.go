package peer

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

var p *Peer

func init() {
	p = NewPeer(logrus.New())
	p.base.version = 1
	p.base.services = 1
	p.base.syncPort = 10338
	p.base.consPort = 10339
	p.base.relay = true
	p.base.height = 12345
	p.base.id = 12345678910
}
func TestGetPeerComInfo(t *testing.T) {
	log := logrus.New()
	p.DumpInfo(log)
	if p.base.GetVersion() != 1 {
		t.Errorf("PeerCom GetVersion error")
	} else {
		p.base.SetVersion(2)
		if p.base.GetVersion() != 2 {
			t.Errorf("PeerCom SetVersion error")
		}
	}

	if p.base.GetServices() != 1 {
		t.Errorf("PeerCom GetServices error")
	} else {
		p.base.SetServices(2)
		if p.base.GetServices() != 2 {
			t.Errorf("PeerCom SetServices error")
		}
	}

	if p.base.GetSyncPort() != 10338 {
		t.Errorf("PeerCom GetSyncPort error")
	} else {
		p.base.SetSyncPort(20338)
		if p.base.GetSyncPort() != 20338 {
			t.Errorf("PeerCom SetSyncPort error")
		}
	}

	if p.base.GetConsPort() != 10339 {
		t.Errorf("PeerCom GetConsPort error")
	} else {
		p.base.SetConsPort(20339)
		if p.base.GetConsPort() != 20339 {
			t.Errorf("PeerCom SetConsPort error")
		}
	}

	if p.base.GetRelay() != true {
		t.Errorf("PeerCom GetRelay error")
	} else {
		p.base.SetRelay(false)
		if p.base.GetRelay() != false {
			t.Errorf("PeerCom SetRelay error")
		}
	}

	if p.base.GetHeight() != 12345 {
		t.Errorf("PeerCom GetHeight error")
	} else {
		p.base.SetHeight(987654321)
		if p.base.GetHeight() != 987654321 {
			t.Errorf("PeerCom SetHeight error")
		}
	}

	if p.base.GetID() != 12345678910 {
		t.Errorf("PeerCom GetID error")
	} else {
		p.base.SetID(10987654321)
		if p.base.GetID() != 10987654321 {
			t.Errorf("PeerCom SetID error")
		}
	}
}

func TestUpdatePeer(t *testing.T) {
	p.UpdateInfo(time.Now(), 3, 3, 30338, 30339, 0x65535, 0, 13579, "abc")
	p.SetConsState(2)
	p.SetSyncState(3)
	p.SyncLink.SetAddr("127.0.0.1:20338")
	log := logrus.New()
	p.DumpInfo(log)
}
