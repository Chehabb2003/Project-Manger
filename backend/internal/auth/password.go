package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type ArgonParams struct {
	Memory      uint32 // in KiB (e.g., 64*1024)
	Time        uint32 // iterations
	Parallelism uint8
	SaltLen     int
	KeyLen      uint32
}

var DefaultArgon = ArgonParams{
	Memory:      64 * 1024,
	Time:        3,
	Parallelism: 1,
	SaltLen:     16,
	KeyLen:      32,
}

func HashPassword(p ArgonParams, password string) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, p.KeyLen)
	// encoded format: argon2id$m=<M>,t=<T>,p=<P>$<b64(salt)>$<b64(key)>
	return fmt.Sprintf("argon2id$m=%d,t=%d,p=%d$%s$%s",
		p.Memory, p.Time, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

var ErrInvalidHash = errors.New("invalid password hash")

func VerifyPassword(password, encoded string) (bool, error) {
	const prefix = "argon2id$"
	if !strings.HasPrefix(encoded, prefix) {
		return false, ErrInvalidHash
	}
	parts := strings.Split(encoded[len(prefix):], "$")
	if len(parts) != 3 {
		return false, ErrInvalidHash
	}

	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[0], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, ErrInvalidHash
	}
	keyRef, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, ErrInvalidHash
	}

	key := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(keyRef)))
	if subtle.ConstantTimeCompare(key, keyRef) == 1 {
		return true, nil
	}
	return false, nil
}
