package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTSigner struct {
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
	Iss  string        // issuer, e.g. "crypto-backend"
	TTL  time.Duration // e.g., 15 * time.Minute
}

func NewJWTSigner(priv ed25519.PrivateKey, iss string, ttl time.Duration) *JWTSigner {
	pub := priv.Public().(ed25519.PublicKey)
	return &JWTSigner{Priv: priv, Pub: pub, Iss: iss, TTL: ttl}
}

// FIX 1: return order (pub, priv, err) -> (priv, pub, err)
func GenerateEd25519() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return priv, pub, err
}

func (s *JWTSigner) IssueToken(sub string, roles []Role) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.TTL)

	claims := jwt.MapClaims{
		"iss":   s.Iss,
		"sub":   sub,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		"jti":   randomJTI(), // FIX 2 uses base64 below
		"roles": roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	ss, err := token.SignedString(s.Priv)
	return ss, exp, err
}

func (s *JWTSigner) ParseAndValidate(tokenStr string) (*Claims, error) {
	keyFunc := func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodEdDSA {
			return nil, errors.New("unexpected signing method")
		}
		return s.Pub, nil
	}

	tok, err := jwt.ParseWithClaims(
		tokenStr,
		jwt.MapClaims{},
		keyFunc,
		jwt.WithIssuer(s.Iss),
	)
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	std := tok.Claims.(jwt.MapClaims)

	getString := func(k string) string {
		if v, ok := std[k].(string); ok {
			return v
		}
		return ""
	}
	getInt64 := func(k string) int64 {
		switch v := std[k].(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		default:
			return 0
		}
	}
	var roles []Role
	if arr, ok := std["roles"].([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				roles = append(roles, Role(s))
			}
		}
	}

	return &Claims{
		Sub:       getString("sub"),
		Roles:     roles,
		TokenID:   getString("jti"),
		IssuedAt:  getInt64("iat"),
		ExpiresAt: getInt64("exp"),
	}, nil
}

// FIX 2: use base64url for a compact jti
func randomJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
