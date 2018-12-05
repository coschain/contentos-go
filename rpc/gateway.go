package rpc

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/coschain/contentos-go/iservices/service-configs"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"net/http"
)

const (
	// DefaultHTTPLimit default max http conns
	DefaultHTTPLimit = 128
	// MaxGateWayRecvMsgSize Deafult max message size  gateway's grpc client can receive
	MaxGateWayRecvMsgSize = 64 * 1024 * 1024
	//GRPCEndpointName      = "rpc"
	GRPCEndpointUsage     = ""
)

func Run(config *service_configs.GRPCConfig) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}),
		runtime.WithProtoErrorHandler(errorHandler))
	opts := []grpc.DialOption{grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGateWayRecvMsgSize))}

	endpoint := flag.String(config.RPCName, config.RPCListen, GRPCEndpointUsage)

	grpcpb.RegisterApiServiceHandlerFromEndpoint(ctx, mux, *endpoint, opts)

	err := http.ListenAndServe(config.HTTPListen, allowCORS(mux, config))
	if err != nil {
		return err
	}

	return nil
}

func allowCORS(h http.Handler, config *service_configs.GRPCConfig) http.Handler {
	httpLimit := 128
	if httpLimit == config.HTTPLimit {
		httpLimit = DefaultHTTPLimit
	}
	httpCh := make(chan bool, httpLimit)

	c := cors.New(cors.Options{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE"},
		AllowedOrigins: config.HTTPCors,
		MaxAge:         600,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case httpCh <- true:
			defer func() { <-httpCh }()
			if len(config.HTTPCors) == 0 {
				h.ServeHTTP(w, r)
			} else {
				c.Handler(h).ServeHTTP(w, r)
			}
		default:
			statusUnavailableHandler(w, r)
		}
	})
}

func statusUnavailableHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("{\"err:\",\"Sorry, we received too many simultaneous requests.\nPlease try again later.\"}"))
}

type errorBody struct {
	Err string `json:"error,omitempty"`
}

func errorHandler(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	const fallback = "failed to marshal error message"

	w.Header().Set("Content-type", marshaler.ContentType())
	if grpc.Code(err) == codes.Unknown {
		w.WriteHeader(runtime.HTTPStatusFromCode(codes.OutOfRange))
	} else {
		w.WriteHeader(runtime.HTTPStatusFromCode(grpc.Code(err)))
	}
	jErr := json.NewEncoder(w).Encode(errorBody{
		Err: grpc.ErrorDesc(err),
	})

	if jErr != nil {
		jsonFallback, tmpErr := json.Marshal(errorBody{Err: fallback})
		if tmpErr != nil {
			//logging.VLog().WithFields(logrus.Fields{
			//	"error":        tmpErr,
			//	"jsonFallback": jsonFallback,
			//}).Debug("Failed to marshal fallback msg")
		}
		_, tmpErr = w.Write(jsonFallback)
		if tmpErr != nil {
			//logging.VLog().WithFields(logrus.Fields{
			//	"error": tmpErr,
			//}).Debug("Failed to write fallback msg")
		}
	}
}
