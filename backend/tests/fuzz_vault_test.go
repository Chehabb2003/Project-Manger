package tests

import (
	"context"
	"os"
	"testing"

	"project-crypto/internal/vault"
)

func TestVaultCreateUnlock(t *testing.T) {
	path := t.TempDir() + "/test.vlt"
	v := vault.New(path)
	master := []byte("passw0rd!")

	if err := v.Create(context.Background(), master); err != nil {
		t.Fatal(err)
	}
	v.Lock()

	v2 := vault.New(path)
	if err := v2.Unlock(context.Background(), master); err != nil {
		t.Fatal(err)
	}
	defer v2.Lock()

	_, err := v2.AddItem(context.Background(), vault.Item{
		Type: "login",
		Fields: map[string]string{
			"site":     "example.com",
			"username": "alice",
			"password": "secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = os.Remove(path)
}
