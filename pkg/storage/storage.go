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
		// GetFile returns the cached file contents for the given cache path.
		GetFile(ctx context.Context, cachePath string) (io.ReadSeekCloser, error)
		// LoadMeta returns metadata for the given cache path.
		LoadMeta(ctx context.Context, cachePath string) (*Meta, error)
		// StoreFile writes the file contents and metadata for the given cache path.
		StoreFile(ctx context.Context, cachePath string, metadata *Meta, data io.Reader) error
	}
)
