package main

import (
	"context"
	"io"
	"time"
)

type meta struct {
	ContentType  string
	LastCached   time.Time
	LastModified time.Time
}

type storage interface {
	GetFile(ctx context.Context, cachePath string) (io.ReadSeekCloser, error)
	LoadMeta(ctx context.Context, cachePath string) (*meta, error)
	StoreFile(ctx context.Context, cachePath string, metadata *meta, data io.Reader) error
}
