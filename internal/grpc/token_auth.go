package grpc

import (
	"context"
	"crypto/subtle"
	"errors"
)

// ExtractBearerToken reads the "authorization" gRPC metadata key and strips the "Bearer " prefix.
func ExtractBearerToken(ctx context.Context) string {
	return extractBearerToken(ctx)
}

// ValidateBearerToken compares provided against expected using constant-time comparison.
func ValidateBearerToken(provided, expected string) error {
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return errors.New("grpc: invalid bearer token")
	}
	return nil
}
