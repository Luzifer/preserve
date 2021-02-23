package main

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type storageLocal struct {
	basePath string
}

func newStorageLocal(basePath string) storageLocal { return storageLocal{basePath} }

func (s storageLocal) GetFile(cachePath string) (io.ReadSeekCloser, error) {
	cachePath = path.Join(s.basePath, cachePath)
	return os.Open(cachePath)
}

func (s storageLocal) LoadMeta(cachePath string) (*meta, error) {
	cachePath = path.Join(s.basePath, cachePath)

	metaPath := strings.Join([]string{cachePath, "meta"}, ".")
	if _, err := os.Stat(metaPath); err != nil {
		return nil, err
	}

	f, err := os.Open(metaPath)
	if err != nil {
		return nil, errors.Wrap(err, "open metadata file")
	}
	defer f.Close()

	out := new(meta)
	return out, errors.Wrap(
		json.NewDecoder(f).Decode(out),
		"decode metadata file",
	)
}

func (s storageLocal) StoreFile(cachePath string, metadata *meta, data io.Reader) (err error) {
	cachePath = path.Join(s.basePath, cachePath)

	if err = os.MkdirAll(path.Dir(cachePath), 0700); err != nil {
		return errors.Wrap(err, "create cache dir")
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return errors.Wrap(err, "create cache file")
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return errors.Wrap(err, "write cache file")
	}

	f, err = os.Create(strings.Join([]string{cachePath, "meta"}, "."))
	if err != nil {
		return errors.Wrap(err, "create cache meta file")
	}
	defer f.Close()

	metadata.LastCached = time.Now()

	return errors.Wrap(
		json.NewEncoder(f).Encode(metadata),
		"write cache meta file",
	)
}
