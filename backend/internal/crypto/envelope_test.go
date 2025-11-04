package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"testing"
)

func randBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}

func TestSealOpenRoundTrip(t *testing.T) {
	master := randBytes(t, 32)
	pt := randBytes(t, 4096)
	aad := []byte("context")
	ct, err := Seal(master, pt, aad)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	out, err := Open(master, ct, aad)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(pt, out) {
		t.Fatal("plaintext mismatch")
	}
}

func TestSealOpenAADMismatch(t *testing.T) {
	master := randBytes(t, 32)
	pt := []byte("secret-data")
	ct, err := Seal(master, pt, []byte("aad-1"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := Open(master, ct, []byte("aad-2")); err == nil {
		t.Fatal("expected auth failure with mismatched AAD")
	}
}

func TestSealOpenTagTamper(t *testing.T) {
	master := randBytes(t, 32)
	pt := []byte("hello")
	ct, err := Seal(master, pt, nil)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	mut := append([]byte(nil), ct...)
	mut[len(mut)-1] ^= 0xFF
	if _, err := Open(master, mut, nil); err == nil {
		t.Fatal("expected failure after tag tamper")
	}
}

func TestSealOpenTruncation(t *testing.T) {
	master := randBytes(t, 32)
	pt := []byte("hello")
	ct, err := Seal(master, pt, nil)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	mut := append([]byte(nil), ct...)
	mut = mut[:len(mut)-1]
	if _, err := Open(master, mut, nil); err == nil {
		t.Fatal("expected failure on truncated ciphertext")
	}
}

func TestSealOpenLegacyFallback(t *testing.T) {
	master := randBytes(t, 32)
	pt := []byte("legacy-support")
	salt := randBytes(t, envelopeSaltSize)
	encKey, macKey, err := deriveLegacyEnvelopeKeys(master, salt)
	if err != nil {
		t.Fatalf("derive legacy: %v", err)
	}
	defer Zero(encKey)
	defer Zero(macKey)
	block, err := aes.NewCipher(encKey)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	iv := randBytes(t, legacyIVSize)
	ctBody := make([]byte, len(pt))
	cipher.NewCTR(block, iv).XORKeyStream(ctBody, pt)
	mac := computeLegacyMAC(macKey, nil, iv, ctBody)
	legacy := make([]byte, 0, envelopeSaltSize+legacyIVSize+len(ctBody)+legacyMacSize)
	legacy = append(legacy, salt...)
	legacy = append(legacy, iv...)
	legacy = append(legacy, ctBody...)
	legacy = append(legacy, mac...)

	if _, err := Open(master, legacy, nil); err == nil {
		t.Fatal("expected new Open to reject legacy ciphertext")
	}
	got, err := OpenAny(master, legacy, nil)
	if err != nil {
		t.Fatalf("OpenAny failed: %v", err)
	}
	if !bytes.Equal(pt, got) {
		t.Fatal("legacy plaintext mismatch")
	}
}

func TestSealUniqueSaltAndNonce(t *testing.T) {
	master := randBytes(t, 32)
	pt := []byte("data")
	ct1, err := Seal(master, pt, nil)
	if err != nil {
		t.Fatalf("seal1: %v", err)
	}
	ct2, err := Seal(master, pt, nil)
	if err != nil {
		t.Fatalf("seal2: %v", err)
	}
	if bytes.Equal(ct1[:envelopeSaltSize], ct2[:envelopeSaltSize]) {
		t.Fatal("expected distinct salts")
	}
	if bytes.Equal(ct1[envelopeSaltSize:envelopeSaltSize+envelopeNonceSize], ct2[envelopeSaltSize:envelopeSaltSize+envelopeNonceSize]) {
		t.Fatal("expected distinct nonces")
	}
}

func FuzzEnvelopeRejectMutations(f *testing.F) {
	f.Add([]byte("hello"), []byte("aad"))
	f.Add([]byte(""), []byte(""))
	f.Fuzz(func(t *testing.T, pt, aad []byte) {
		master := randBytes(t, 32)
		ct, err := Seal(master, pt, aad)
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		if _, err := Open(master, ct, aad); err != nil {
			t.Fatalf("open baseline: %v", err)
		}
		if len(ct) == 0 {
			return
		}
		mut := append([]byte(nil), ct...)
		idx := len(pt) % len(mut)
		mut[idx] ^= 0xFF
		if _, err := Open(master, mut, aad); err == nil {
			t.Fatalf("mutation at %d succeeded", idx)
		}
	})
}
