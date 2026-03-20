package runner

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("aaaabbbbccccddddeeeeffffgggghhhh") // 32 bytes

func TestGenerateAndValidateToken(t *testing.T) {
	tok, err := GenerateStateToken("exec-1", "ValidateOrder", 1, testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	payload, err := ValidateStateToken(tok, testSecret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if payload.ExecID != "exec-1" || payload.State != "ValidateOrder" || payload.Attempt != 1 {
		t.Fatalf("unexpected claims: ExecID=%q State=%q Attempt=%d", payload.ExecID, payload.State, payload.Attempt)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tok, _ := GenerateStateToken("exec-1", "StateA", 1, testSecret)
	_, err := ValidateStateToken(tok, []byte("wrong-secret-wrong-secret-wrong!"))
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_Tampered(t *testing.T) {
	tok, _ := GenerateStateToken("exec-1", "StateA", 1, testSecret)
	tampered := tok[:len(tok)-4] + "XXXX"
	_, err := ValidateStateToken(tampered, testSecret)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_MalformedNoDot(t *testing.T) {
	_, err := ValidateStateToken("nodottoken", testSecret)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// Craft a token with a past expiry by hand.
	claims := tokenClaims{
		ExecID:    "exec-exp",
		StateName: "StateX",
		Attempt:   1,
		ExpiresAt: time.Now().Add(-time.Hour).Unix(), // already expired
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := computeHMAC(testSecret, encoded)
	expiredTok := encoded + "." + sig

	_, err = ValidateStateToken(expiredTok, testSecret)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestTokenTTL(t *testing.T) {
	start := time.Now()
	tok, err := GenerateStateToken("e", "s", 1, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := ValidateStateToken(tok, testSecret)
	if err != nil {
		t.Fatalf("fresh token should be valid: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("token generation took too long")
	}
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		t.Fatal("expected two parts")
	}
	if payload.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero ExpiresAt")
	}
}
