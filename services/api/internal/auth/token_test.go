package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestServiceSignAndParse(t *testing.T) {
	svc := NewService("secret-key", 2)

	token, err := svc.Sign("u_1", "store_1", "owner")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != "u_1" {
		t.Fatalf("unexpected user id: %s", claims.UserID)
	}
	if claims.ActiveStoreID != "store_1" {
		t.Fatalf("unexpected active store id: %s", claims.ActiveStoreID)
	}
	if claims.RoleCode != "owner" {
		t.Fatalf("unexpected role code: %s", claims.RoleCode)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now().UTC()) {
		t.Fatalf("expected valid expiry in future")
	}
}

func TestServiceParseInvalidToken(t *testing.T) {
	svc := NewService("secret-key", 2)

	if _, err := svc.Parse("not-a-jwt"); err == nil {
		t.Fatalf("expected parse error for invalid token")
	}
}

func TestServiceParseExpiredToken(t *testing.T) {
	svc := NewService("secret-key", 2)

	claims := Claims{
		UserID:        "u_1",
		ActiveStoreID: "store_1",
		RoleCode:      "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(-1 * time.Hour)),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString([]byte("secret-key"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := svc.Parse(token); err == nil {
		t.Fatalf("expected parse error for expired token")
	}
}

func TestServiceParseRejectsInvalidClaims(t *testing.T) {
	svc := NewService("secret-key", 2)

	token, err := svc.Sign("", "store_1", "owner")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.Parse(token); err == nil {
		t.Fatalf("expected parse error for missing user_id claim")
	}
}
