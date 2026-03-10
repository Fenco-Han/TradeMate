package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID        string `json:"user_id"`
	ActiveStoreID string `json:"active_store_id"`
	RoleCode      string `json:"role_code"`
	jwt.RegisteredClaims
}

type Service struct {
	secret []byte
	expiry time.Duration
}

func NewService(secret string, expiryHours int) *Service {
	if expiryHours <= 0 {
		expiryHours = 24
	}

	return &Service{
		secret: []byte(secret),
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

func (s *Service) Sign(userID, activeStoreID, roleCode string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:        userID,
		ActiveStoreID: activeStoreID,
		RoleCode:      roleCode,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *Service) Parse(token string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(_ *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}

	if claims.UserID == "" || claims.ActiveStoreID == "" {
		return Claims{}, errors.New("invalid token claims")
	}

	return *claims, nil
}
