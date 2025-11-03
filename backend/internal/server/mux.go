package server

import (
	"crypto/ed25519"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"project-crypto/internal/auth"
)

type Server struct {
	mux    *http.ServeMux
	signer *auth.JWTSigner
	users  auth.UserStore
}

func NewServer(priv ed25519.PrivateKey, users auth.UserStore) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		signer: auth.NewJWTSigner(priv, "crypto-backend", 15*time.Minute),
		users:  users,
	}
	// Public
	s.mux.Handle("/api/login", &LoginHandler{Users: users, Signer: s.signer})

	// Protected demo endpoints
	protected := auth.AuthRequired(s.signer)
	adminOnly := auth.RequireRole(auth.RoleAdmin)

	// /api/me : any authenticated user
	s.mux.Handle("/api/me", protected(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(claims)
	})))

	// /api/admin : only admins
	s.mux.Handle("/api/admin", protected(adminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("admin zone"))
	}))))

	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

// Helper to stand up a demo server quickly
func RunExample() {
	priv, _, err := auth.GenerateEd25519()
	if err != nil {
		log.Fatal(err)
	}

	users := auth.NewMemoryUserStore()
	// create a demo admin and user
	adminHash, _ := auth.HashPassword(auth.DefaultArgon, "admin123!")
	userHash, _ := auth.HashPassword(auth.DefaultArgon, "user123!")

	if err := users.Add(&auth.User{Username: "admin", Email: "admin@example.com", PassHash: adminHash, Roles: []auth.Role{auth.RoleAdmin, auth.RoleUser}}); err != nil {
		log.Fatal(err)
	}
	if err := users.Add(&auth.User{Username: "user", Email: "user@example.com", PassHash: userHash, Roles: []auth.Role{auth.RoleUser}}); err != nil {
		log.Fatal(err)
	}

	s := NewServer(priv, users)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", s.Handler()))
}
