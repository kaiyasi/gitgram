package githubwebhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"zen":"Keep it logically awesome."}`)
	signature := signBody(secret, body)

	if !VerifySignature(secret, body, signature) {
		t.Fatal("expected valid signature")
	}
	if VerifySignature("wrong-secret", body, signature) {
		t.Fatal("expected wrong secret to fail")
	}
	if VerifySignature(secret, body, "sha1=bad") {
		t.Fatal("expected wrong signature prefix to fail")
	}
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
