package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	httpHelpers "github.com/Luzifer/go_helpers/http"
	"github.com/Luzifer/preserve/pkg/storage"
	"github.com/Luzifer/preserve/pkg/storage/gcs"
	"github.com/Luzifer/preserve/pkg/storage/local"
	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		BucketURI       string `flag:"bucket-uri" default:"" description:"[gcs] Format: gs://bucket/prefix"`
		Listen          string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel        string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		StorageDir      string `flag:"storage-dir" default:"./data/" description:"[local] Where to store cached files"`
		StorageProvider string `flag:"storage-provider" default:"local" description:"Storage providers to use ('list' to print a list)"`
		UserAgent       string `flag:"user-agent" default:"" description:"Override user-agent"`
		VersionAndExit  bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	store   storage.Storage
	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return fmt.Errorf("parsing cli options: %w", err)
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parsing log-level: %w", err)
	}
	logrus.SetLevel(l)

	return nil
}

func main() {
	var err error

	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		fmt.Printf("preserve %s\n", version) //nolint:forbidigo // Fine here
		os.Exit(0)
	}

	switch cfg.StorageProvider {
	case "gcs":
		if store, err = gcs.New(cfg.BucketURI); err != nil {
			logrus.WithError(err).Fatal("creating GCS storage")
		}

	case "list":
		// Special "provider" to list possible providers
		logrus.Println("Available Storage Providers: gcs, local")
		return

	case "local":
		store = local.New(cfg.StorageDir)

	default:
		logrus.Fatalf("invalid storage provider: %q", cfg.StorageProvider)
	}

	r := mux.NewRouter()
	r.PathPrefix("/latest/").HandlerFunc(handleCacheLatest)
	r.PathPrefix("/").HandlerFunc(handleCacheOnce)

	r.SkipClean(true)

	r.Use(httpHelpers.NewHTTPLogHandler)
	r.Use(httpHelpers.GzipHandler)

	server := http.Server{
		Addr:              cfg.Listen,
		Handler:           r,
		ReadHeaderTimeout: time.Second,
	}

	logrus.WithFields(logrus.Fields{"addr": cfg.Listen, "version": version}).Info("preserve starting...")
	if err = server.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("running HTTP server")
	}
}

func handleCacheLatest(w http.ResponseWriter, r *http.Request) {
	handleCache(w, r, strings.TrimPrefix(r.RequestURI, "/latest/"), true)
}

func handleCacheOnce(w http.ResponseWriter, r *http.Request) {
	handleCache(w, r, strings.TrimPrefix(r.RequestURI, "/"), false)
}

//revive:disable-next-line:flag-parameter // This is fine in this case
func handleCache(w http.ResponseWriter, r *http.Request, uri string, update bool) {
	if strings.HasPrefix(uri, "b64:") {
		u, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(uri, "b64:"))
		if err != nil {
			http.Error(w, "decoding base64 URL", http.StatusBadRequest)
			return
		}
		uri = string(u)
	}

	var (
		cachePath   = urlToCachePath(uri)
		cacheHeader = "HIT"
		logger      = logrus.WithFields(logrus.Fields{
			"url":  uri,
			"path": cachePath,
		})
	)

	if u, err := url.Parse(uri); err != nil || u.Scheme == "" {
		http.Error(w, "parsing requested URL", http.StatusBadRequest)
		return
	}

	logger.Debug("Received request")

	metadata, err := store.LoadMeta(r.Context(), cachePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logrus.WithError(err).Error("loading meta")
		http.Error(w, "accessing entry metadata", http.StatusInternalServerError)
		return
	}

	if update || errors.Is(err, fs.ErrNotExist) {
		logger.Debug("updating cache")
		cacheHeader = "MISS"

		// Using background context to cache the file even in case of the request being aborted
		metadata, err = renewCache(context.Background(), uri) //nolint:contextcheck // See line above
		if err != nil {
			logger.WithError(err).Warn("refreshing file")
		}
	}

	if metadata == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("X-Last-Cached", metadata.LastCached.UTC().Format(http.TimeFormat))
	w.Header().Set("X-Cache", cacheHeader)

	f, err := store.GetFile(r.Context(), cachePath)
	if err != nil {
		logrus.WithError(err).Error("loading cached file")
		http.Error(w, "accessing cache entry", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing storage file (leaked fd)")
		}
	}()

	http.ServeContent(w, r, "", metadata.LastModified, f)
}
