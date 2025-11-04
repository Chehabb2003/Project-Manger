package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"project-crypto/internal/auth"
	cr "project-crypto/internal/crypto"
	"project-crypto/internal/storage"
	"project-crypto/internal/totp"
	"project-crypto/internal/vault"
)

type loginReq struct {
	Username   string `json:"username"`
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Vault     string    `json:"vault"`
	Note      string    `json:"note,omitempty"`
}

type signupReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type signupResp struct {
	ChallengeID string    `json:"challenge_id,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	Note        string    `json:"note,omitempty"`
	TOTPSecret  string    `json:"totp_secret,omitempty"`
	TOTPUri     string    `json:"totp_uri,omitempty"`
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

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req signupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	if req.Email == "" || !isValidEmail(req.Email) {
		http.Error(w, "valid email required", http.StatusBadRequest)
		return
	}
	if err := validatePassword(req.Password); err != nil {
		http.Error(w, "weak password: "+err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := s.users.FindByUsername(req.Username); err == nil {
		http.Error(w, "username already exists", http.StatusConflict)
		return
	}
	if _, err := s.users.FindByEmail(req.Email); err == nil {
		http.Error(w, "email already exists", http.StatusConflict)
		return
	}

	hash, err := auth.HashPassword(auth.DefaultArgon, req.Password)
	if err != nil {
		http.Error(w, "hash password failed", http.StatusInternalServerError)
		return
	}

	secret, err := totp.GenerateSecret()
	if err != nil {
		http.Error(w, "totp generation failed", http.StatusInternalServerError)
		return
	}

	user := &auth.User{
		Username:   req.Username,
		Email:      req.Email,
		PassHash:   hash,
		Roles:      []auth.Role{auth.RoleUser},
		TOTPSecret: secret,
	}
	if err := s.users.Add(user); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	master := []byte(req.Password)
	req.Password = ""

	challengeID, err := randomToken(16)
	if err != nil {
		cr.Zero(master)
		http.Error(w, "challenge generation failed", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(10 * time.Minute)

	s.mu.Lock()
	for id, ch := range s.challs {
		if ch.Username == user.Username {
			cr.Zero(ch.Master)
			delete(s.challs, id)
		}
	}
	s.challs[challengeID] = &twoFAChallenge{
		Username: user.Username,
		Roles:    user.Roles,
		Master:   master,
		Expires:  expires,
	}
	s.mu.Unlock()

	provisionURI := totp.ProvisionURI(req.Username, s.cfg.TOTPIssuer, secret)
	writeJSON(w, signupResp{
		ChallengeID: challengeID,
		ExpiresAt:   expires,
		Note:        "Scan the QR code and confirm with a 6-digit authenticator code to finish.",
		TOTPUri:     provisionURI,
		TOTPSecret:  secret,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
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

	var user *auth.User
	var err error
	if identifier != "" {
		user, err = s.users.FindByUsername(identifier)
		if err != nil {
			user, err = s.users.FindByEmail(identifier)
		}
	} else {
		err = errors.New("identifier missing")
	}
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	ok, err := auth.VerifyPassword(req.Password, user.PassHash)
	if err != nil || !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	master := []byte(req.Password)
	req.Password = ""

	if strings.TrimSpace(user.TOTPSecret) == "" {
		cr.Zero(master)
		http.Error(w, "user is not enrolled for TOTP; contact support", http.StatusConflict)
		return
	}

	challengeID, err := randomToken(16)
	if err != nil {
		cr.Zero(master)
		http.Error(w, "challenge generation failed", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(3 * time.Minute)

	s.mu.Lock()
	for id, ch := range s.challs {
		if ch.Username == user.Username {
			cr.Zero(ch.Master)
			delete(s.challs, id)
		}
	}
	s.challs[challengeID] = &twoFAChallenge{
		Username: user.Username,
		Roles:    user.Roles,
		Master:   master,
		Expires:  expires,
	}
	s.mu.Unlock()

	writeJSON(w, twoFAChallengeResp{
		ChallengeID: challengeID,
		ExpiresAt:   expires,
		Note:        "Submit the 6-digit code from your authenticator app.",
	})
}

func (s *Server) handleLoginVerify(w http.ResponseWriter, r *http.Request) {
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
	if ch, ok := s.challs[challengeID]; ok {
		if time.Now().After(ch.Expires) {
			cr.Zero(ch.Master)
			delete(s.challs, challengeID)
		} else {
			challenge = ch
		}
	}
	s.mu.Unlock()

	if challenge == nil {
		http.Error(w, "invalid or expired challenge", http.StatusUnauthorized)
		return
	}

	user, err := s.users.FindByUsername(challenge.Username)
	if err != nil {
		s.clearChallenge(challengeID)
		http.Error(w, "invalid challenge", http.StatusUnauthorized)
		return
	}

    if !totp.Verify(code, user.TOTPSecret, time.Now().UTC()) {
        http.Error(w, "invalid code", http.StatusUnauthorized)
        return
    }

	resp, err := s.completeLogin(r.Context(), challenge.Username, challenge.Master, challenge.Roles)
	if err != nil {
		s.clearChallenge(challengeID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.clearChallenge(challengeID)
	writeJSON(w, resp)
}

func (s *Server) clearChallenge(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.challs[id]; ok {
		cr.Zero(ch.Master)
		delete(s.challs, id)
	}
}

func shouldResetVault(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, cr.ErrCiphertextTooShort) {
		return true
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}
	return false
}

func (s *Server) completeLogin(ctx context.Context, username string, master []byte, roles []auth.Role) (loginResp, error) {
	metaColl, blobColl := collectionNames(username)

    var blobs storage.BlobStore
    var err error
    if s.storageClient != nil {
        blobs, err = storage.NewMongoBlobStoreWithClient(s.storageClient, s.cfg.MongoDB, blobColl)
    } else {
        blobs, err = storage.NewMongoBlobStore(ctx, s.cfg.MongoURI, s.cfg.MongoDB, blobColl)
    }
    if err != nil {
        return loginResp{}, fmt.Errorf("mongo blobs: %w", err)
    }
    var meta *storage.MongoMetaStore
    if s.storageClient != nil {
        meta, err = storage.NewMongoMetaStoreWithClient(s.storageClient, s.cfg.MongoDB, metaColl)
    } else {
        meta, err = storage.NewMongoMetaStore(ctx, s.cfg.MongoURI, s.cfg.MongoDB, metaColl)
    }
    if err != nil {
        return loginResp{}, fmt.Errorf("mongo meta: %w", err)
    }

	vpath := s.vaultPath(username)
	v := vault.NewWithStores(vpath, blobs, meta)

	masterCopy := append([]byte(nil), master...)
	defer cr.Zero(masterCopy)

	if _, statErr := os.Stat(vpath); errors.Is(statErr, os.ErrNotExist) {
		if err := v.Create(ctx, masterCopy); err != nil {
			return loginResp{}, fmt.Errorf("create vault: %w", err)
		}
	} else {
		if err := v.Unlock(ctx, masterCopy); err != nil {
			if shouldResetVault(err) {
				s.logger.Printf("[vault] %s vault corrupted (%v); recreating", username, err)
				if nukeErr := s.nukeUserVault(ctx, username); nukeErr != nil {
					return loginResp{}, fmt.Errorf("unlock: %w", err)
				}
				v = vault.NewWithStores(vpath, blobs, meta)
				if err := v.Create(ctx, masterCopy); err != nil {
					return loginResp{}, fmt.Errorf("recreate vault: %w", err)
				}
			} else {
				return loginResp{}, fmt.Errorf("unlock: %w", err)
			}
		}
	}

	tok, exp, err := s.signer.IssueToken(username, roles)
	if err != nil {
		return loginResp{}, fmt.Errorf("token issue failed: %w", err)
	}

	s.mu.Lock()
	s.sessions[username] = &userSession{v: v, vpath: vpath, unlocked: true}
	s.mu.Unlock()

	return loginResp{Token: tok, ExpiresAt: exp, Vault: filepath.Base(vpath)}, nil
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
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
		Note: "If the account exists, you'll receive a reset link shortly.",
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

	token, err := randomToken(24)
	if err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}
	exp := time.Now().Add(15 * time.Minute)

	s.mu.Lock()
	for t, existing := range s.resets {
		if existing.Username == user.Username {
			delete(s.resets, t)
		}
	}
	s.resets[token] = resetToken{Username: user.Username, Email: user.Email, Expires: exp}
	s.mu.Unlock()

	if s.mail.Enabled() {
		if err := s.mail.SendResetPassword(user.Email, token, exp); err != nil {
			s.logger.Printf("reset email error: %v", err)
		}
	} else {
		s.logger.Printf("password reset link for %s -> token=%s", user.Email, token)
	}

	writeJSON(w, resp)
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "token and next password required", http.StatusBadRequest)
		return
	}
	if err := validatePassword(next); err != nil {
		http.Error(w, "weak password: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	info, ok := s.resets[token]
	if ok && time.Now().After(info.Expires) {
		delete(s.resets, token)
		ok = false
	}
	s.mu.Unlock()

	if !ok {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	user, err := s.users.FindByUsername(info.Username)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	hash, err := auth.HashPassword(auth.DefaultArgon, next)
	if err != nil {
		http.Error(w, "hash password failed", http.StatusInternalServerError)
		return
	}

	if err := s.users.UpdatePassword(user.Username, hash); err != nil {
		http.Error(w, "update password failed", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	delete(s.resets, token)
	s.mu.Unlock()

	writeJSON(w, resetPasswordResp{
		Note: "Password updated. Sign in with your new password and authenticator code.",
	})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
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

	passOK, err := auth.VerifyPassword(current, user.PassHash)
	if err != nil || !passOK {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	masterCurrent := []byte(current)
	masterNext := []byte(next)
	defer func() {
		cr.Zero(masterCurrent)
		cr.Zero(masterNext)
	}()

	s.mu.Lock()
	sess := s.sessions[claims.Sub]
	s.mu.Unlock()

	if sess == nil {
		metaColl, blobColl := collectionNames(claims.Sub)
    var blobs storage.BlobStore
    var err error
    if s.storageClient != nil {
        blobs, err = storage.NewMongoBlobStoreWithClient(s.storageClient, s.cfg.MongoDB, blobColl)
    } else {
        blobs, err = storage.NewMongoBlobStore(ctx, s.cfg.MongoURI, s.cfg.MongoDB, blobColl)
    }
    if err != nil {
        http.Error(w, "mongo blobs: "+err.Error(), http.StatusInternalServerError)
        return
    }
    var meta *storage.MongoMetaStore
    if s.storageClient != nil {
        meta, err = storage.NewMongoMetaStoreWithClient(s.storageClient, s.cfg.MongoDB, metaColl)
    } else {
        meta, err = storage.NewMongoMetaStore(ctx, s.cfg.MongoURI, s.cfg.MongoDB, metaColl)
    }
    if err != nil {
        http.Error(w, "mongo meta: "+err.Error(), http.StatusInternalServerError)
        return
    }
		vpath := s.vaultPath(claims.Sub)
		v := vault.NewWithStores(vpath, blobs, meta)
		if err := v.Unlock(ctx, masterCurrent); err != nil {
			http.Error(w, "unlock: "+err.Error(), http.StatusUnauthorized)
			return
		}
		sess = &userSession{v: v, vpath: vpath, unlocked: true}
	} else if !sess.unlocked {
		if err := sess.v.Unlock(ctx, masterCurrent); err != nil {
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

	if err := sess.v.RotateMaster(ctx, masterNext); err != nil {
		http.Error(w, "vault rotate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.users.UpdatePassword(claims.Sub, newHash); err != nil {
		_ = sess.v.RotateMaster(ctx, masterCurrent)
		_ = s.users.UpdatePassword(claims.Sub, user.PassHash)
		http.Error(w, "update password failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sess.v.Lock()
	if err := sess.v.Unlock(ctx, masterNext); err != nil {
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
		Vault:     filepath.Base(sess.vpath),
		Note:      "Password updated; vault master rotated.",
	})
}
