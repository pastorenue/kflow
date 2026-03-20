// Package runner implements state token generation and validation for K8s Job
// containers. Tokens are HMAC-SHA256 signed and carry (execID, stateName,
// attempt, expiry) so the RunnerServiceServer can authorise callbacks.
package runner

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const tokenTTL = 24 * time.Hour

var ErrInvalidToken = errors.New("runner: invalid state token")

// TokenPayload contains the verified claims from a state token.
type TokenPayload struct {
	ExecID    string    `json:"exec_id"`
	State     string    `json:"state"`
	Attempt   int       `json:"attempt"`
	ExpiresAt time.Time `json:"expires_at"`
}

type tokenClaims struct {
	ExecID    string `json:"eid"`
	StateName string `json:"sn"`
	Attempt   int    `json:"att"`
	ExpiresAt int64  `json:"exp"`
}

// GenerateStateToken creates an HMAC-SHA256 signed token authorising a single
// (execID, stateName, attempt) execution. The token is valid for 24 hours.
// secret must be at least 32 bytes.
func GenerateStateToken(execID, stateName string, attempt int, secret []byte) (string, error) {
	if len(secret) < 32 {
		return "", fmt.Errorf("runner: token secret must be at least 32 bytes")
	}
	claims := tokenClaims{
		ExecID:    execID,
		StateName: stateName,
		Attempt:   attempt,
		ExpiresAt: time.Now().Add(tokenTTL).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("runner: marshal token claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := computeHMAC(secret, encoded)
	return encoded + "." + sig, nil
}

// ValidateStateToken parses and verifies a state token. Returns the embedded
// TokenPayload on success. Uses constant-time comparison to prevent timing attacks.
func ValidateStateToken(token string, secret []byte) (TokenPayload, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return TokenPayload{}, ErrInvalidToken
	}
	encoded, sig := parts[0], parts[1]

	expected := computeHMAC(secret, encoded)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return TokenPayload{}, ErrInvalidToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return TokenPayload{}, ErrInvalidToken
	}

	var claims tokenClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return TokenPayload{}, ErrInvalidToken
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return TokenPayload{}, ErrInvalidToken
	}

	return TokenPayload{
		ExecID:    claims.ExecID,
		State:     claims.StateName,
		Attempt:   claims.Attempt,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0),
	}, nil
}

func computeHMAC(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
