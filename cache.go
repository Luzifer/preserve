package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Luzifer/preserve/pkg/storage"
)

const lastSuccessStatus = 299

func renewCache(ctx context.Context, url string) (*storage.Meta, error) {
	cachePath := urlToCachePath(url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching source file: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("closing response body (leaked fd)")
		}
	}()

	if resp.StatusCode > lastSuccessStatus {
		return nil, fmt.Errorf("HTTP status signaled failure: %d", resp.StatusCode)
	}

	lm := time.Now()
	if t, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified")); err == nil {
		lm = t
	}

	metadata := &storage.Meta{
		ContentType:  resp.Header.Get("Content-Type"),
		LastCached:   time.Now(),
		LastModified: lm,
	}

	if err := store.StoreFile(ctx, cachePath, metadata, resp.Body); err != nil {
		return nil, fmt.Errorf("storing file: %w", err)
	}

	return metadata, nil
}

func urlToCachePath(url string) string {
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))
	return path.Join(h[0:2], h)
}
