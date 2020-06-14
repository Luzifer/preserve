package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type meta struct {
	ContentType  string
	LastCached   time.Time
	LastModified time.Time
}

func loadMeta(p string) (*meta, error) {
	var metaPath = strings.Join([]string{p, "meta"}, ".")
	if _, err := os.Stat(metaPath); err != nil {
		return nil, err
	}

	f, err := os.Open(metaPath)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to open metadata file")
	}
	defer f.Close()

	var out = new(meta)
	return out, errors.Wrap(
		json.NewDecoder(f).Decode(out),
		"Unable to decode metadata file",
	)
}

func saveMeta(p string, m meta) error {
	f, err := os.Create(strings.Join([]string{p, "meta"}, "."))
	if err != nil {
		return errors.Wrap(err, "Unable to create cache meta file")
	}
	defer f.Close()

	return errors.Wrap(
		json.NewEncoder(f).Encode(m),
		"Unable to write cache meta file",
	)
}
