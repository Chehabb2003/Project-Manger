package vault

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	cr "project-crypto/internal/crypto"
	"project-crypto/internal/storage"
)

type Vault interface {
	Create(ctx context.Context, master []byte) error
	Unlock(ctx context.Context, master []byte) error
	Lock()
	AddItem(ctx context.Context, item Item) (string, error)
	GetItem(ctx context.Context, id string) (Item, error)
	UpdateItem(ctx context.Context, id string, upd Item) error
	List(ctx context.Context, q Query) ([]ItemMeta, error)
	RotateMaster(ctx context.Context, newMaster []byte) error
	DeleteItem(ctx context.Context, id string) error
}

type vault struct {
	path     string
	header   Header
	kd       KeyDirectory
	unlocked bool

	kek [32]byte
	vrk [32]byte

	store     storage.BlobStore
	metaStore storage.MetaStore

	meta map[string]ItemMeta
}

func New(path string) Vault {
	blobDir := "." + filepath.Base(path) + ".blobs"
	return NewWithStores(path, storage.NewFileBlobStore(blobDir), nil)
}

func NewWithStores(path string, blobs storage.BlobStore, meta storage.MetaStore) Vault {
	return &vault{
		path:      path,
		store:     blobs,
		metaStore: meta,
		meta:      make(map[string]ItemMeta),
	}
}

func (v *vault) Create(ctx context.Context, master []byte) error {
	v.header.Version = 2
	kdf := cr.DefaultDesktopKDF()
	v.header.KDF = KDFHeader{
		Algo: "argon2id",
		M:    kdf.M,
		T:    kdf.T,
		P:    kdf.P,
		Salt: kdf.Salt,
	}
	v.kek = cr.DeriveKEK(master, kdf)
	defer zero32(&v.kek)

	_, _ = rand.Read(v.vrk[:])

	vrkWrap, err := cr.Seal(v.kek[:], v.vrk[:], []byte("vrk-wrap"))
	if err != nil {
		return err
	}
	v.header.VRKWrap = vrkWrap

	v.kd = KeyDirectory{
		Items:   map[string]KDItem{},
		Devices: map[string]Device{},
		Policy:  DefaultPolicy(),
	}
	if err := v.flushKD(); err != nil {
		return err
	}
	v.unlocked = true
	return nil
}

func (v *vault) Unlock(ctx context.Context, master []byte) error {
	h, err := readHeader(v.path)
	if err != nil {
		return err
	}
	v.header = h
	kdf := cr.KDFParams{M: h.KDF.M, T: h.KDF.T, P: h.KDF.P, Salt: h.KDF.Salt}
	v.kek = cr.DeriveKEK(master, kdf)

	vrk, err := cr.OpenAny(v.kek[:], v.header.VRKWrap, []byte("vrk-wrap"))
	if err != nil {
		return err
	}
	copy(v.vrk[:], vrk)
	cr.Zero(vrk)

	kdBytes, err := cr.OpenAny(v.vrk[:], v.header.KDCipher, []byte("kd"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(kdBytes, &v.kd); err != nil {
		return err
	}
	v.unlocked = true
	return nil
}

func (v *vault) Lock() {
	v.unlocked = false
	zero32(&v.kek)
	zero32(&v.vrk)
}

func (v *vault) List(ctx context.Context, q Query) ([]ItemMeta, error) {
	if !v.unlocked {
		return nil, ErrNotUnlocked
	}

	if v.metaStore != nil {
		filter := map[string]interface{}{}
		if q.Type != "" {
			filter["type"] = q.Type
		}
		smetas, err := v.metaStore.ListMeta(ctx, filter)
		if err != nil {
			return nil, err
		}

		out := make([]ItemMeta, 0, len(smetas))
		for _, m := range smetas {
			out = append(out, ItemMeta{
				ID:      m.ID,
				Type:    m.Type,
				Created: m.Created,
				Updated: m.Updated,
				Version: m.Version,
			})
		}
		return out, nil
	}

	out := make([]ItemMeta, 0, len(v.meta))
	for _, m := range v.meta {
		if q.Type == "" || q.Type == m.Type {
			out = append(out, m)
		}
	}
	return out, nil
}

func (v *vault) RotateMaster(ctx context.Context, newMaster []byte) error {
	if !v.unlocked {
		return ErrNotUnlocked
	}

	newKDF := cr.DefaultDesktopKDF()
	newKEK := cr.DeriveKEK(newMaster, newKDF)
	defer zero32(&newKEK)

	vrkWrap, err := cr.Seal(newKEK[:], v.vrk[:], []byte("vrk-wrap"))
	if err != nil {
		return err
	}

	v.header.KDF = KDFHeader{
		Algo: "argon2id",
		M:    newKDF.M, T: newKDF.T, P: newKDF.P,
		Salt: newKDF.Salt,
	}
	v.header.VRKWrap = vrkWrap
	return writeHeader(v.path, v.header)
}

func (v *vault) flushKD() error {
	kdBytes, _ := json.Marshal(v.kd)
	ct, err := cr.Seal(v.vrk[:], kdBytes, []byte("kd"))
	if err != nil {
		return err
	}
	v.header.KDCipher = ct
	return writeHeader(v.path, v.header)
}

func (v *vault) newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (v *vault) dekKey(dek []byte) []byte { return dek }

func zero32(x *[32]byte) {
	for i := range x {
		x[i] = 0
	}
}

func (v *vault) DeleteItem(ctx context.Context, id string) error {
	if !v.unlocked {
		return ErrNotUnlocked
	}
	delete(v.kd.Items, id)
	if v.store != nil {
		_ = v.store.Delete(ctx, id)
	}
	delete(v.meta, id)
	return v.flushKD()
}
