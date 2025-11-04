package crypto

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

type KDFParams struct {
	M    uint32
	T    uint32
	P    uint8
	Salt []byte
}

func DefaultDesktopKDF() KDFParams {
	salt := make([]byte, 32)
	_, _ = rand.Read(salt)
	return KDFParams{M: 1024 * 1024, T: 3, P: 4, Salt: salt}
}

func DefaultMobileKDF() KDFParams {
	salt := make([]byte, 32)
	_, _ = rand.Read(salt)
	return KDFParams{M: 128 * 1024, T: 3, P: 4, Salt: salt}
}

func DeriveKEK(master []byte, p KDFParams) (kek [32]byte) {
	key := argon2.IDKey(master, p.Salt, p.T, p.M, p.P, 32)
	copy(kek[:], key)
	for i := range key {
		key[i] = 0
	}
	return
}

func EncodeSalt(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
