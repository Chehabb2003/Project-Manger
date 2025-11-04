package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"project-crypto/internal/vault"
)

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	v, err := s.withSessionVault(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		typ := r.URL.Query().Get("type")
		q := vault.Query{Type: typ}

		metas, err := v.List(r.Context(), q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		out := make([]map[string]any, 0, len(metas))
		for _, m := range metas {
			it, err := v.GetItem(r.Context(), m.ID)
			if err != nil {
				continue
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
		if strings.TrimSpace(it.Type) == "" {
			http.Error(w, "type required", http.StatusBadRequest)
			return
		}
		if it.Fields == nil || strings.TrimSpace(it.Fields["password"]) == "" {
			http.Error(w, "fields.password required", http.StatusBadRequest)
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

func (s *Server) handleItemByID(w http.ResponseWriter, r *http.Request) {
	v, err := s.withSessionVault(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/items/")
	if id == "" || id == "/" {
		http.NotFound(w, r)
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
