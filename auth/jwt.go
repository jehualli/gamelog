// Package auth provides JWT token generation/validation and Casbin RBAC.
//
// LEARN — What is JWT?
// A JWT (JSON Web Token) is a compact, self-contained token with three parts:
//
//   Header.Payload.Signature
//   eyJhbG...  .  eyJ1c2Vy...  .  SflKxwR...
//
// Header  → algorithm used (HS256)
// Payload → the claims: who the user is, when token expires (readable by anyone!)
// Signature → HMAC of header+payload using your secret (only you can create/verify)
//
// The server never stores tokens. It just verifies the signature on every request.
// If the signature is valid and the token isn't expired, the user is authenticated.
package auth

import (
	"errors"
	"time"

	"gamelog/config"
	"gamelog/models"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the token payload. We embed jwt.RegisteredClaims for standard fields
// (ExpiresAt, IssuedAt) and add our own (UserID, Username, Role).
// LEARN: Claims are not encrypted — only signed. Don't put passwords or secrets here.
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user.
// The token is signed with HS256 (HMAC-SHA256) using our JWT_SECRET.
func GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gamelog",
		},
	}

	// NewWithClaims wraps claims in a token object (not yet signed).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// SignedString finalizes the token: encodes header+payload as base64, then signs.
	return token.SignedString([]byte(config.App.JWTSecret))
}

// ValidateToken parses a JWT string and returns the claims if valid.
// Returns an error if the token is expired, tampered with, or malformed.
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// LEARN: Always check the signing method — an attacker could send a token
		// signed with "none" algorithm hoping the server skips verification.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.App.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
