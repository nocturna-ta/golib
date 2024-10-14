package grpc

import (
	"context"
	"fmt"
	"golib/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

func UnaryServerPanicInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				var reqString string
				if req != nil {
					reqString = fmt.Sprintf("%+v", req)
				}

				log.WithFields(log.Fields{
					"err":         r,
					"full-method": info.FullMethod,
					"req":         reqString,
					"stacktrace":  string(stack),
				}).ErrorWithCtx(ctx, "[UnaryServerPanicInterceptor] panic on handling request")

				err = status.Error(codes.Internal, codes.Internal.String())
				return
			}
		}()

		return handler(ctx, req)
	}
}
