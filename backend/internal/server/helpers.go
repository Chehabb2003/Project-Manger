package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"regexp"
	"strings"

	"project-crypto/internal/auth"
)

func writeJSON(w http.ResponseWriter, v any) {
	writeJSONStatus(w, http.StatusOK, v)
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func tooMany(w http.ResponseWriter, retryAfterSeconds int) {
	if retryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	}
	http.Error(w, "too many requests", http.StatusTooManyRequests)
}

var (
	reUpper = regexp.MustCompile(`[A-Z]`)
	reLower = regexp.MustCompile(`[a-z]`)
	reDigit = regexp.MustCompile(`[0-9]`)
	reSym   = regexp.MustCompile(`[^A-Za-z0-9]`)
	reEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func validatePassword(pw string) error {
	switch {
	case len(pw) < 12:
		return errors.New("password must be at least 12 characters")
	case strings.Contains(pw, " "):
		return errors.New("password must not contain spaces")
	case !reUpper.MatchString(pw):
		return errors.New("password must include an uppercase letter")
	case !reLower.MatchString(pw):
		return errors.New("password must include a lowercase letter")
	case !reDigit.MatchString(pw):
		return errors.New("password must include a digit")
	case !reSym.MatchString(pw):
		return errors.New("password must include a special character")
	default:
		return nil
	}
}

func isValidEmail(email string) bool {
	return reEmail.MatchString(email)
}

func sha256Hex(in string) string {
	sum := sha256.Sum256([]byte(in))
	return hex.EncodeToString(sum[:16])
}

func collectionNames(username string) (meta, blobs string) {
	sum := sha256.Sum256([]byte(username))
	short := hex.EncodeToString(sum[:6])
	return "meta_" + short, "blobs_" + short
}

func roleNames(rs []auth.Role) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = string(r)
	}
	return out
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
