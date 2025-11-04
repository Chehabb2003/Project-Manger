package server

import (
	"time"

	"project-crypto/internal/auth"
)

type SeedUser struct {
	Username string
	Email    string
	Password string
	Roles    []auth.Role
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Pass     string
	From     string
	Security string
}

type Config struct {
	MongoURI        string
	MongoDB         string
	UsersCollection string
	VaultDir        string
	JWTIssuer       string
	TokenTTL        time.Duration
	TOTPIssuer      string
	SMTP            SMTPConfig
	SeedUsers       []SeedUser
}

func (c *Config) setDefaults() {
	if c.UsersCollection == "" {
		c.UsersCollection = "users"
	}
	if c.VaultDir == "" {
		c.VaultDir = "./vaults"
	}
	if c.JWTIssuer == "" {
		c.JWTIssuer = "vaultcraft-backend"
	}
	if c.TokenTTL <= 0 {
		c.TokenTTL = 15 * time.Minute
	}
	if c.TOTPIssuer == "" {
		c.TOTPIssuer = "VaultCraft"
	}
	if c.SMTP.Security == "" {
		c.SMTP.Security = "starttls"
	}
}
