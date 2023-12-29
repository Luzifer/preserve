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

	"github.com/Luzifer/preserve/pkg/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		return nil, errors.Wrap(err, "open metadata file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing metadata file (leaked fd)")
		}
	}()

	out := new(storage.Meta)
	return out, errors.Wrap(
		json.NewDecoder(f).Decode(out),
		"decode metadata file",
	)
}

// StoreFile implements the storage.Storage StoreFile method
func (s Storage) StoreFile(_ context.Context, cachePath string, metadata *storage.Meta, data io.Reader) (err error) {
	cachePath = path.Join(s.basePath, cachePath)

	if err = os.MkdirAll(path.Dir(cachePath), storageLocalDirPermission); err != nil {
		return errors.Wrap(err, "create cache dir")
	}

	f, err := os.Create(cachePath) //#nosec:G304 // Safe source of variable
	if err != nil {
		return errors.Wrap(err, "create cache file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing cache file (leaked fd)")
		}
	}()

	if _, err := io.Copy(f, data); err != nil {
		return errors.Wrap(err, "write cache file")
	}

	f, err = os.Create(strings.Join([]string{cachePath, "meta"}, "."))
	if err != nil {
		return errors.Wrap(err, "create cache meta file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing metadata file (leaked fd)")
		}
	}()

	metadata.LastCached = time.Now()

	return errors.Wrap(
		json.NewEncoder(f).Encode(metadata),
		"write cache meta file",
	)
}
