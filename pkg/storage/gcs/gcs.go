// Package gcs implements a storage backend saving files in GCS
package gcs

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/Luzifer/preserve/pkg/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	gcsMetaLastCached   = "x-preserve-last-cached"
	gcsMetaLastModified = "x-preserve-last-modified"
)

type nopSeekCloser struct {
	io.ReadSeeker
}

func (nopSeekCloser) Close() error { return nil }

// Storage implements the storage.Storage interface for GCS storage
type Storage struct {
	bucket string
	client *gcs.Client
	prefix string
}

// New returns a new GCS storage backend
func New(bucketURI string) (*Storage, error) {
	uri, err := url.Parse(bucketURI)
	if err != nil {
		return nil, errors.Wrap(err, "parse GCS bucket URI")
	}

	if uri.Scheme != "gs" || uri.Host == "" {
		return nil, errors.New("invalid GCS bucket URI")
	}

	client, err := gcs.NewClient(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "create GCS client")
	}

	return &Storage{
		bucket: uri.Host,
		client: client,
		prefix: strings.TrimLeft(uri.Path, "/"),
	}, nil
}

// GetFile implements the storage.Storage GetFile method
func (s Storage) GetFile(ctx context.Context, cachePath string) (io.ReadSeekCloser, error) {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	r, err := objHdl.NewReader(ctx)
	switch err {
	case nil:
		// This is fine

	case gcs.ErrObjectNotExist:
		return nil, os.ErrNotExist

	default:
		return nil, errors.Wrap(err, "get object reader")
	}
	defer func() {
		if err := r.Close(); err != nil {
			logrus.WithError(err).Error("closing object reeader (leaked fd)")
		}
	}()

	cache := new(bytes.Buffer)
	if _, err = io.Copy(cache, r); err != nil {
		return nil, errors.Wrap(err, "cache object in memory")
	}

	return nopSeekCloser{bytes.NewReader(cache.Bytes())}, nil
}

// LoadMeta implements the storage.Storage LoadMeta method
func (s Storage) LoadMeta(ctx context.Context, cachePath string) (*storage.Meta, error) {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	attrs, err := objHdl.Attrs(ctx)
	switch err {
	case nil:
		// This is fine

	case gcs.ErrObjectNotExist:
		return nil, os.ErrNotExist // Surrounding code reacts on ErrNotExist

	default:
		return nil, errors.Wrap(err, "get object meta")
	}

	out := &storage.Meta{
		ContentType: attrs.ContentType,
	}

	if out.LastCached, err = time.Parse(time.RFC3339Nano, attrs.Metadata[gcsMetaLastCached]); err != nil {
		return nil, errors.Wrap(err, "parse last-cached date")
	}

	if out.LastModified, err = time.Parse(time.RFC3339Nano, attrs.Metadata[gcsMetaLastModified]); err != nil {
		return nil, errors.Wrap(err, "parse last-modified date")
	}

	return out, nil
}

// StoreFile implements the storage.Storage StoreFile method
func (s Storage) StoreFile(ctx context.Context, cachePath string, metadata *storage.Meta, data io.Reader) error {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	w := objHdl.NewWriter(ctx)
	w.ContentType = metadata.ContentType
	w.Metadata = map[string]string{
		gcsMetaLastCached:   metadata.LastCached.Format(time.RFC3339Nano),
		gcsMetaLastModified: metadata.LastModified.Format(time.RFC3339Nano),
	}

	if _, err := io.Copy(w, data); err != nil {
		return errors.Wrap(err, "upload content")
	}

	return errors.Wrap(w.Close(), "finish upload")
}
