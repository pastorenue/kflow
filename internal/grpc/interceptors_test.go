package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryAuthInterceptor_ValidToken(t *testing.T) {
	interceptor := UnaryAuthInterceptor("secret-key")
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	md := metadata.Pairs("authorization", "Bearer secret-key")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestUnaryAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := UnaryAuthInterceptor("secret-key")
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	md := metadata.Pairs("authorization", "Bearer wrong-key")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, info, handler)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}

func TestUnaryAuthInterceptor_NoAPIKey(t *testing.T) {
	interceptor := UnaryAuthInterceptor("") // dev mode
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	_, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("expected no error in dev mode, got %v", err)
	}
}

func TestUnaryRecoveryInterceptor_Panic(t *testing.T) {
	interceptor := UnaryRecoveryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	}

	_, err := interceptor(context.Background(), nil, info, handler)
	if err == nil {
		t.Fatal("expected error after panic")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal code after panic, got %v", status.Code(err))
	}
}
