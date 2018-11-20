package rpc

import (
	"github.com/coschain/contentos-go/common/logging"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
	"context"
)

func streamLoggingInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	logging.VLog().WithFields(logrus.Fields{
		"method": info.FullMethod,
	}).Info("Rpc request.")

	return handler(srv, ss)
}

func unaryLoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	if strings.Contains(info.FullMethod, "ApiService") {
		logging.VLog().WithFields(logrus.Fields{
			"method": info.FullMethod,
			"params": req,
		}).Info("Rpc request.")
	} else {
		logging.VLog().WithFields(logrus.Fields{
			"method": info.FullMethod,
		}).Info("Rpc request.")
	}

	return handler(ctx, req)
}

func streamRecoveryInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	defer func() {
		if err := recover(); err != nil {
			logging.CLog().Errorf("stream recovery interceptor err: [%x]", err)
		}
	}()

	return handler(srv, ss)
}

func unaryRecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if err := recover(); err != nil {
			logging.CLog().Errorf("unary recovery interceptor err: [%x]", err)
		}
	}()

	return handler(ctx, req)
}