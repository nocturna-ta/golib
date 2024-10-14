package grpc

import (
	"context"
	"github.com/google/uuid"
	libCtx "github.com/nocturna-ta/golib/context"
	"github.com/nocturna-ta/golib/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RequestContextServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		headers, ok := metadata.FromIncomingContext(ctx)

		if !ok {
			return nil, status.Error(codes.Internal, "Error while reading the context")
		}

		reqCtx := readMetadataToRequestContext(headers)

		ctx = context.WithValue(ctx, libCtx.RequestIdKey, reqCtx.RequestId)
		ctx = context.WithValue(ctx, libCtx.RequestContextKey, reqCtx)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Missing request context")
			return nil, err
		}

		return handler(ctx, req)
	}
}

func RequestContextClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		send, _ := metadata.FromOutgoingContext(ctx)

		reqCtx, err := libCtx.GetRequestContext(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Missing request context")
			return err
		}

		newMD := buildMetadataFromReqCtx(reqCtx)

		ctx = metadata.NewOutgoingContext(ctx, metadata.Join(send, newMD))

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func readMetadataToRequestContext(md metadata.MD) libCtx.RequestContext {
	var userId, requestId, channelId, accountId string

	userId = md.Get(libCtx.XUserId)[0]
	requestId = md.Get(libCtx.XRequestId)[0]
	if requestId == "" {
		requestId = uuid.New().String()
	}

	channelId = md.Get(libCtx.XChannelId)[0]
	accountId = md.Get(libCtx.XAccountId)[0]

	return libCtx.RequestContext{
		UserId:    userId,
		RequestId: requestId,
		ChannelId: channelId,
		AccountId: accountId,
	}
}

func buildMetadataFromReqCtx(reqCtx *libCtx.RequestContext) metadata.MD {
	newMD := make(metadata.MD, 0)

	newMD.Set(libCtx.XUserId, reqCtx.UserId)
	newMD.Set(libCtx.XRequestId, reqCtx.RequestId)
	newMD.Set(libCtx.XChannelId, reqCtx.ChannelId)
	newMD.Set(libCtx.XAccountId, reqCtx.AccountId)

	return newMD
}
