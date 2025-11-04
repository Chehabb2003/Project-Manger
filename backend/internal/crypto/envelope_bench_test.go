package crypto

import (
	"crypto/rand"
	"testing"
)

func BenchmarkEnvelopeSeal1KB(b *testing.B) {
	master := make([]byte, 32)
	rand.Read(master)
	pt := make([]byte, 1024)
	rand.Read(pt)
	b.SetBytes(int64(len(pt)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Seal(master, pt, nil); err != nil {
			b.Fatalf("seal failed: %v", err)
		}
	}
}

func BenchmarkEnvelopeOpen1KB(b *testing.B) {
	master := make([]byte, 32)
	rand.Read(master)
	pt := make([]byte, 1024)
	rand.Read(pt)
	ciphertext, err := Seal(master, pt, nil)
	if err != nil {
		b.Fatalf("seal failed: %v", err)
	}
	b.SetBytes(int64(len(pt)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Open(master, ciphertext, nil); err != nil {
			b.Fatalf("open failed: %v", err)
		}
	}
}

func BenchmarkEnvelopeSeal16KB(b *testing.B) {
	master := make([]byte, 32)
	rand.Read(master)
	pt := make([]byte, 16*1024)
	rand.Read(pt)
	b.SetBytes(int64(len(pt)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Seal(master, pt, nil); err != nil {
			b.Fatalf("seal failed: %v", err)
		}
	}
}

func BenchmarkEnvelopeOpen16KB(b *testing.B) {
	master := make([]byte, 32)
	rand.Read(master)
	pt := make([]byte, 16*1024)
	rand.Read(pt)
	ciphertext, err := Seal(master, pt, nil)
	if err != nil {
		b.Fatalf("seal failed: %v", err)
	}
	b.SetBytes(int64(len(pt)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Open(master, ciphertext, nil); err != nil {
			b.Fatalf("open failed: %v", err)
		}
	}
}
