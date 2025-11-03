package vault

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	cr "project-crypto/internal/crypto"
	"project-crypto/internal/storage"
)

func (v *vault) AddItem(ctx context.Context, item Item) (string, error) {
	if !v.unlocked {
		return "", ErrNotUnlocked
	}
	// generate DEK
	dek := make([]byte, 32)
	_, _ = rand.Read(dek)
	defer cr.Zero(dek)

	// encrypt payload
	payload := struct {
		Type    string            `json:"type"`
		Fields  map[string]string `json:"fields"`
		Created int64             `json:"created"`
		Updated int64             `json:"updated"`
		Version int               `json:"version"`
	}{
		Type:    item.Type,
		Fields:  item.Fields,
		Created: time.Now().Unix(),
		Updated: time.Now().Unix(),
		Version: 1,
	}
	pt, _ := json.Marshal(payload)

	id := v.newID()
	aad := []byte(fmt.Sprintf("item:%s", id))
	ct, err := cr.SealX(v.dekKey(dek), pt, aad)
	if err != nil {
		return "", err
	}
	// wrap DEK with VRK
	dekWrap, err := cr.SealX(v.vrk[:], dek, []byte("dek-wrap:"+id))
	if err != nil {
		return "", err
	}

	// persist in KD
	v.kd.Items[id] = KDItem{DekWrap: dekWrap}

	// store ciphertext blob via configured store
	if v.store == nil {
		return "", fmt.Errorf("no blob store configured")
	}
	if err := v.store.Put(ctx, id, ct); err != nil {
		return "", err
	}

	// update local meta index
	m := ItemMeta{
		ID:      id,
		Type:    item.Type,
		Created: payload.Created,
		Updated: payload.Updated,
		Version: payload.Version,
	}
	v.meta[id] = m

	// also persist metadata to remote meta store if configured
	if v.metaStore != nil {
		_ = v.metaStore.PutMeta(ctx, storage.ItemMeta{
			ID:      m.ID,
			Type:    m.Type,
			Created: m.Created,
			Updated: m.Updated,
			Version: m.Version,
		})
	}

	return id, v.flushKD()
}

func (v *vault) GetItem(ctx context.Context, id string) (Item, error) {
	if !v.unlocked {
		return Item{}, ErrNotUnlocked
	}
	ki, ok := v.kd.Items[id]
	if !ok {
		return Item{}, fmt.Errorf("item not found: %s", id)
	}
	dek, err := cr.OpenX(v.vrk[:], ki.DekWrap, []byte("dek-wrap:"+id))
	if err != nil {
		return Item{}, err
	}
	defer cr.Zero(dek)

	ct, err := v.store.Get(ctx, id)
	if err != nil {
		return Item{}, err
	}
	aad := []byte("item:" + id)
	pt, err := cr.OpenX(v.dekKey(dek), ct, aad)
	if err != nil {
		return Item{}, err
	}
	var payload struct {
		Type   string            `json:"type"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(pt, &payload); err != nil {
		return Item{}, err
	}
	return Item{Type: payload.Type, Fields: payload.Fields}, nil
}

func (v *vault) UpdateItem(ctx context.Context, id string, upd Item) error {
	if !v.unlocked {
		return ErrNotUnlocked
	}
	ki, ok := v.kd.Items[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}
	dek, err := cr.OpenX(v.vrk[:], ki.DekWrap, []byte("dek-wrap:"+id))
	if err != nil {
		return err
	}
	defer cr.Zero(dek)

	payload := struct {
		Type    string            `json:"type"`
		Fields  map[string]string `json:"fields"`
		Created int64             `json:"created"`
		Updated int64             `json:"updated"`
		Version int               `json:"version"`
	}{
		Type:    upd.Type,
		Fields:  upd.Fields,
		Created: v.meta[id].Created,
		Updated: time.Now().Unix(),
		Version: v.meta[id].Version + 1,
	}
	pt, _ := json.Marshal(payload)
	aad := []byte("item:" + id)
	ct, err := cr.SealX(v.dekKey(dek), pt, aad)
	if err != nil {
		return err
	}
	if err := v.store.Put(ctx, id, ct); err != nil {
		return err
	}
	v.meta[id] = ItemMeta{
		ID:      id,
		Type:    upd.Type,
		Created: payload.Created,
		Updated: payload.Updated,
		Version: payload.Version,
	}
	// optional: also bump remote meta
	if v.metaStore != nil {
		_ = v.metaStore.PutMeta(ctx, storage.ItemMeta{
			ID:      id,
			Type:    upd.Type,
			Created: payload.Created,
			Updated: payload.Updated,
			Version: payload.Version,
		})
	}
	return v.flushKD()
}
