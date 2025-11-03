package crypto

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

type KDFParams struct {
	M    uint32 // KiB
	T    uint32 // iterations
	P    uint8  // parallelism
	Salt []byte
}

func DefaultDesktopKDF() KDFParams {
	salt := make([]byte, 32)
	_, _ = rand.Read(salt)
	return KDFParams{M: 1024 * 1024, T: 3, P: 4, Salt: salt} // 1 GiB
}

func DefaultMobileKDF() KDFParams {
	salt := make([]byte, 32)
	_, _ = rand.Read(salt)
	return KDFParams{M: 128 * 1024, T: 3, P: 4, Salt: salt} // 128 MiB
}

func DeriveKEK(master []byte, p KDFParams) (kek [32]byte) {
	key := argon2.IDKey(master, p.Salt, p.T, p.M, p.P, 32)
	copy(kek[:], key)
	for i := range key { key[i] = 0 }
	return
}

func EncodeSalt(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
