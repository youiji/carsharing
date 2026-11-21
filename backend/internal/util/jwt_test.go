package util

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-test-secret-test-secret!"

func TestTokenRoundTrip(t *testing.T) {
	m := NewTokenManager(testSecret, time.Minute)
	token, err := m.Issue(42, "client")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("свой токен не принят: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "client" {
		t.Errorf("claims = %+v", claims)
	}
}

func TestTokenTampered(t *testing.T) {
	m := NewTokenManager(testSecret, time.Minute)
	token, _ := m.Issue(42, "client")
	parts := strings.Split(token, ".")
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":42,"role":"admin","exp":9999999999}`))
	forged := parts[0] + "." + payload + "." + parts[2]
	if _, err := m.Parse(forged); err == nil {
		t.Fatal("токен с подменённой ролью принят")
	}
}

func TestTokenWrongSecret(t *testing.T) {
	token, _ := NewTokenManager(testSecret, time.Minute).Issue(42, "client")
	other := NewTokenManager("another-secret-another-secret-12345", time.Minute)
	if _, err := other.Parse(token); err == nil {
		t.Fatal("токен с чужой подписью принят")
	}
}

func TestTokenExpired(t *testing.T) {
	m := NewTokenManager(testSecret, -time.Minute)
	token, _ := m.Issue(42, "client")
	if _, err := m.Parse(token); err == nil {
		t.Fatal("просроченный токен принят")
	}
}
