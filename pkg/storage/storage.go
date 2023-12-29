// Package storage defines the interface to talk to the storage backends
package storage

import (
	"context"
	"io"
	"time"
)

type (
	// Meta contains the metadata to be written / read
	Meta struct {
		ContentType  string
		LastCached   time.Time
		LastModified time.Time
	}

	// Storage is the interface to implement when building a storage backend
	Storage interface {
		GetFile(ctx context.Context, cachePath string) (io.ReadSeekCloser, error)
		LoadMeta(ctx context.Context, cachePath string) (*Meta, error)
		StoreFile(ctx context.Context, cachePath string, metadata *Meta, data io.Reader) error
	}
)
