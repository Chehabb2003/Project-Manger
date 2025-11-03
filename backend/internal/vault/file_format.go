package vault

type Header struct {
	Version int       `json:"version"`
	KDF     KDFHeader `json:"kdf"`
	VRKWrap []byte    `json:"vrk_wrap"` // AEAD_KEK(VRK||...)
	KDCipher []byte   `json:"kd_cipher"`// AEAD_VRK(KeyDirectory)
	Padding []byte    `json:"padding,omitempty"`
}

type KDFHeader struct {
	Algo string `json:"algo"` // "argon2id"
	M    uint32 `json:"m"`
	T    uint32 `json:"t"`
	P    uint8  `json:"p"`
	Salt []byte `json:"salt"`
}

type KeyDirectory struct {
	Items   map[string]KDItem `json:"items"`
	Devices map[string]Device `json:"devices"`
	Policy  Policy            `json:"policy"`
}

type KDItem struct {
	DekWrap []byte `json:"dek_wrap"` // AEAD_VRK(DEK)
	MetaMAC []byte `json:"meta_mac,omitempty"`
}

type Device struct {
	ID         string `json:"id"`
	PubX25519  []byte `json:"pubX25519"`
	PubEd25519 []byte `json:"pubEd25519"`
}

// Item and queries (public API structs)
type Item struct {
	Type   string            `json:"type"`
	Fields map[string]string `json:"fields"`
}

type ItemMeta struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	Version int    `json:"version"`
}

type Query struct {
	Type string // filter by type, optional
}
