package rpc

import (
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"net/http"
	"strings"
)

const (
	// DefaultHTTPLimit default max http conns
	DefaultProxyHTTPLimit = 2000
)

func makeHttpOriginFunc() func(origin string) bool {
	return func(origin string) bool {
		return true
	}}

func RunWebProxy(api *APIService, grpcServer *grpc.Server, config *service_configs.GRPCConfig) error {

	options := []grpcweb.Option{
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(makeHttpOriginFunc()),
	}

	wrappedGrpc := grpcweb.WrapServer(grpcServer, options... )

	mux := http.NewServeMux()

	httpLimit := DefaultProxyHTTPLimit
	if config.HTTPLimit != 0 {
		httpLimit = config.HTTPLimit
	}
	httpCh := make(chan bool, httpLimit)

	mux.HandleFunc("/", http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// todo how to restrict broadcast request only ?
		remote := req.Header.Get("X-Forwarded-For")
		idx := strings.Index(remote, ",")
		if remote != "" && idx > -1 {
			remote = remote[:idx]
		} else if remote == "" {
			remote = req.Header.Get("X-Real-IP")
		}

		if err := api.CheckIp(remote); err != nil {
			return
		}

		select {
		case httpCh <- true:
			defer func() { <-httpCh }()
			wrappedGrpc.ServeHTTP(resp, req)
		}
	}) )

	go func() {
		http.ListenAndServe(config.HTTPListen, mux)
	}()

	return nil
}

