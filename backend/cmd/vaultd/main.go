// backend/cmd/vaultd/main.go
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"project-crypto/internal/storage"
	"project-crypto/internal/vault"
)

// Default MongoDB URI (can be overridden with --mongo or MONGODB_URI)
const defaultMongoURI = "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true"

type server struct {
	mu        sync.Mutex
	v         vault.Vault
	vpath     string
	unlocked  bool

	mongoURI  string
	mongoDB   string
	mongoMeta string
	mongoBlob string
}

func main() {
	port := flag.String("port", "8080", "HTTP port to listen on")
	mongoURI := flag.String("mongo", "", "MongoDB URI (or use MONGODB_URI env, or default)")
	mongoDB := flag.String("db", "vaultdb", "MongoDB database name")
	mongoMeta := flag.String("meta", "meta", "MongoDB collection for metadata")
	mongoBlob := flag.String("blobs", "blobs", "MongoDB collection for blobs")
	flag.Parse()

	if *mongoURI == "" {
		if env := os.Getenv("MONGODB_URI"); env != "" {
			*mongoURI = env
		} else {
			*mongoURI = defaultMongoURI
		}
	}

	s := &server{
		mongoURI:  *mongoURI,
		mongoDB:   *mongoDB,
		mongoMeta: *mongoMeta,
		mongoBlob: *mongoBlob,
	}

	mux := http.NewServeMux()

	// Health
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

	// Session state (authoritative)
	mux.HandleFunc("/api/session", s.handleSession)

	// Session actions
	mux.HandleFunc("/api/unlock", s.handleUnlock) // POST {vault, master}
	mux.HandleFunc("/api/lock", s.handleLock)     // POST

	// Items
	mux.HandleFunc("/api/items", s.handleItems)     // GET / POST
	mux.HandleFunc("/api/items/", s.handleItemByID) // GET / PUT / DELETE

	log.Printf("HTTP on :%s (mongo=enabled db=%s meta=%s blobs=%s)", *port, s.mongoDB, s.mongoMeta, s.mongoBlob)
	log.Fatal(http.ListenAndServe(":"+*port, mux))
}

type unlockReq struct {
	VaultPath string `json:"vault"`
	Master    string `json:"master"`
}

func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	unlocked := s.unlocked
	name := path.Base(s.vpath)
	s.mu.Unlock()
	writeJSON(w, map[string]any{"unlocked": unlocked, "vault": name})
}

func (s *server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req unlockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.VaultPath == "" {
		req.VaultPath = "./main.vlt"
	}

	// Build Mongo-backed stores
	blobs, err := storage.NewMongoBlobStore(r.Context(), s.mongoURI, s.mongoDB, s.mongoBlob)
	if err != nil {
		http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	meta, err := storage.NewMongoMetaStore(r.Context(), s.mongoURI, s.mongoDB, s.mongoMeta)
	if err != nil {
		http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
		return
	}
	v := vault.NewWithStores(req.VaultPath, blobs, meta)

	// Create or unlock
	if _, err := os.Stat(req.VaultPath); errors.Is(err, os.ErrNotExist) {
		if err := v.Create(r.Context(), []byte(req.Master)); err != nil {
			http.Error(w, "create: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		if err := v.Unlock(r.Context(), []byte(req.Master)); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized)
			return
		}
	}

	s.mu.Lock()
	s.v = v
	s.vpath = req.VaultPath
	s.unlocked = true
	s.mu.Unlock()

	writeJSON(w, map[string]any{"ok": true, "vault": path.Base(req.VaultPath)})
}

func (s *server) handleLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	if s.v != nil {
		s.v.Lock()
	}
	s.unlocked = false
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func last4Digits(s string) string {
	d := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			d = append(d, r)
		}
	}
	if len(d) <= 4 {
		return string(d)
	}
	return string(d[len(d)-4:])
}

func canonType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "secure note", "secure-note":
		return "note"
	default:
		return t
	}
}

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		typ := r.URL.Query().Get("type")
		q := vault.Query{Type: typ}

		s.mu.Lock()
		v := s.v
		s.mu.Unlock()
		if v == nil {
			http.Error(w, "vault not unlocked", http.StatusUnauthorized)
			return
		}

		metas, err := v.List(r.Context(), q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		out := make([]map[string]any, 0, len(metas))
		for _, m := range metas {
			it, err := v.GetItem(r.Context(), m.ID)
			if err != nil {
				continue // skip dangling
			}
			fields := map[string]string{}
			for k, val := range it.Fields {
				fields[k] = val
			}
			if _, ok := fields["username"]; !ok {
				if u, ok2 := fields["user"]; ok2 {
					fields["username"] = u
				}
			}
			if _, ok := fields["site"]; !ok || fields["site"] == "" {
				switch strings.ToLower(m.Type) {
				case "card":
					l4 := last4Digits(fields["number"])
					if l4 != "" {
						fields["site"] = "Card •••• " + l4
					} else {
						fields["site"] = "Card"
					}
				default:
					if t, ok2 := fields["title"]; ok2 && t != "" {
						fields["site"] = t
					} else if n, ok3 := fields["name"]; ok3 && n != "" {
						fields["site"] = n
					} else {
						fields["site"] = "(untitled)"
					}
				}
			}

			out = append(out, map[string]any{
				"id":      m.ID,
				"type":    canonType(m.Type),
				"created": m.Created,
				"updated": m.Updated,
				"version": m.Version,
				"fields":  fields,
			})
		}
		writeJSON(w, out)

	case http.MethodPost:
		var it vault.Item
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		v := s.v
		s.mu.Unlock()
		if v == nil {
			http.Error(w, "vault not unlocked", http.StatusUnauthorized)
			return
		}
		id, err := v.AddItem(r.Context(), it)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSONStatus(w, http.StatusCreated, map[string]string{"id": id})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleItemByID(w http.ResponseWriter, r *http.Request) {
	// Always use r.URL.Path
	id := strings.TrimPrefix(r.URL.Path, "/api/items/")
	if id == "" || id == "/" {
		http.NotFound(w, r)
		return
	}

	s.mu.Lock()
	v := s.v
	s.mu.Unlock()
	if v == nil {
		http.Error(w, "vault not unlocked", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		it, err := v.GetItem(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, it)

	case http.MethodPut:
		var patch vault.Item
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if err := v.UpdateItem(r.Context(), id, patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"updated": true})

	case http.MethodDelete:
		if err := v.DeleteItem(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
