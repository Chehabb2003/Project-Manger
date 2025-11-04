package server

import "net/http"

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/health", s.handleHealth)

	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/login/verify", s.handleLoginVerify)
	s.mux.HandleFunc("/api/signup", s.handleSignup)
	s.mux.HandleFunc("/api/password/forgot", s.handleForgotPassword)
	s.mux.HandleFunc("/api/password/reset", s.handleResetPassword)

	s.mux.HandleFunc("/api/session", s.handleSession)
	s.mux.HandleFunc("/api/unlock", s.handleUnlock)
	s.mux.HandleFunc("/api/lock", s.handleLock)
	s.mux.HandleFunc("/api/password", s.handleChangePassword)
	s.mux.HandleFunc("/api/items", s.handleItems)
	s.mux.HandleFunc("/api/items/", s.handleItemByID)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
