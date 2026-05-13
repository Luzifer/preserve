// Package local implements a storage.Storage backend for local file storage
package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Luzifer/preserve/pkg/storage"
)

const storageLocalDirPermission = 0o700

// Storage implements the storage.Storage interface for local file storage
type Storage struct {
	basePath string
}

// New returns a new local file storage
func New(basePath string) Storage { return Storage{basePath} }

// GetFile implements the storage.Storage GetFile method
func (s Storage) GetFile(_ context.Context, cachePath string) (io.ReadSeekCloser, error) {
	cachePath = path.Join(s.basePath, cachePath)
	rsc, err := os.Open(cachePath) //#nosec:G304 // Safe source of variable
	if err != nil {
		return nil, fmt.Errorf("opening cache file: %w", err)
	}

	return rsc, nil
}

// LoadMeta implements the storage.Storage LoadMeta method
func (s Storage) LoadMeta(_ context.Context, cachePath string) (*storage.Meta, error) {
	cachePath = path.Join(s.basePath, cachePath)

	metaPath := strings.Join([]string{cachePath, "meta"}, ".")
	if _, err := os.Stat(metaPath); err != nil {
		return nil, fmt.Errorf("getting cache file stat: %w", err)
	}

	f, err := os.Open(metaPath) //#nosec:G304 // Safe source of variable
	if err != nil {
		return nil, fmt.Errorf("open metadata file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing metadata file (leaked fd)")
		}
	}()

	out := new(storage.Meta)
	if err = json.NewDecoder(f).Decode(out); err != nil {
		return nil, fmt.Errorf("decode metadata file: %w", err)
	}

	return out, nil
}

// StoreFile implements the storage.Storage StoreFile method
func (s Storage) StoreFile(_ context.Context, cachePath string, metadata *storage.Meta, data io.Reader) (err error) {
	cachePath = path.Join(s.basePath, cachePath)

	if err = os.MkdirAll(path.Dir(cachePath), storageLocalDirPermission); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	f, err := os.Create(cachePath) //#nosec:G304 // Safe source of variable
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing cache file (leaked fd)")
		}
	}()

	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	f, err = os.Create(strings.Join([]string{cachePath, "meta"}, "."))
	if err != nil {
		return fmt.Errorf("create cache meta file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing metadata file (leaked fd)")
		}
	}()

	metadata.LastCached = time.Now()

	if err = json.NewEncoder(f).Encode(metadata); err != nil {
		return fmt.Errorf("write cache meta file: %w", err)
	}

	return nil
}
