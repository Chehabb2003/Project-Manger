package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"project-crypto/internal/storage"
)

func randomBytes(tb testing.TB, n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		tb.Fatalf("rand.Read: %v", err)
	}
	return b
}

func TestRotateMasterUpdatesWrapKeepsBlobs(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	blobsDir := filepath.Join(dir, "blobs")
	blobs := storage.NewFileBlobStore(blobsDir)
	vpath := filepath.Join(dir, "vault.vlt")
	v := NewWithStores(vpath, blobs, nil)
	master1 := randomBytes(t, 32)
	if err := v.Create(ctx, master1); err != nil {
		t.Fatalf("create: %v", err)
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
		t.Fatalf("add item: %v", err)
	}

	typed := v.(*vault)
	wrapBefore := append([]byte(nil), typed.header.VRKWrap...)
	blobPath := filepath.Join(blobsDir, id+".blob")
	blBefore, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}

	master2 := randomBytes(t, 32)
	if err := v.RotateMaster(ctx, master2); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	wrapAfter := append([]byte(nil), typed.header.VRKWrap...)
	if bytes.Equal(wrapBefore, wrapAfter) {
		t.Fatal("expected VRKWrap to change after rotation")
	}
	blAfter, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("read blob after: %v", err)
	}
	if !bytes.Equal(blBefore, blAfter) {
		t.Fatal("item blob changed unexpectedly during rotation")
	}

	v2 := NewWithStores(vpath, blobs, nil)
	if err := v2.Unlock(ctx, master2); err != nil {
		t.Fatalf("unlock with rotated master: %v", err)
	}
	got, err := v2.GetItem(ctx, id)
	if err != nil {
		t.Fatalf("get item after rotation: %v", err)
	}
	if got.Fields["password"] != "secret" {
		t.Fatal("unexpected password after rotation")
	}
}
