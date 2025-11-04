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
	envelopeSaltSize = 32
	envelopeIVSize   = aes.BlockSize // 16 bytes
	envelopeMacSize  = sha256.Size   // 32 bytes
	envelopeMinSize  = envelopeSaltSize + envelopeIVSize + envelopeMacSize
)

var (
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
	ErrInvalidMAC         = errors.New("crypto: message authentication failed")
)

// Seal applies encrypt-then-MAC using AES-CTR for confidentiality and HMAC-SHA256
// for integrity. Keys are derived from the provided master key with HKDF-SHA256,
// using a per-message random salt baked into the ciphertext. Returned layout:
// [salt||iv||ciphertext||mac].
func Seal(masterKey, plaintext, aad []byte) ([]byte, error) {
	if len(masterKey) == 0 {
		return nil, errors.New("crypto: empty master key")
	}

	salt := make([]byte, envelopeSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	encKey, macKey, err := deriveEnvelopeKeys(masterKey, salt)
	if err != nil {
		return nil, err
	}
	defer Zero(encKey)
	defer Zero(macKey)

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, envelopeIVSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	ct := make([]byte, len(plaintext))
	cipher.NewCTR(block, iv).XORKeyStream(ct, plaintext)

	macTag := computeMAC(macKey, aad, iv, ct)

	out := make([]byte, 0, envelopeSaltSize+envelopeIVSize+len(ct)+envelopeMacSize)
	out = append(out, salt...)
	out = append(out, iv...)
	out = append(out, ct...)
	out = append(out, macTag...)
	return out, nil
}

// Open decrypts and authenticates data previously sealed with Seal.
func Open(masterKey, ciphertext, aad []byte) ([]byte, error) {
	if len(ciphertext) < envelopeMinSize {
		return nil, ErrCiphertextTooShort
	}
	if len(masterKey) == 0 {
		return nil, errors.New("crypto: empty master key")
	}

	salt := ciphertext[:envelopeSaltSize]
	iv := ciphertext[envelopeSaltSize : envelopeSaltSize+envelopeIVSize]
	macStart := len(ciphertext) - envelopeMacSize
	body := ciphertext[envelopeSaltSize+envelopeIVSize : macStart]
	macTag := ciphertext[macStart:]

	encKey, macKey, err := deriveEnvelopeKeys(masterKey, salt)
	if err != nil {
		return nil, err
	}
	defer Zero(encKey)
	defer Zero(macKey)

	expected := computeMAC(macKey, aad, iv, body)
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

func deriveEnvelopeKeys(masterKey, salt []byte) (encKey, macKey []byte, err error) {
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

func computeMAC(macKey, aad, iv, ciphertext []byte) []byte {
	mac := hmac.New(sha256.New, macKey)
	if len(aad) > 0 {
		mac.Write(aad)
	}
	mac.Write(iv)
	mac.Write(ciphertext)
	return mac.Sum(nil)
}
