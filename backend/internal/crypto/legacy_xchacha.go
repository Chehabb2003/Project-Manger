package crypto

import (
	"errors"

	xchacha "golang.org/x/crypto/chacha20poly1305"
)

func OpenAny(key, ciphertext, aad []byte) ([]byte, error) {
	if pt, err := Open(key, ciphertext, aad); err == nil {
		return pt, nil
	}
	if pt, err := openLegacyCTR(key, ciphertext, aad); err == nil {
		return pt, nil
	}

	a, err := xchacha.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < xchacha.NonceSizeX {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:xchacha.NonceSizeX]
	ct := ciphertext[xchacha.NonceSizeX:]
	return a.Open(nil, nonce, ct, aad)
}
