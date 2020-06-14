package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
)

func renewCache(url string) (*meta, error) {
	var cachePath = urlToCachePath(url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to fetch source file")
	}

	if resp.StatusCode > 299 {
		return nil, errors.Errorf("HTTP status signaled failure: %d", resp.StatusCode)
	}

	if err = os.MkdirAll(path.Dir(cachePath), 0700); err != nil {
		return nil, errors.Wrap(err, "Unable to create cache dir")
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create cache file")
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, errors.Wrap(err, "Unable to write cache file")
	}

	var lm = time.Now()
	if t, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified")); err == nil {
		lm = t
	}

	metadata := &meta{
		ContentType:  resp.Header.Get("Content-Type"),
		LastCached:   time.Now(),
		LastModified: lm,
	}

	return metadata, saveMeta(cachePath, *metadata)
}
