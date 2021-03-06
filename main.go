package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	httpHelpers "github.com/Luzifer/go_helpers/v2/http"
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

	store   storage
	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("preserve %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	var err error

	switch cfg.StorageProvider {
	case "gcs":
		if store, err = newStorageGCS(cfg.BucketURI); err != nil {
			log.WithError(err).Fatal("Unable to create GCS storage")
		}

	case "list":
		// Special "provider" to list possible providers
		fmt.Println("Available Storage Providers: gcs, local")
		return

	case "local":
		store = newStorageLocal(cfg.StorageDir)

	default:
		log.Fatalf("Invalid storage provider: %q", cfg.StorageProvider)
	}

	r := mux.NewRouter()
	r.PathPrefix("/latest/").HandlerFunc(handleCacheLatest)
	r.PathPrefix("/").HandlerFunc(handleCacheOnce)

	r.SkipClean(true)

	r.Use(httpHelpers.NewHTTPLogHandler)
	r.Use(httpHelpers.GzipHandler)

	http.ListenAndServe(cfg.Listen, r)
}

func handleCacheLatest(w http.ResponseWriter, r *http.Request) {
	handleCache(w, r, strings.TrimPrefix(r.RequestURI, "/latest/"), true)
}

func handleCacheOnce(w http.ResponseWriter, r *http.Request) {
	handleCache(w, r, strings.TrimPrefix(r.RequestURI, "/"), false)
}

func handleCache(w http.ResponseWriter, r *http.Request, uri string, update bool) {
	var (
		cachePath   = urlToCachePath(uri)
		cacheHeader = "HIT"
		logger      = log.WithFields(log.Fields{
			"url":  uri,
			"path": cachePath,
		})
	)

	if u, err := url.Parse(uri); err != nil || u.Scheme == "" {
		http.Error(w, "Unable to parse requested URL", http.StatusBadRequest)
		return
	}

	logger.Debug("Received request")

	metadata, err := store.LoadMeta(r.Context(), cachePath)
	if err != nil && !os.IsNotExist(err) {
		log.WithError(err).Error("Unable to load meta")
		http.Error(w, "Unable to access entry metadata", http.StatusInternalServerError)
		return
	}

	if update || os.IsNotExist(err) {
		logger.Debug("Updating cache")
		cacheHeader = "MISS"

		// Using background context to cache the file even in case of the request being aborted
		metadata, err = renewCache(context.Background(), uri)
		if err != nil {
			logger.WithError(err).Warn("Unable to refresh file")
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
		log.WithError(err).Error("Unable to load cached file")
		http.Error(w, "Unable to access cache entry", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	http.ServeContent(w, r, "", metadata.LastModified, f)
}
