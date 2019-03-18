package pprof

import (
	"github.com/coschain/contentos-go/common/logging"
	"net/http"
	_ "net/http/pprof"
)

func StartPprof() {
	go func() {
		logging.CLog().Infof("%s", http.ListenAndServe("127.0.0.1:6060", nil))
	}()
}
