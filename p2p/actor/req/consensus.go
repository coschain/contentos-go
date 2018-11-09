package req

import (
	"github.com/ontio/ontology-eventbus/actor"
)

var ConsensusPid *actor.PID

func SetConsensusPid(conPid *actor.PID) {
	ConsensusPid = conPid
}
