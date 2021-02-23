package main

import (
	"io"
	"time"
)

type meta struct {
	ContentType  string
	LastCached   time.Time
	LastModified time.Time
}

type storage interface {
	GetFile(cachePath string) (io.ReadSeekCloser, error)
	LoadMeta(cachePath string) (*meta, error)
	StoreFile(cachePath string, metadata *meta, data io.Reader) error
}
