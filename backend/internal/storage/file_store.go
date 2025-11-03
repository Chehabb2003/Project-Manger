package storage

import (
	"context"
	"os"
	"path/filepath"
)

type FileBlobStore struct{ dir string }

func NewFileBlobStore(dir string) *FileBlobStore {
	_ = os.MkdirAll(dir, 0700)
	return &FileBlobStore{dir: dir}
}

func (f *FileBlobStore) Put(_ context.Context, id string, data []byte) error {
	return os.WriteFile(filepath.Join(f.dir, id+".blob"), data, 0600)
}

func (f *FileBlobStore) Get(_ context.Context, id string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(f.dir, id+".blob"))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return b, err
}

func (f *FileBlobStore) Delete(_ context.Context, id string) error {
	err := os.Remove(filepath.Join(f.dir, id+".blob"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
