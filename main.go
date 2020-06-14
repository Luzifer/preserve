package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		StorageDir     string `flag:"storage-dir" default:"./data/" description:"Where to store cached files"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

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
	r := mux.NewRouter()
	r.PathPrefix("/latest/").HandlerFunc(handleCacheLatest)
	r.PathPrefix("/").HandlerFunc(handleCacheOnce)

	r.SkipClean(true)

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

	metadata, err := loadMeta(cachePath)
	if err != nil && !os.IsNotExist(err) {
		log.WithError(err).Error("Unable to load meta")
		http.Error(w, "Unable to access entry metadata", http.StatusInternalServerError)
		return
	}

	if update || os.IsNotExist(err) {
		logger.Debug("Updating cache")
		cacheHeader = "MISS"

		metadata, err = renewCache(uri)
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

	f, err := os.Open(cachePath)
	if err != nil {
		log.WithError(err).Error("Unable to load cached file")
		http.Error(w, "Unable to access cache entry", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	http.ServeContent(w, r, "", metadata.LastModified, f)
}

func urlToCachePath(url string) string {
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))
	return path.Join(cfg.StorageDir, h[0:2], h)
}
