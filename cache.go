package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/pkg/errors"
)

func renewCache(url string) (*meta, error) {
	cachePath := urlToCachePath(url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create request")
	}

	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to fetch source file")
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, errors.Errorf("HTTP status signaled failure: %d", resp.StatusCode)
	}

	lm := time.Now()
	if t, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified")); err == nil {
		lm = t
	}

	metadata := &meta{
		ContentType:  resp.Header.Get("Content-Type"),
		LastCached:   time.Now(),
		LastModified: lm,
	}

	return metadata, store.StoreFile(cachePath, metadata, resp.Body)
}

func urlToCachePath(url string) string {
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))
	return path.Join(h[0:2], h)
}
