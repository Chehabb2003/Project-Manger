package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type ctxKey int

const claimsKey ctxKey = 1

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}

type TokenParser interface {
	ParseAndValidate(tokenStr string) (*Claims, error)
}

// AuthRequired checks Bearer token and adds claims to context.
func AuthRequired(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			claims, err := parser.ParseAndValidate(token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

// RequireRole wraps a handler and ensures claim has given role.
func RequireRole(role Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, "no auth context", http.StatusUnauthorized)
				return
			}
			if !hasRole(claims, role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasRole(c *Claims, role Role) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Helper to extract user or fail early in handlers
func MustClaims(r *http.Request) (*Claims, error) {
	if c, ok := FromContext(r.Context()); ok {
		return c, nil
	}
	return nil, errors.New("no claims")
}
