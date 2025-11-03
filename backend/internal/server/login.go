package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"project-crypto/internal/auth"
)

type LoginHandler struct {
	Users  auth.UserStore
	Signer *auth.JWTSigner
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}
	var u *auth.User
	var err error
	if identifier != "" {
		u, err = h.Users.FindByUsername(identifier)
		if err != nil {
			u, err = h.Users.FindByEmail(identifier)
		}
	} else {
		err = errors.New("missing identifier")
	}
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	ok, err := auth.VerifyPassword(req.Password, u.PassHash)
	if err != nil || !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	tok, exp, err := h.Signer.IssueToken(u.Username, u.Roles)
	if err != nil {
		http.Error(w, "token issue failed", http.StatusInternalServerError)
		return
	}
	resp := auth.LoginResponse{Token: tok, ExpiresAt: exp}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
