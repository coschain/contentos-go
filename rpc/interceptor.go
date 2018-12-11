package rpc

import (
	"context"
	"github.com/coschain/contentos-go/iservices"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
)

type GRPCIntercepter struct {
	log       *logrus.Logger
}

func NewGRPCIntercepter(ilog iservices.ILog) *GRPCIntercepter {
	return &GRPCIntercepter{log: ilog.GetLog()}
}

func (gi *GRPCIntercepter) streamRecoveryLoggingInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {

	defer func() {
		if r := recover(); r != nil {
			gi.log.Errorf("stream recovery interceptor err: [%v]", r)
			err = ErrPanicResp
		}
	}()

	gi.log.WithFields(logrus.Fields{
		"method": info.FullMethod,
	}).Info("Rpc request.")

	return handler(srv, ss)
}

func (gi *GRPCIntercepter) unaryRecoveryLoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			gi.log.Errorf("unary recovery interceptor err: [%v]", r)
			err = ErrPanicResp
		}
	}()

	if strings.Contains(info.FullMethod, "ApiService") {
		gi.log.WithFields(logrus.Fields{
			"method": info.FullMethod,
			"params": req,
		}).Info("Rpc request.")
	} else {
		gi.log.WithFields(logrus.Fields{
			"method": info.FullMethod,
		}).Info("Rpc request.")
	}

	return handler(ctx, req)
}
