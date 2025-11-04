package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"project-crypto/internal/auth"
	srv "project-crypto/internal/server"

	"github.com/joho/godotenv"
)

const defaultMongoURI = "mongodb+srv://mmh164:Aub%40ohf02@crypto.5soje3e.mongodb.net/?retryWrites=true&w=majority&ssl=true"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_ = godotenv.Load(".env", "backend/.env")

	port := flag.String("port", "8080", "HTTP port to listen on")
	mongoURI := flag.String("mongo", "", "MongoDB URI (or use MONGODB_URI env, or default)")
	mongoDB := flag.String("db", "vaultdb", "MongoDB database name")
	usersColl := flag.String("users", "users", "MongoDB users collection")
	vaultDir := flag.String("vaultdir", "./vaults", "Directory for user vault files")
	jwtIssuer := flag.String("issuer", getenvDefault("JWT_ISSUER", "vaultcraft-backend"), "JWT issuer")
	totpIssuer := flag.String("totp-issuer", getenvDefault("TOTP_ISSUER", "VaultCraft"), "TOTP issuer (for authenticator apps)")
	flag.Parse()

	if *mongoURI == "" {
		if env := os.Getenv("MONGODB_URI"); env != "" {
			*mongoURI = env
		} else {
			*mongoURI = defaultMongoURI
		}
	}

	cfg := srv.Config{
		MongoURI:        *mongoURI,
		MongoDB:         *mongoDB,
		UsersCollection: *usersColl,
		VaultDir:        *vaultDir,
		JWTIssuer:       *jwtIssuer,
		TokenTTL:        15 * time.Minute,
		TOTPIssuer:      *totpIssuer,
		SMTP: srv.SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     firstNonEmpty(os.Getenv("SMTP_PORT"), "587"),
			User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
			Pass:     os.Getenv("SMTP_PASS"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
			Security: firstNonEmpty(strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_SECURITY"))), "starttls"),
		},
		SeedUsers: []srv.SeedUser{
			{
				Username: getenvDefault("ADMIN_USER", "admin"),
				Email:    strings.TrimSpace(strings.ToLower(getenvDefault("ADMIN_EMAIL", "admin@example.com"))),
				Password: getenvDefault("ADMIN_PASS", "admin123!"),
				Roles:    []auth.Role{auth.RoleAdmin, auth.RoleUser},
			},
			{
				Username: getenvDefault("APP_USER", "user"),
				Email:    strings.TrimSpace(strings.ToLower(getenvDefault("APP_EMAIL", "user@example.com"))),
				Password: getenvDefault("APP_PASS", "user123!"),
				Roles:    []auth.Role{auth.RoleUser},
			},
		},
	}

	ctx := context.Background()
	s, err := srv.New(ctx, cfg)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	if err := os.MkdirAll(cfg.VaultDir, 0o700); err != nil {
		log.Fatalf("mkdir vaultdir: %v", err)
	}

	log.Printf("HTTP on :%s (mongo db=%s, users=%s, totp issuer=%s)", *port, cfg.MongoDB, cfg.UsersCollection, cfg.TOTPIssuer)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", *port), s.Handler()))
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func firstNonEmpty(v string, def string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return def
}
