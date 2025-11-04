package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	envelopeSaltSize  = 32
	envelopeNonceSize = 12
	envelopeTagSize   = 16
	envelopeMinSize   = envelopeSaltSize + envelopeNonceSize + envelopeTagSize

	legacyIVSize  = aes.BlockSize
	legacyMacSize = sha256.Size
	legacyMinSize = envelopeSaltSize + legacyIVSize + legacyMacSize
)

var (
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
	ErrInvalidMAC         = errors.New("crypto: message authentication failed")
)

func Seal(masterKey, plaintext, aad []byte) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, errors.New("crypto: empty master key")
	}

	salt := make([]byte, envelopeSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	encKey, err := deriveGCMKey(masterKey, salt)
	if err != nil {
		return nil, err
	}
	defer Zero(encKey)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if gcm.NonceSize() != envelopeNonceSize {
		return nil, errors.New("crypto: unexpected gcm nonce size")
	}
	if gcm.Overhead() != envelopeTagSize {
		return nil, errors.New("crypto: unexpected gcm tag size")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ct := gcm.Seal(nil, nonce, plaintext, aad)
	out := make([]byte, 0, envelopeSaltSize+len(nonce)+len(ct))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func Open(masterKey, ciphertext, aad []byte) ([]byte, error) {
	if len(ciphertext) < envelopeMinSize {
		return nil, ErrCiphertextTooShort
	}
	if len(masterKey) == 0 {
		return nil, errors.New("crypto: empty master key")
	}

	salt := ciphertext[:envelopeSaltSize]
	nonce := ciphertext[envelopeSaltSize : envelopeSaltSize+envelopeNonceSize]
	body := ciphertext[envelopeSaltSize+envelopeNonceSize:]

	encKey, err := deriveGCMKey(masterKey, salt)
	if err != nil {
		return nil, err
	}
	defer Zero(encKey)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ErrCiphertextTooShort
	}

	pt, err := gcm.Open(nil, nonce, body, aad)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func deriveGCMKey(masterKey, salt []byte) ([]byte, error) {
	stream := hkdf.New(sha256.New, masterKey, salt, []byte("vault/envelope/v2"))
	encKey := make([]byte, 32)
	if _, err := io.ReadFull(stream, encKey); err != nil {
		return nil, err
	}
	return encKey, nil
}

func openLegacyCTR(masterKey, ciphertext, aad []byte) ([]byte, error) {
	if len(ciphertext) < legacyMinSize {
		return nil, ErrCiphertextTooShort
	}
	if len(masterKey) == 0 {
		return nil, errors.New("crypto: empty master key")
	}

	salt := ciphertext[:envelopeSaltSize]
	iv := ciphertext[envelopeSaltSize : envelopeSaltSize+legacyIVSize]
	macStart := len(ciphertext) - legacyMacSize
	body := ciphertext[envelopeSaltSize+legacyIVSize : macStart]
	macTag := ciphertext[macStart:]

	encKey, macKey, err := deriveLegacyEnvelopeKeys(masterKey, salt)
	if err != nil {
		return nil, err
	}
	defer Zero(encKey)
	defer Zero(macKey)

	expected := computeLegacyMAC(macKey, aad, iv, body)
	if subtle.ConstantTimeCompare(expected, macTag) != 1 {
		return nil, ErrInvalidMAC
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	pt := make([]byte, len(body))
	cipher.NewCTR(block, iv).XORKeyStream(pt, body)
	return pt, nil
}

func deriveLegacyEnvelopeKeys(masterKey, salt []byte) (encKey, macKey []byte, err error) {
	stream := hkdf.New(sha256.New, masterKey, salt, []byte("vault/envelope/v1"))
	encKey = make([]byte, 32)
	macKey = make([]byte, 32)
	if _, err = io.ReadFull(stream, encKey); err != nil {
		return nil, nil, err
	}
	if _, err = io.ReadFull(stream, macKey); err != nil {
		return nil, nil, err
	}
	return encKey, macKey, nil
}

func computeLegacyMAC(macKey, aad, iv, ciphertext []byte) []byte {
	mac := hmac.New(sha256.New, macKey)
	if len(aad) > 0 {
		mac.Write(aad)
	}
	mac.Write(iv)
	mac.Write(ciphertext)
	return mac.Sum(nil)
}
