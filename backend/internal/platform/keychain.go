package platform

// Stubs: wrap/unwrap device private keys with OS keystore.
// Provide per-OS implementations later via build tags.

type Keychain interface {
	Store(keyID string, priv []byte) error
	Load(keyID string) ([]byte, error)
}

type fileKeychain struct{}

func (f fileKeychain) Store(keyID string, priv []byte) error { return nil }
func (f fileKeychain) Load(keyID string) ([]byte, error)     { return nil, nil }

func NewKeychain() Keychain { return fileKeychain{} }
