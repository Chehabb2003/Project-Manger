package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"project-crypto/internal/auth"
	"project-crypto/internal/storage"
	"project-crypto/internal/vault"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Default MongoDB URI (can be overridden with --mongo or MONGODB_URI)
const defaultMongoURI = "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true"

type server struct {
	// vault sessions keyed by username (from JWT "sub")
	mu          sync.Mutex
	sessions    map[string]*userSession
	resetTokens map[string]resetToken
	pending2FA  map[string]*twoFAChallenge
	mailer      mailConfig

	// mongo config (defaults; per-user collections are derived dynamically)
	mongoURI  string
	mongoDB   string
	mongoMeta string
	mongoBlob string

	// router + auth
	mux    *http.ServeMux
	users  auth.UserStore
	signer *auth.JWTSigner
}

type userSession struct {
	v        vault.Vault
	vpath    string
	unlocked bool
}

type resetToken struct {
	Username string
	Email    string
	Expires  time.Time
}

type twoFAChallenge struct {
	Username string
	Email    string
	Roles    []auth.Role
	Password string
	Code     string
	Expires  time.Time
}

type mailConfig struct {
	Host     string
	Port     string
	User     string
	Pass     string
	From     string
	enabled  bool
	security string
}

// ---- ServeHTTP: central middleware + routing ----
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Panic safety
	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}()

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// JSON default for /api/*
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	// Public allowlist
	switch r.URL.Path {
	case "/health", "/api/health", "/api/login", "/api/signup", "/api/password/forgot", "/api/password/reset":
		s.mux.ServeHTTP(w, r)
		return
	}

	// JWT gate for /api/*
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) })
		auth.AuthRequired(s.signer)(h).ServeHTTP(w, r)
		return
	}

	// default
	s.mux.ServeHTTP(w, r)
}

// ---- boot + wiring ----
func main() {
	// make logs visible and include file:line
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_ = godotenv.Load(".env", "backend/.env")
	port := flag.String("port", "8080", "HTTP port to listen on")
	mongoURI := flag.String("mongo", "", "MongoDB URI (or use MONGODB_URI env, or default)")
	mongoDB := flag.String("db", "vaultdb", "MongoDB database name")
	mongoMeta := flag.String("meta", "meta", "MongoDB collection for metadata (default only)")
	mongoBlob := flag.String("blobs", "blobs", "MongoDB collection for blobs (default only)")
	flag.Parse()

	if *mongoURI == "" {
		if env := os.Getenv("MONGODB_URI"); env != "" {
			*mongoURI = env
		} else {
			*mongoURI = defaultMongoURI
		}
	}

	s := &server{
		mongoURI:    *mongoURI,
		mongoDB:     *mongoDB,
		mongoMeta:   *mongoMeta,
		mongoBlob:   *mongoBlob,
		sessions:    make(map[string]*userSession),
		resetTokens: make(map[string]resetToken),
		pending2FA:  make(map[string]*twoFAChallenge),
	}
	s.mailer = loadMailConfig()

	s.initAuth()
	s.routes()

	if err := os.MkdirAll("./vaults", 0o700); err != nil {
		log.Fatalf("mkdir vaults: %v", err)
	}

	log.Printf("HTTP on :%s (mongo db=%s)", *port, s.mongoDB)
	log.Fatal(http.ListenAndServe(":"+*port, s)) // s implements http.Handler
}

// ---- routes registration ----
func (s *server) routes() {
	s.mux = http.NewServeMux()

	// Health (public)
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Auth (public)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/login/verify", s.handleLoginVerify)
	s.mux.HandleFunc("/api/signup", s.handleSignup)
	s.mux.HandleFunc("/api/password/forgot", s.handleForgotPassword)
	s.mux.HandleFunc("/api/password/reset", s.handleResetPassword)

	// Protected endpoints (JWT gate happens in ServeHTTP)
	s.mux.HandleFunc("/api/session", s.handleSession)
	// Unlock is mostly redundant now (login/signup auto-unlock), but kept for completeness
	s.mux.HandleFunc("/api/unlock", s.handleUnlock)           // POST {master}
	s.mux.HandleFunc("/api/lock", s.handleLock)               // POST
	s.mux.HandleFunc("/api/password", s.handleChangePassword) // PUT {current,next}
	s.mux.HandleFunc("/api/items", s.handleItems)             // GET / POST
	s.mux.HandleFunc("/api/items/", s.handleItemByID)
}

// ---- auth bootstrapping ----
func (s *server) initAuth() {
	// Ed25519 signer (load from env/file later if needed)
	priv, _, err := auth.GenerateEd25519()
	if err != nil {
		log.Fatalf("ed25519: %v", err)
	}
	s.signer = auth.NewJWTSigner(priv, "crypto-backend", 15*time.Minute)

	// Users: Mongo-backed
	ctx := context.Background()
	ustore, err := auth.NewMongoUserStore(ctx, s.mongoURI, s.mongoDB, "users")
	if err != nil {
		log.Fatalf("mongo users: %v", err)
	}
	s.users = ustore

	// Optional: seed admin/user only if missing
	adminUser := getenvDefault("ADMIN_USER", "admin")
	adminPass := getenvDefault("ADMIN_PASS", "admin123!")
	adminEmail := strings.TrimSpace(strings.ToLower(getenvDefault("ADMIN_EMAIL", adminUser+"@example.com")))
	userUser := getenvDefault("APP_USER", "user")
	userPass := getenvDefault("APP_PASS", "user123!")
	userEmail := strings.TrimSpace(strings.ToLower(getenvDefault("APP_EMAIL", userUser+"@example.com")))

	// Admin
	if _, err := s.users.FindByUsername(adminUser); err != nil {
		hash, hErr := auth.HashPassword(auth.DefaultArgon, adminPass)
		if hErr != nil {
			log.Fatalf("hash admin: %v", hErr)
		}
		if addErr := s.users.Add(&auth.User{
			Username: adminUser,
			Email:    adminEmail,
			PassHash: hash,
			Roles:    []auth.Role{auth.RoleAdmin, auth.RoleUser},
		}); addErr != nil {
			log.Fatalf("seed admin: %v", addErr)
		}
		log.Printf("seeded admin user: %s", adminUser)
	}

	// Regular user
	if _, err := s.users.FindByUsername(userUser); err != nil {
		hash, hErr := auth.HashPassword(auth.DefaultArgon, userPass)
		if hErr != nil {
			log.Fatalf("hash user: %v", hErr)
		}
		if addErr := s.users.Add(&auth.User{
			Username: userUser,
			Email:    userEmail,
			PassHash: hash,
			Roles:    []auth.Role{auth.RoleUser},
		}); addErr != nil {
			log.Fatalf("seed user: %v", addErr)
		}
		log.Printf("seeded regular user: %s", userUser)
	}

	log.Printf("auth ready (mongo users), token_ttl=%s", s.signer.TTL)
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func loadMailConfig() mailConfig {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := os.Getenv("SMTP_PASS")
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	security := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_SECURITY")))
	if security == "" {
		security = "starttls"
	}

	if host == "" || from == "" {
		log.Printf("[mailer] SMTP_HOST or SMTP_FROM missing; password reset emails disabled (link will be logged)")
		return mailConfig{}
	}
	if port == "" {
		port = "587"
	}
	if user == "" || pass == "" {
		log.Printf("[mailer] SMTP_USER or SMTP_PASS missing; attempting unauthenticated SMTP to %s:%s", host, port)
	}
	cfg := mailConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Pass:     pass,
		From:     from,
		enabled:  true,
		security: security,
	}
	log.Printf("[mailer] configured host=%s port=%s security=%s user=%s", host, port, security, maskForLog(user))
	return cfg
}

// ---- helpers ----
func writeJSON(w http.ResponseWriter, v any) { writeJSONStatus(w, http.StatusOK, v) }
func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func userVaultPath(username string) string {
	sum := sha256.Sum256([]byte(username))
	// Use first 16 bytes -> 32 hex chars (shorter file names)
	name := hex.EncodeToString(sum[:16]) + ".vlt"
	return filepath.Join("./vaults", name)
}

// derive per-user collection names (meta_xxx, blobs_xxx)
func userCollections(username string) (metaColl string, blobColl string) {
	sum := sha256.Sum256([]byte(username))
	short := hex.EncodeToString(sum[:6]) // 6 bytes → 12 hex chars
	metaColl = "meta_" + short
	blobColl = "blobs_" + short
	return
}

// Password policy: >=12 chars, at least 1 upper, 1 lower, 1 digit, 1 symbol, no spaces.
var (
	reUpper = regexp.MustCompile(`[A-Z]`)
	reLower = regexp.MustCompile(`[a-z]`)
	reDigit = regexp.MustCompile(`[0-9]`)
	reSym   = regexp.MustCompile(`[^A-Za-z0-9]`)
	reEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func validatePassword(pw string) error {
	if len(pw) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	if strings.Contains(pw, " ") {
		return errors.New("password must not contain spaces")
	}
	if !reUpper.MatchString(pw) {
		return errors.New("password must include an uppercase letter")
	}
	if !reLower.MatchString(pw) {
		return errors.New("password must include a lowercase letter")
	}
	if !reDigit.MatchString(pw) {
		return errors.New("password must include a digit")
	}
	if !reSym.MatchString(pw) {
		return errors.New("password must include a special character")
	}
	return nil
}

func maskForLog(s string) string {
	if s == "" {
		return "(none)"
	}
	if len(s) <= 2 {
		return "***"
	}
	return s[:1] + "***" + s[len(s)-1:]
}

func isValidEmail(email string) bool {
	return reEmail.MatchString(email)
}

// ---- DTOs ----
type loginReq struct {
	Username   string `json:"username"`
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}
type loginResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Vault     string    `json:"vault"`
}

type signupReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}
type signupResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Vault     string    `json:"vault"`
	Note      string    `json:"note"`
}

type unlockReq struct {
	Master string `json:"master"`
}

type changePasswordReq struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

type changePasswordResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Vault     string    `json:"vault"`
	Note      string    `json:"note,omitempty"`
}

type forgotPasswordReq struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type forgotPasswordResp struct {
	Note string `json:"note"`
}

type resetPasswordReq struct {
	Token string `json:"token"`
	Next  string `json:"next"`
}

type resetPasswordResp struct {
	Note string `json:"note"`
}

type twoFAChallengeResp struct {
	ChallengeID string    `json:"challenge_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	Note        string    `json:"note"`
}

type loginVerifyReq struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

// ---- handlers: signup/login ----
func (s *server) handleSignup(w http.ResponseWriter, r *http.Request) {
	log.Println("[signup] request received")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req signupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[signup] JSON decode error: %v\n", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	log.Printf("[signup] raw input -> username=%q, password length=%d\n", req.Username, len(req.Password))

	if req.Username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}
	if !isValidEmail(req.Email) {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}
	if err := validatePassword(req.Password); err != nil {
		http.Error(w, "weak password: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Optional pre-check (nice 409 instead of generic error)
	if _, err := s.users.FindByUsername(req.Username); err == nil {
		http.Error(w, "username already exists", http.StatusConflict)
		return
	}
	if _, err := s.users.FindByEmail(req.Email); err == nil {
		http.Error(w, "email already exists", http.StatusConflict)
		return
	}

	// Hash and persist user in Mongo (implements UserStore)
	hash, err := auth.HashPassword(auth.DefaultArgon, req.Password)
	if err != nil {
		http.Error(w, "hash password failed", http.StatusInternalServerError)
		return
	}
	if err := s.users.Add(&auth.User{
		Username: req.Username,
		Email:    req.Email,
		PassHash: hash,
		Roles:    []auth.Role{auth.RoleUser},
	}); err != nil {
		// If unique index triggers, this still returns a duplicate error
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// === per-user MongoDB collections ===
	metaColl, blobColl := userCollections(req.Username)
	log.Printf("[signup] using collections: %s / %s\n", metaColl, blobColl)

	blobs, err := storage.NewMongoBlobStore(r.Context(), s.mongoURI, s.mongoDB, blobColl)
	if err != nil {
		log.Printf("[signup] mongo blob store error: %v\n", err)
		http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	meta, err := storage.NewMongoMetaStore(r.Context(), s.mongoURI, s.mongoDB, metaColl)
	if err != nil {
		log.Printf("[signup] mongo meta store error: %v\n", err)
		http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// === per-user vault ===
	vpath := userVaultPath(req.Username)
	v := vault.NewWithStores(vpath, blobs, meta)
	log.Printf("[signup] vault path for %q: %s\n", req.Username, vpath)

	// create vault file locally
	if err := v.Create(r.Context(), []byte(req.Password)); err != nil {
		log.Printf("[signup] vault create error for %q: %v\n", req.Username, err)
		http.Error(w, "create vault: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("[signup] vault created successfully for %q\n", req.Username)

	// issue token & store session
	tok, exp, err := s.signer.IssueToken(req.Username, []auth.Role{auth.RoleUser})
	if err != nil {
		http.Error(w, "token issue failed", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.sessions[req.Username] = &userSession{v: v, vpath: vpath, unlocked: true}
	s.mu.Unlock()

	writeJSON(w, signupResp{
		Token:     tok,
		ExpiresAt: exp,
		Vault:     path.Base(vpath),
		Note:      fmt.Sprintf("account created; vault initialized; collections [%s, %s]", metaColl, blobColl),
	})

	log.Printf("[signup] signup completed successfully for %q (meta=%s, blobs=%s)\n", req.Username, metaColl, blobColl)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}
	var u *auth.User
	var err error
	if identifier != "" {
		u, err = s.users.FindByUsername(identifier)
		if err != nil {
			u, err = s.users.FindByEmail(identifier)
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

	if strings.TrimSpace(u.Email) == "" {
		http.Error(w, "user email missing; cannot send 2FA", http.StatusConflict)
		return
	}

	code, err := generateSixDigitCode()
	if err != nil {
		http.Error(w, "2fa code generation failed", http.StatusInternalServerError)
		return
	}
	challengeID, err := generateResetToken()
	if err != nil {
		http.Error(w, "challenge generation failed", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(3 * time.Minute)

	s.mu.Lock()
	for id, ch := range s.pending2FA {
		if ch.Username == u.Username {
			delete(s.pending2FA, id)
		}
	}
	s.pending2FA[challengeID] = &twoFAChallenge{
		Username: u.Username,
		Email:    u.Email,
		Roles:    u.Roles,
		Password: req.Password,
		Code:     code,
		Expires:  expires,
	}
	s.mu.Unlock()

	if err := s.sendTwoFACode(u.Email, code, expires); err != nil {
		s.mu.Lock()
		delete(s.pending2FA, challengeID)
		s.mu.Unlock()
		http.Error(w, "failed to dispatch 2fa code", http.StatusInternalServerError)
		return
	}

	writeJSON(w, twoFAChallengeResp{
		ChallengeID: challengeID,
		ExpiresAt:   expires,
		Note:        "Two-factor code sent to your email address.",
	})
}

func (s *server) handleLoginVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginVerifyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	challengeID := strings.TrimSpace(req.ChallengeID)
	code := strings.TrimSpace(req.Code)
	if challengeID == "" || code == "" {
		http.Error(w, "challenge id and code required", http.StatusBadRequest)
		return
	}

	var challenge *twoFAChallenge
	s.mu.Lock()
	if ch, ok := s.pending2FA[challengeID]; ok {
		if time.Now().After(ch.Expires) {
			delete(s.pending2FA, challengeID)
		} else {
			challenge = ch
		}
	}
	s.mu.Unlock()

	if challenge == nil {
		http.Error(w, "invalid or expired challenge", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(challenge.Code), []byte(code)) != 1 {
		http.Error(w, "invalid code", http.StatusUnauthorized)
		return
	}

	resp, err := s.completeLogin(r.Context(), challenge.Username, challenge.Password, challenge.Roles)
	if err != nil {
		s.mu.Lock()
		delete(s.pending2FA, challengeID)
		s.mu.Unlock()
		log.Printf("[2fa] complete login failed for %s: %v", challenge.Username, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	delete(s.pending2FA, challengeID)
	s.mu.Unlock()

	writeJSON(w, resp)
}

func (s *server) completeLogin(ctx context.Context, username, password string, roles []auth.Role) (loginResp, error) {
	metaColl, blobColl := userCollections(username)
	log.Printf("[login] using collections: %s / %s", metaColl, blobColl)

	blobs, err := storage.NewMongoBlobStore(ctx, s.mongoURI, s.mongoDB, blobColl)
	if err != nil {
		return loginResp{}, fmt.Errorf("mongo blobs: %w", err)
	}
	meta, err := storage.NewMongoMetaStore(ctx, s.mongoURI, s.mongoDB, metaColl)
	if err != nil {
		return loginResp{}, fmt.Errorf("mongo meta: %w", err)
	}

	vpath := userVaultPath(username)
	v := vault.NewWithStores(vpath, blobs, meta)

	if _, statErr := os.Stat(vpath); errors.Is(statErr, os.ErrNotExist) {
		if err := v.Create(ctx, []byte(password)); err != nil {
			return loginResp{}, fmt.Errorf("create vault: %w", err)
		}
	} else {
		if err := v.Unlock(ctx, []byte(password)); err != nil {
			return loginResp{}, fmt.Errorf("unlock: %w", err)
		}
	}

	tok, exp, err := s.signer.IssueToken(username, roles)
	if err != nil {
		return loginResp{}, fmt.Errorf("token issue failed: %w", err)
	}

	s.mu.Lock()
	s.sessions[username] = &userSession{v: v, vpath: vpath, unlocked: true}
	s.mu.Unlock()

	return loginResp{Token: tok, ExpiresAt: exp, Vault: path.Base(vpath)}, nil
}

func (s *server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "no auth context", http.StatusUnauthorized)
		return
	}

	var req changePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	current := strings.TrimSpace(req.Current)
	next := strings.TrimSpace(req.Next)
	if current == "" || next == "" {
		http.Error(w, "current and next passwords required", http.StatusBadRequest)
		return
	}
	if current == next {
		http.Error(w, "new password must differ from current password", http.StatusBadRequest)
		return
	}
	if err := validatePassword(next); err != nil {
		http.Error(w, "weak password: "+err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.users.FindByUsername(claims.Sub)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	okPass, err := auth.VerifyPassword(current, user.PassHash)
	if err != nil || !okPass {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	s.mu.Lock()
	sess := s.sessions[claims.Sub]
	s.mu.Unlock()

	if sess == nil {
		metaColl, blobColl := userCollections(claims.Sub)
		blobs, err := storage.NewMongoBlobStore(ctx, s.mongoURI, s.mongoDB, blobColl)
		if err != nil {
			http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
			return
		}
		meta, err := storage.NewMongoMetaStore(ctx, s.mongoURI, s.mongoDB, metaColl)
		if err != nil {
			http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
			return
		}
		vpath := userVaultPath(claims.Sub)
		v := vault.NewWithStores(vpath, blobs, meta)
		if err := v.Unlock(ctx, []byte(current)); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized)
			return
		}
		sess = &userSession{v: v, vpath: vpath, unlocked: true}
	} else if !sess.unlocked {
		if err := sess.v.Unlock(ctx, []byte(current)); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized)
			return
		}
		sess.unlocked = true
	}

	newHash, err := auth.HashPassword(auth.DefaultArgon, next)
	if err != nil {
		http.Error(w, "hash password failed", http.StatusInternalServerError)
		return
	}

	if err := sess.v.RotateMaster(ctx, []byte(next)); err != nil {
		http.Error(w, "vault rotate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.users.UpdatePassword(claims.Sub, newHash); err != nil {
		// best-effort rollback to previous master
		_ = sess.v.RotateMaster(ctx, []byte(current))
		_ = s.users.UpdatePassword(claims.Sub, user.PassHash)
		http.Error(w, "update password failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sess.v.Lock()
	if err := sess.v.Unlock(ctx, []byte(next)); err != nil {
		http.Error(w, "unlock: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sess.unlocked = true

	s.mu.Lock()
	s.sessions[claims.Sub] = sess
	s.mu.Unlock()

	tok, exp, err := s.signer.IssueToken(user.Username, user.Roles)
	if err != nil {
		http.Error(w, "token issue failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, changePasswordResp{
		Token:     tok,
		ExpiresAt: exp,
		Vault:     path.Base(sess.vpath),
		Note:      "password updated; vault master rotated",
	})
}

func (s *server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req forgotPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	username := strings.TrimSpace(req.Username)
	if email == "" && username == "" {
		http.Error(w, "email or username required", http.StatusBadRequest)
		return
	}

	resp := forgotPasswordResp{
		Note: "If the account exists, you'll receive an email with reset instructions.",
	}

	var user *auth.User
	var err error
	if email != "" {
		user, err = s.users.FindByEmail(email)
	} else {
		user, err = s.users.FindByUsername(username)
	}
	if err != nil || user == nil || user.Email == "" {
		writeJSON(w, resp)
		return
	}

	token, err := generateResetToken()
	if err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}
	exp := time.Now().Add(15 * time.Minute)

	s.mu.Lock()
	for t, existing := range s.resetTokens {
		if existing.Username == user.Username {
			delete(s.resetTokens, t)
		}
	}
	s.resetTokens[token] = resetToken{Username: user.Username, Email: user.Email, Expires: exp}
	s.mu.Unlock()

	s.sendResetEmail(user.Email, token, exp)

	writeJSON(w, resp)
}

func (s *server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req resetPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(req.Token)
	next := strings.TrimSpace(req.Next)
	if token == "" || next == "" {
		http.Error(w, "token and new password required", http.StatusBadRequest)
		return
	}
	if err := validatePassword(next); err != nil {
		http.Error(w, "weak password: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	rt, ok := s.resetTokens[token]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	if time.Now().After(rt.Expires) {
		s.mu.Lock()
		delete(s.resetTokens, token)
		s.mu.Unlock()
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	user, err := s.users.FindByUsername(rt.Username)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	newHash, err := auth.HashPassword(auth.DefaultArgon, next)
	if err != nil {
		http.Error(w, "hash password failed", http.StatusInternalServerError)
		return
	}

	if err := s.users.UpdatePassword(user.Username, newHash); err != nil {
		http.Error(w, "update password failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.nukeUserVault(r.Context(), user.Username); err != nil {
		log.Printf("[reset-password] unable to fully reset vault for %q: %v", user.Username, err)
	}

	s.mu.Lock()
	delete(s.resetTokens, token)
	delete(s.sessions, user.Username)
	s.mu.Unlock()

	resp := resetPasswordResp{
		Note: fmt.Sprintf("Password reset for %s. Vault data was cleared; log in to create a new vault.", user.Username),
	}
	writeJSON(w, resp)
}

func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func generateSixDigitCode() (string, error) {
	max := big.NewInt(1_000_000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *server) sendTwoFACode(email, code string, exp time.Time) error {
	if !s.mailer.enabled {
		return errors.New("mailer disabled")
	}
	body := fmt.Sprintf("Your VaultCraft verification code is %s.\n\nIt expires at %s.\n\nIf you did not request this sign-in, please secure your account immediately.", code, exp.Format(time.RFC1123Z))
	if err := s.sendEmail(email, "Your VaultCraft verification code", body); err != nil {
		return err
	}
	addr := net.JoinHostPort(s.mailer.Host, s.mailer.Port)
	log.Printf("[2fa] sent code to %s via %s (expires %s)", email, addr, exp.Format(time.RFC3339))
	return nil
}

func (s *server) sendResetEmail(email, token string, exp time.Time) {
	base := strings.TrimRight(getenvDefault("APP_BASE_URL", "http://localhost:5173"), "/")
	link := fmt.Sprintf("%s/reset-password?token=%s", base, token)

	if !s.mailer.enabled {
		log.Printf("[reset-email] (mailer disabled) to %s (expires %s): %s", email, exp.Format(time.RFC3339), link)
		return
	}
	body := fmt.Sprintf("Hi,\n\nUse the link below to reset your Project Vault password.\n\n%s\n\nThe link expires at %s. If you didn't request this, you can ignore this message.", link, exp.Format(time.RFC1123Z))
	if err := s.sendEmail(email, "Reset your vault password", body); err != nil {
		addr := net.JoinHostPort(s.mailer.Host, s.mailer.Port)
		log.Printf("[reset-email] send failed via %s: %v (link: %s)", addr, err, link)
		return
	}
	addr := net.JoinHostPort(s.mailer.Host, s.mailer.Port)
	log.Printf("[reset-email] sent to %s via %s (expires %s)", email, addr, exp.Format(time.RFC3339))
}

func (s *server) sendEmail(to, subject, body string) error {
	addr := net.JoinHostPort(s.mailer.Host, s.mailer.Port)
	normalizedBody := strings.ReplaceAll(body, "\n", "\r\n")
	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"From: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=utf-8\r\n"+
			"\r\n"+
			"%s\r\n",
		to, s.mailer.From, subject, normalizedBody,
	))

	var auth smtp.Auth
	if s.mailer.User != "" && s.mailer.Pass != "" {
		auth = smtp.PlainAuth("", s.mailer.User, s.mailer.Pass, s.mailer.Host)
	}

	switch strings.ToLower(s.mailer.security) {
	case "implicit", "ssl":
		tlsCfg := &tls.Config{ServerName: s.mailer.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, s.mailer.Host)
		if err != nil {
			conn.Close()
			return err
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
		if err := client.Mail(s.mailer.From); err != nil {
			return err
		}
		if err := client.Rcpt(to); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg); err != nil {
			w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return client.Quit()
	default:
		return smtp.SendMail(addr, auth, s.mailer.From, []string{to}, msg)
	}
}

func (s *server) nukeUserVault(ctx context.Context, username string) error {
	metaColl, blobColl := userCollections(username)
	var errs []string

	if err := s.dropCollection(ctx, metaColl); err != nil {
		errs = append(errs, fmt.Sprintf("meta: %v", err))
	}
	if err := s.dropCollection(ctx, blobColl); err != nil {
		errs = append(errs, fmt.Sprintf("blobs: %v", err))
	}
	vpath := userVaultPath(username)
	if err := os.Remove(vpath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Sprintf("vault file: %v", err))
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (s *server) dropCollection(ctx context.Context, collName string) error {
	dropCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cli, err := mongo.Connect(dropCtx, options.Client().ApplyURI(s.mongoURI))
	if err != nil {
		return err
	}
	defer cli.Disconnect(dropCtx)

	err = cli.Database(s.mongoDB).Collection(collName).Drop(dropCtx)
	if isNamespaceNotFound(err) {
		return nil
	}
	return err
}

func isNamespaceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
		return true
	}
	return false
}

// ---- session / lock / unlock ----
func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
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

// Optional: allows re-unlock if session cleared (expects {master})
func (s *server) handleUnlock(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(req.Master) == "" {
		http.Error(w, "master required", http.StatusBadRequest)
		return
	}

	// per-user collections (based on token subject)
	metaColl, blobColl := userCollections(claims.Sub)
	log.Printf("[unlock] using collections: %s / %s\n", metaColl, blobColl)

	vpath := userVaultPath(claims.Sub)
	blobs, err := storage.NewMongoBlobStore(r.Context(), s.mongoURI, s.mongoDB, blobColl)
	if err != nil {
		http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	meta, err := storage.NewMongoMetaStore(r.Context(), s.mongoURI, s.mongoDB, metaColl)
	if err != nil {
		http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
		return
	}
	v := vault.NewWithStores(vpath, blobs, meta)

	if _, statErr := os.Stat(vpath); errors.Is(statErr, os.ErrNotExist) {
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
	s.sessions[claims.Sub] = &userSession{v: v, vpath: vpath, unlocked: true}
	s.mu.Unlock()
	writeJSON(w, map[string]any{"ok": true, "vault": path.Base(vpath)})
}

func (s *server) handleLock(w http.ResponseWriter, r *http.Request) {
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

// ---- items (scoped to the caller's vault) ----
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

func (s *server) withSessionVault(r *http.Request) (vault.Vault, error) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return nil, errors.New("no auth context")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[claims.Sub]
	if sess == nil || !sess.unlocked || sess.v == nil {
		return nil, errors.New("vault not unlocked (login again)")
	}
	return sess.v, nil
}

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
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
		// Expected minimal body: { "type": "fb", "fields": {"password":"..."} }
		var it vault.Item
		if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(it.Type) == "" {
			http.Error(w, "type required (e.g., fb, insta, etc.)", http.StatusBadRequest)
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

func (s *server) handleItemByID(w http.ResponseWriter, r *http.Request) {
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
