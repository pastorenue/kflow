// Package grpc provides the gRPC server, interceptors, and grpc-gateway mux.
package grpc

import (
	"context"
	"crypto/subtle"
	"log"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryAuthInterceptor validates the "authorization" metadata header.
// No-op when apiKey is empty (dev mode).
func UnaryAuthInterceptor(apiKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if apiKey == "" {
			return handler(ctx, req)
		}
		provided := extractBearerToken(ctx)
		if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			return nil, status.Error(codes.PermissionDenied, "invalid or missing API key")
		}
		return handler(ctx, req)
	}
}

// UnaryLoggingInterceptor logs method, duration, and status code for each RPC.
func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}
		log.Printf("grpc: method=%s duration=%s code=%s", info.FullMethod, time.Since(start), code)
		return resp, err
	}
}

// UnaryRecoveryInterceptor catches panics and returns gRPC INTERNAL.
func UnaryRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("grpc: PANIC in %s: %v\n%s", info.FullMethod, r, debug.Stack())
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func extractBearerToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ""
	}
	v := vals[0]
	const prefix = "Bearer "
	if len(v) > len(prefix) && v[:len(prefix)] == prefix {
		return v[len(prefix):]
	}
	return ""
}
