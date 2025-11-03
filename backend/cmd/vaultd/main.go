package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"project-crypto/internal/vault"
)

type server struct {
	mu    sync.Mutex
	v     vault.Vault
	vpath string
}

func main() {
	s := &server{}
	mux := http.NewServeMux()

	// Health (both paths, handy for quick checks)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Session
	mux.HandleFunc("/api/unlock", s.handleUnlock) // POST {vault, master}
	mux.HandleFunc("/api/lock", s.handleLock)     // POST

	// Items
	mux.HandleFunc("/api/items", s.handleItems)     // GET(list) / POST(create)
	mux.HandleFunc("/api/items/", s.handleItemByID) // GET / PUT / DELETE

	log.Println("Dev HTTP on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type unlockReq struct {
	VaultPath string `json:"vault"`
	Master    string `json:"master"`
}

func (s *server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
	if req.VaultPath == "" { req.VaultPath = "./main.vlt" }

	s.mu.Lock()
	defer s.mu.Unlock()

	s.v = vault.New(req.VaultPath)
	s.vpath = req.VaultPath

	// If the vault file doesn't exist, create it; otherwise unlock it.
	if _, err := os.Stat(req.VaultPath); errors.Is(err, os.ErrNotExist) {
		if err := s.v.Create(context.Background(), []byte(req.Master)); err != nil {
			http.Error(w, "create: "+err.Error(), http.StatusBadRequest); return
		}
	} else {
		if err := s.v.Unlock(context.Background(), []byte(req.Master)); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized); return
		}
	}

	writeJSON(w, map[string]any{"ok": true, "vault": path.Base(req.VaultPath)})
}

func (s *server) handleLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.v != nil { s.v.Lock() }
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		typ := r.URL.Query().Get("type")
		q := vault.Query{Type: typ}
		s.mu.Lock(); v := s.v; s.mu.Unlock()
		if v == nil { http.Error(w, "vault not unlocked", http.StatusUnauthorized); return }
		list, err := v.List(context.Background(), q)
		if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
		writeJSON(w, list)
	case http.MethodPost:
		var it vault.Item
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
		s.mu.Lock(); v := s.v; s.mu.Unlock()
		if v == nil { http.Error(w, "vault not unlocked", http.StatusUnauthorized); return }
		id, err := v.AddItem(context.Background(), it)
		if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
		writeJSONStatus(w, http.StatusCreated, map[string]string{"id": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleItemByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/items/")
	if id == "" || id == "/" { http.NotFound(w, r); return }

	s.mu.Lock(); v := s.v; s.mu.Unlock()
	if v == nil { http.Error(w, "vault not unlocked", http.StatusUnauthorized); return }

	switch r.Method {
	case http.MethodGet:
		it, err := v.GetItem(context.Background(), id)
		if err != nil { http.Error(w, err.Error(), http.StatusNotFound); return }
		writeJSON(w, it)
	case http.MethodPut:
		var patch vault.Item
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil { http.Error(w, "bad json", http.StatusBadRequest); return }
		if err := v.UpdateItem(context.Background(), id, patch); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
		writeJSON(w, map[string]any{"updated": true})
	case http.MethodDelete:
		if err := v.DeleteItem(context.Background(), id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) { writeJSONStatus(w, http.StatusOK, v) }
func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
