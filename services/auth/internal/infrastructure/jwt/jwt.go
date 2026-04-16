package jwt

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	privateKey     *rsa.PrivateKey
	accessTokenTTL time.Duration
}

type jwtClaims struct {
	UserId    string `json:"user_id"`
	CompanyId string `json:"company_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTService(keyPath string, accessTokenTTL time.Duration) (*JWTService, error) {
	pemBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &JWTService{privateKey: privateKey, accessTokenTTL: accessTokenTTL}, nil
}

func (s *JWTService) Generate(userId uuid.UUID, companyId uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserId:    userId.String(),
		CompanyId: companyId.String(),
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userId.String(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
}

func (s *JWTService) PublicKey() *rsa.PublicKey {
	return &s.privateKey.PublicKey
}
