package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"strings"

	"project-crypto/internal/auth"
	cr "project-crypto/internal/crypto"
	"project-crypto/internal/storage"
	"project-crypto/internal/vault"
)

type unlockReq struct {
	Master string `json:"master"`
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "no auth context", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	sess := s.sessions[claims.Sub]
	s.mu.Unlock()
	writeJSON(w, map[string]any{
		"user":     claims.Sub,
		"unlocked": sess != nil && sess.unlocked,
		"vault": func() string {
			if sess != nil {
				return path.Base(sess.vpath)
			}
			return ""
		}(),
	})
}

func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "no auth context", http.StatusUnauthorized)
		return
	}

	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	masterStr := strings.TrimSpace(req.Master)
	if masterStr == "" {
		http.Error(w, "master required", http.StatusBadRequest)
		return
	}

	master := []byte(masterStr)
	defer cr.Zero(master)

	metaColl, blobColl := collectionNames(claims.Sub)
	var blobs storage.BlobStore
	var err error
	if s.storageClient != nil {
		blobs, err = storage.NewMongoBlobStoreWithClient(s.storageClient, s.cfg.MongoDB, blobColl)
	} else {
		blobs, err = storage.NewMongoBlobStore(r.Context(), s.cfg.MongoURI, s.cfg.MongoDB, blobColl)
	}
	if err != nil {
		http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var meta *storage.MongoMetaStore
	if s.storageClient != nil {
		meta, err = storage.NewMongoMetaStoreWithClient(s.storageClient, s.cfg.MongoDB, metaColl)
	} else {
		meta, err = storage.NewMongoMetaStore(r.Context(), s.cfg.MongoURI, s.cfg.MongoDB, metaColl)
	}
	if err != nil {
		http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
		return
	}

	vpath := s.vaultPath(claims.Sub)
	v := vault.NewWithStores(vpath, blobs, meta)

	if _, statErr := os.Stat(vpath); errors.Is(statErr, os.ErrNotExist) {
		if err := v.Create(r.Context(), master); err != nil {
			http.Error(w, "create: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		if err := v.Unlock(r.Context(), master); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized)
			return
		}
	}

	s.mu.Lock()
	s.sessions[claims.Sub] = &userSession{v: v, vpath: vpath, unlocked: true}
	s.mu.Unlock()
	writeJSON(w, map[string]any{"ok": true, "vault": path.Base(vpath)})
}

func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "no auth context", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	if sess, ok := s.sessions[claims.Sub]; ok && sess.v != nil {
		sess.v.Lock()
	}
	delete(s.sessions, claims.Sub)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) withSessionVault(r *http.Request) (vault.Vault, error) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return nil, errors.New("no auth context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[claims.Sub]
	if sess == nil || !sess.unlocked || sess.v == nil {
		return nil, errors.New("vault not unlocked")
	}
	return sess.v, nil
}
