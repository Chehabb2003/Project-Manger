package platform

type Keychain interface {
	Store(keyID string, priv []byte) error
	Load(keyID string) ([]byte, error)
}

type fileKeychain struct{}

func (f fileKeychain) Store(keyID string, priv []byte) error { return nil }
func (f fileKeychain) Load(keyID string) ([]byte, error)     { return nil, nil }

func NewKeychain() Keychain { return fileKeychain{} }
