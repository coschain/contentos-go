package commoninterface

//
// This file contains interfaces of node.Service's
//

import (
	"github.com/coschain/contentos-go/db/storage"
)

// Database Service
type IDatabaseService interface {
	storage.TagRevertible
	storage.Transactional
	storage.Database
}
