package main

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
	"github.com/pkg/errors"
)

const (
	gcsMetaLastCached   = "x-preserve-last-cached"
	gcsMetaLastModified = "x-preserve-last-modified"
)

type nopSeekCloser struct {
	io.ReadSeeker
}

func (nopSeekCloser) Close() error { return nil }

type storageGCS struct {
	bucket string
	client *gcs.Client
	prefix string
}

func newStorageGCS(bucketURI string) (*storageGCS, error) {
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

	return &storageGCS{
		bucket: uri.Host,
		client: client,
		prefix: strings.TrimLeft(uri.Path, "/"),
	}, nil
}

func (s storageGCS) GetFile(cachePath string) (io.ReadSeekCloser, error) {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	r, err := objHdl.NewReader(context.Background())
	switch err {
	case nil:
		// This is fine

	case gcs.ErrObjectNotExist:
		return nil, os.ErrNotExist

	default:
		return nil, errors.Wrap(err, "get object reader")
	}
	defer r.Close()

	cache := new(bytes.Buffer)
	if _, err = io.Copy(cache, r); err != nil {
		return nil, errors.Wrap(err, "cache object in memory")
	}

	return nopSeekCloser{bytes.NewReader(cache.Bytes())}, nil
}

func (s storageGCS) LoadMeta(cachePath string) (*meta, error) {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	attrs, err := objHdl.Attrs(context.Background())
	switch err {
	case nil:
		// This is fine

	case gcs.ErrObjectNotExist:
		return nil, os.ErrNotExist // Surrounding code reacts on ErrNotExist

	default:
		return nil, errors.Wrap(err, "get object meta")
	}

	out := &meta{
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

func (s storageGCS) StoreFile(cachePath string, metadata *meta, data io.Reader) error {
	cachePath = strings.TrimLeft(path.Join(s.prefix, cachePath), "/")
	objHdl := s.client.Bucket(s.bucket).Object(cachePath)

	w := objHdl.NewWriter(context.Background())
	w.ObjectAttrs.ContentType = metadata.ContentType
	w.ObjectAttrs.Metadata = map[string]string{
		gcsMetaLastCached:   metadata.LastCached.Format(time.RFC3339Nano),
		gcsMetaLastModified: metadata.LastModified.Format(time.RFC3339Nano),
	}

	if _, err := io.Copy(w, data); err != nil {
		return errors.Wrap(err, "upload content")
	}

	return errors.Wrap(w.Close(), "finish upload")
}
