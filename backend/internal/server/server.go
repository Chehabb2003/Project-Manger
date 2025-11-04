package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"project-crypto/internal/auth"
	"project-crypto/internal/totp"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/time/rate"
)

type Server struct {
	cfg Config

	mux      *http.ServeMux
	signer   *auth.JWTSigner
	users    auth.UserStore
	mail     mailer
	logger   *log.Logger
	mu       sync.Mutex
	sessions map[string]*userSession
	resets   map[string]resetToken
	challs   map[string]*twoFAChallenge

	storageClient *mongo.Client

	rlLoginIP       *multiLimiter
	rlLoginID       *multiLimiter
	rlTotpIP        *multiLimiter
	rlTotpChallenge *multiLimiter
	rlTotpUser      *multiLimiter
	rlForgotIP      *multiLimiter
	rlForgotID      *multiLimiter
	rlResetIP       *multiLimiter
	rlResetToken    *multiLimiter
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	cfg.setDefaults()
	if cfg.MongoURI == "" {
		return nil, errors.New("server: MongoURI required")
	}
	if cfg.MongoDB == "" {
		return nil, errors.New("server: MongoDB required")
	}

	if err := os.MkdirAll(cfg.VaultDir, 0o700); err != nil {
		return nil, err
	}

	priv, _, err := auth.GenerateEd25519()
	if err != nil {
		return nil, err
	}

	users, err := auth.NewMongoUserStore(ctx, cfg.MongoURI, cfg.MongoDB, cfg.UsersCollection)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:      cfg,
		mux:      http.NewServeMux(),
		signer:   auth.NewJWTSigner(priv, cfg.JWTIssuer, cfg.TokenTTL),
		users:    users,
		logger:   log.New(os.Stdout, "[server] ", log.LstdFlags|log.Lshortfile),
		sessions: map[string]*userSession{},
		resets:   map[string]resetToken{},
		challs:   map[string]*twoFAChallenge{},
	}
	s.mail = newSMTPMailer(cfg.SMTP, s.logger)

	sc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sc.Ping(pingCtx, nil); err != nil {
		_ = sc.Disconnect(context.Background())
		return nil, err
	}
	s.storageClient = sc

	perWindow := func(n int, window time.Duration) float64 { return float64(n) / window.Seconds() }

	s.rlLoginIP = newMultiLimiter(rate.Limit(perWindow(10, time.Minute)), 10, 1*time.Hour)
	s.rlLoginID = newMultiLimiter(rate.Limit(perWindow(5, time.Minute)), 5, 1*time.Hour)

	s.rlTotpIP = newMultiLimiter(rate.Limit(perWindow(10, time.Minute)), 10, 10*time.Minute)
	s.rlTotpChallenge = newMultiLimiter(rate.Limit(perWindow(3, time.Minute)), 3, 10*time.Minute)
	s.rlTotpUser = newMultiLimiter(rate.Limit(perWindow(5, time.Minute)), 5, 10*time.Minute)

	s.rlForgotIP = newMultiLimiter(rate.Limit(perWindow(5, 15*time.Minute)), 5, 30*time.Minute)
	s.rlForgotID = newMultiLimiter(rate.Limit(perWindow(3, 15*time.Minute)), 3, 30*time.Minute)

	s.rlResetIP = newMultiLimiter(rate.Limit(perWindow(10, 15*time.Minute)), 10, 30*time.Minute)
	s.rlResetToken = newMultiLimiter(rate.Limit(perWindow(5, 15*time.Minute)), 5, 30*time.Minute)

	if err := s.ensureSeedUsers(ctx); err != nil {
		return nil, err
	}

	s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Printf("panic: %v", rec)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}()

	s.addDefaultHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := r.URL.Path
	if strings.HasPrefix(path, "/api/") {
		if s.isPublic(path) {
			s.mux.ServeHTTP(w, r)
			return
		}
		handler := auth.AuthRequired(s.signer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.mux.ServeHTTP(w, r)
		}))
		handler.ServeHTTP(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Handler() http.Handler {
	return s
}

func (s *Server) isPublic(path string) bool {
	switch path {
	case "/health", "/api/health", "/api/login", "/api/signup", "/api/password/forgot", "/api/password/reset", "/api/login/verify":
		return true
	default:
		return false
	}
}

func (s *Server) addDefaultHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
}

func (s *Server) ensureSeedUsers(ctx context.Context) error {
	for _, seed := range s.cfg.SeedUsers {
		if strings.TrimSpace(seed.Username) == "" || strings.TrimSpace(seed.Password) == "" {
			continue
		}
		if _, err := s.users.FindByUsername(seed.Username); err == nil {
			continue
		}
		hash, err := auth.HashPassword(auth.DefaultArgon, seed.Password)
		if err != nil {
			return err
		}
		secret, err := totp.GenerateSecret()
		if err != nil {
			return err
		}
		user := &auth.User{
			Username:   seed.Username,
			Email:      strings.TrimSpace(strings.ToLower(seed.Email)),
			PassHash:   hash,
			Roles:      seed.Roles,
			TOTPSecret: secret,
		}
		if err := s.users.Add(user); err != nil {
			return err
		}
		s.logger.Printf("seeded user %s (%s) totp_secret=%s", seed.Username, strings.Join(roleNames(seed.Roles), ","), secret)
	}
	return nil
}

func (s *Server) vaultPath(username string) string {
	name := sha256Hex(username) + ".vlt"
	return filepath.Join(s.cfg.VaultDir, name)
}

func (s *Server) nukeUserVault(ctx context.Context, username string) error {
	metaColl, blobColl := collectionNames(username)
	var errs []string

	if err := s.dropCollection(ctx, metaColl); err != nil {
		errs = append(errs, fmt.Sprintf("meta: %v", err))
	}
	if err := s.dropCollection(ctx, blobColl); err != nil {
		errs = append(errs, fmt.Sprintf("blobs: %v", err))
	}
	vpath := s.vaultPath(username)
	if err := os.Remove(vpath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Sprintf("vault file: %v", err))
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (s *Server) dropCollection(ctx context.Context, collName string) error {
	dropCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cli, err := mongo.Connect(dropCtx, options.Client().ApplyURI(s.cfg.MongoURI))
	if err != nil {
		return err
	}
	defer cli.Disconnect(dropCtx)

	err = cli.Database(s.cfg.MongoDB).Collection(collName).Drop(dropCtx)
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
	return errors.As(err, &cmdErr) && cmdErr.Code == 26
}
