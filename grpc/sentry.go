package grpc

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"google.golang.org/grpc"
)

func SentryUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		sentryTxn := sentry.StartTransaction(ctx, fmt.Sprintf("gRPC:%s", info.FullMethod))
		defer sentryTxn.Finish()

		return handler(ctx, req)
	}
}
