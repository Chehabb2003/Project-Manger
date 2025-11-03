package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
)

type DHKey struct {
	Priv *ecdh.PrivateKey
	Pub  *ecdh.PublicKey
}

func NewX25519() (*DHKey, error) {
	dh := ecdh.X25519()
	priv, err := dh.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := priv.PublicKey()
	return &DHKey{Priv: priv, Pub: pub}, nil
}

func SharedSecret(priv *ecdh.PrivateKey, peer *ecdh.PublicKey) ([]byte, error) {
	return priv.ECDH(peer)
}
