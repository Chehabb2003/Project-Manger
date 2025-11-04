package tests

import (
	"bytes"
	"crypto/rand"
	"testing"

	cr "project-crypto/internal/crypto"
)

func FuzzEnvelope(f *testing.F) {
	f.Add([]byte("hello"), []byte("aad"))
	f.Fuzz(func(t *testing.T, pt, aad []byte) {
		key := make([]byte, 32)
		rand.Read(key)
		ct, err := cr.Seal(key, pt, aad)
		if err != nil {
			t.Skip()
		}
		got, err := cr.Open(key, ct, aad)
		if err != nil {
			t.Fatalf("open err: %v", err)
		}
		if !bytes.Equal(pt, got) {
			t.Fatalf("roundtrip mismatch")
		}
	})
}
