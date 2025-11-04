package server

import (
	"time"

	"project-crypto/internal/auth"
	"project-crypto/internal/vault"
)

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
	Roles    []auth.Role
	Master   []byte
	Expires  time.Time
}

type mailer interface {
	SendResetPassword(to, token string, expires time.Time) error
	Enabled() bool
}
