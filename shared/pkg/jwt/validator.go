package jwt

import (
	"crypto/rsa"
	"fmt"
	"shared/pkg/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Validator struct {
	publicKey *rsa.PublicKey
}

type jwtClaims struct {
	UserId    string `json:"user_id"`
	CompanyId string `json:"company_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func NewValidatorFromKey(pub *rsa.PublicKey) *Validator {
	return &Validator{publicKey: pub}
}

func NewValidator(publicKeyPEM []byte) (*Validator, error) {
	pub, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return &Validator{publicKey: pub}, nil
}

func (v *Validator) Validate(tokenString string) (*auth.AccessClaims, error) {
	claims := &jwtClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	userID, err := uuid.Parse(claims.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}
	companyID, err := uuid.Parse(claims.CompanyId)
	if err != nil {
		return nil, fmt.Errorf("invalid company_id: %w", err)
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("invalid exp: %w", err)
	}

	return &auth.AccessClaims{
		TokenID:   claims.ID,
		UserID:    userID,
		CompanyID: companyID,
		Role:      claims.Role,
		ExpiresAt: exp.Time,
	}, nil
}
