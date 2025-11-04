package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("blob not found")

type BlobStore interface {
	Put(ctx context.Context, id string, data []byte) error
	Get(ctx context.Context, id string) ([]byte, error)
	Delete(ctx context.Context, id string) error
}
