package webui

import (
	"testing"
	"time"
)

func TestValidateJWT_ValidToken(t *testing.T) {
	secret := "test-secret-key"
	claims := JWTClaims{
		Subject:   "user1",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	result, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if result.Subject != "user1" {
		t.Errorf("expected subject 'user1', got %q", result.Subject)
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	secret := "test-secret-key"
	claims := JWTClaims{
		Subject:   "user1",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}

	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateJWT_InvalidSignature(t *testing.T) {
	claims := JWTClaims{
		Subject:   "user1",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := GenerateJWT("correct-secret", claims)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateJWT_MissingClaims(t *testing.T) {
	secret := "test-secret-key"
	claims := JWTClaims{
		Subject:   "", // missing
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err != ErrMissingClaims {
		t.Errorf("expected ErrMissingClaims, got %v", err)
	}
}

func TestValidateJWT_MalformedToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"no dots", "nodots"},
		{"one dot", "one.dot"},
		{"four parts", "a.b.c.d"},
		{"invalid base64", "!!!.!!!.!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.token, "secret")
			if err == nil {
				t.Error("expected error for malformed token")
			}
		})
	}
}

func TestValidateJWT_NoExpiration(t *testing.T) {
	secret := "test-secret-key"
	claims := JWTClaims{
		Subject:  "user1",
		IssuedAt: time.Now().Unix(),
		// ExpiresAt is 0 — no expiration
	}

	token, err := GenerateJWT(secret, claims)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	result, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("expected no error for token without expiration, got %v", err)
	}
	if result.Subject != "user1" {
		t.Errorf("expected subject 'user1', got %q", result.Subject)
	}
}
