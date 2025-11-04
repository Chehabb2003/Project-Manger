package vault

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"project-crypto/internal/storage"
)

func BenchmarkVaultAddItem(b *testing.B) {
	ctx := context.Background()
	dir := b.TempDir()
	blobs := storage.NewFileBlobStore(filepath.Join(dir, "blobs"))
	vpath := filepath.Join(dir, "bench.vlt")
	v := NewWithStores(vpath, blobs, nil)
	master := make([]byte, 32)
	rand.Read(master)
	if err := v.Create(ctx, master); err != nil {
		b.Fatalf("create vault: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := Item{
			Type: "login",
			Fields: map[string]string{
				"site":     "example.com",
				"username": "alice",
				"password": "secret",
			},
		}
		if _, err := v.AddItem(ctx, item); err != nil {
			b.Fatalf("add item: %v", err)
		}
	}
}

func BenchmarkVaultGetItem(b *testing.B) {
	ctx := context.Background()
	dir := b.TempDir()
	blobs := storage.NewFileBlobStore(filepath.Join(dir, "blobs"))
	vpath := filepath.Join(dir, "bench.vlt")
	v := NewWithStores(vpath, blobs, nil)
	master := make([]byte, 32)
	rand.Read(master)
	if err := v.Create(ctx, master); err != nil {
		b.Fatalf("create vault: %v", err)
	}
	item := Item{
		Type: "login",
		Fields: map[string]string{
			"site":     "example.com",
			"username": "alice",
			"password": "secret",
		},
	}
	id, err := v.AddItem(ctx, item)
	if err != nil {
		b.Fatalf("add item: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := v.GetItem(ctx, id); err != nil {
			b.Fatalf("get item: %v", err)
		}
	}
}
