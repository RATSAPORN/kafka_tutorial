package services

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents the identity and role information extracted from a
// validated JWT. This is the ONLY source of truth for who the caller is —
// it must never be supplemented or overridden by client-supplied headers.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService interface {
	// ValidateToken parses and verifies a JWT, returning the embedded
	// claims if (and only if) the signature, expiry, and issuer all check
	// out. The token string should NOT include the "Bearer " prefix —
	// callers are responsible for stripping it.
	ValidateToken(tokenString string) (*Claims, error)
}

type authService struct {
	signingKey []byte
	issuer     string
}

// NewAuthService constructs an AuthService backed by HMAC-signed JWTs.
// signingKey must be the same secret used to issue tokens elsewhere in
// the system (e.g. at login). issuer is checked against the token's
// "iss" claim to reject tokens minted by anything else.
func NewAuthService(signingKey []byte, issuer string) AuthService {
	return &authService{
		signingKey: signingKey,
		issuer:     issuer,
	}
}

func (a *authService) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			// Reject anything not signed with the algorithm we expect.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return a.signingKey, nil
		},
		jwt.WithIssuer(a.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
	if err != nil {
		log.Printf("JWT validation error: %v", err)
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.UserID == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// NewToken is a helper for minting tokens (e.g. in a login handler or in
// Postman/tests) — not used by ValidateToken itself.
func NewToken(signingKey []byte, issuer, userID, role string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}
