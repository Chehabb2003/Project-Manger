package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"

	xchacha "golang.org/x/crypto/chacha20poly1305"
)

func NewXChaCha(key []byte) (cipher.AEAD, error) {
	return xchacha.NewX(key)
}

func SealX(key, plaintext, aad []byte) ([]byte, error) {
	aead, err := xchacha.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, xchacha.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(nonce)+len(plaintext)+aead.Overhead())
	out = append(out, nonce...)
	out = aead.Seal(out[:len(nonce)], nonce, plaintext, aad)
	return out, nil
}

func OpenX(key, ciphertext, aad []byte) ([]byte, error) {
	aead, err := xchacha.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < xchacha.NonceSizeX {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:xchacha.NonceSizeX]
	ct := ciphertext[xchacha.NonceSizeX:]
	return aead.Open(nil, nonce, ct, aad)
}
