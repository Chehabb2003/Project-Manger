package crypto

import (
    "errors"

    xchacha "golang.org/x/crypto/chacha20poly1305"
)

// OpenAny first tries the current envelope Open(); if that fails due to MAC
// mismatch, it falls back to legacy XChaCha20-Poly1305 format used by older
// vault files/items. This provides backward compatibility for data created
// before the encryption refactor.
func OpenAny(key, ciphertext, aad []byte) ([]byte, error) {
    if pt, err := Open(key, ciphertext, aad); err == nil {
        return pt, nil
    }
    // Fallback attempt using XChaCha20-Poly1305
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

