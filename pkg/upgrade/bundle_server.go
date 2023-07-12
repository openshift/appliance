package upgrade

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
)

// BundleServerBuilder contains the data and logic needed to create an HTTP server that serves the
// bundle file. Don't create instances of this type directly, use the NewBundleServer function
// instead.
type BundleServerBuilder struct {
	logger     logr.Logger
	rootDir    string
	bundleFile string
	listenAddr string
}

// BundleServer is an HTTP server that servers the bundle file. Don't instances of this type
// directly, use the NewBundleServer function instead.
type BundleServer struct {
	logger     logr.Logger
	rootDir    string
	bundleFile string
	listenAddr string
}

// NewBundleServer creates a builder that can then be used to configure and create bundle
// servers.
func NewBundleServer() *BundleServerBuilder {
	return &BundleServerBuilder{}
}

// SetLogger sets the logger that the server will use to write log messages. This is mandatory.
func (b *BundleServerBuilder) SetLogger(value logr.Logger) *BundleServerBuilder {
	b.logger = value
	return b
}

// SetRootDir sets the root directory. This is optional, and when specified the bundle file is
// relative to it. This is intended for running the extractor in a privileged pod with the node root
// filesystem mounted in a regular directory.
func (b *BundleServerBuilder) SetRootDir(value string) *BundleServerBuilder {
	b.rootDir = value
	return b
}

// SetBundleFile sets the location of the bundle file. If that file exists the server will respond
// with '200 Ok', otherwise it will return with '404 Not found'. Note that this is mandatory even if
// the file doesn't exist.
func (b *BundleServerBuilder) SetBundleFile(value string) *BundleServerBuilder {
	b.bundleFile = value
	return b
}

// SetListenAddr sets the address where this server should listen. This is mandatory.
func (b *BundleServerBuilder) SetListenAddr(value string) *BundleServerBuilder {
	b.listenAddr = value
	return b
}

// Build uses the data stored in the builder to create and configure a new bundle server.
func (b *BundleServerBuilder) Build() (result *BundleServer, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.bundleFile == "" {
		err = errors.New("bundle file is mandatory")
		return
	}
	if b.listenAddr == "" {
		err = errors.New("listen address is mandatory")
		return
	}

	// Create and populate the object:
	result = &BundleServer{
		logger:     b.logger,
		rootDir:    b.rootDir,
		bundleFile: b.bundleFile,
		listenAddr: b.listenAddr,
	}
	return
}

func (s *BundleServer) Run(ctx context.Context) error {
	handler := &bundleServerHandler{
		logger:     s.logger,
		rootDir:    s.rootDir,
		bundleFile: s.bundleFile,
	}
	return http.ListenAndServe(s.listenAddr, handler)
}

type bundleServerHandler struct {
	logger     logr.Logger
	rootDir    string
	bundleFile string
}

func (h *bundleServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodHead:
		h.serveHead(w, r)
	case http.MethodGet:
		h.serveGet(w, r)
	default:
		h.logger.Info(
			"Method isn't implemented",
			"method", r.Method,
		)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *bundleServerHandler) serveHead(w http.ResponseWriter, r *http.Request) {
	exists, err := h.checkFile()
	if err != nil {
		h.logger.Error(err, "Failed to check file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		h.logger.Info("File doesn't exist")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	h.logger.Info("Sent response")
}

func (h *bundleServerHandler) serveGet(w http.ResponseWriter, r *http.Request) {
	exists, err := h.checkFile()
	if err != nil {
		h.logger.Error(err, "Failed to check file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		h.logger.Info("File doesn't exist")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	file := h.absolutePath(h.bundleFile)
	stream, err := os.Open(file)
	if err != nil {
		h.logger.Error(err, "Failed to open file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		err := stream.Close()
		if err != nil {
			h.logger.Error(err, "Failed to close file")
		}
	}()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	h.logger.Info("Sending file")
	before := time.Now()
	_, err = io.Copy(w, stream)
	if err != nil {
		h.logger.Error(err, "Failed to send file")
		return
	}
	elapsed := time.Since(before)
	h.logger.Info(
		"Sent file",
		"elapsed", elapsed.String(),
	)
}

func (h *bundleServerHandler) checkFile() (exists bool, err error) {
	file := h.absolutePath(h.bundleFile)
	_, err = os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}
	if err != nil {
		return
	}
	exists = true
	return
}

func (h *bundleServerHandler) absolutePath(relative string) string {
	absolute := relative
	if h.rootDir != "" {
		absolute = filepath.Join(h.rootDir, relative)
	}
	return absolute
}
