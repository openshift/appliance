package registry

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	dconfiguration "github.com/distribution/distribution/v3/configuration"
	dcontext "github.com/distribution/distribution/v3/context"
	dhandlers "github.com/distribution/distribution/v3/registry/handlers"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	_ "github.com/distribution/distribution/v3/registry/storage/driver/filesystem"
)

// RegistryServerBuilder contains the data and logic needed to build a simple image registry server.
// Don't create instances of this type directly, use the NewRegistryServer function instead.
type RegistryServerBuilder struct {
	logger     logr.Logger
	listenAddr string
	rootDir    string
	certBytes  []byte
	keyBytes   []byte
}

// RegistryServer implements a simple registry server. Don't create instances of this type directly,
// use the NewRegistryServer function instead.
type RegistryServer struct {
	logger      logr.Logger
	logrusEntry *logrus.Entry
	listenAddr  string
	rootDir     string
	tmpDir      string
	certBytes   []byte
	keyBytes    []byte
	listener    net.Listener
	server      *http.Server
}

// NewRegistryServer creates a builder that can then be used to configure and create a new registry
// server.
func NewRegistryServer() *RegistryServerBuilder {
	return &RegistryServerBuilder{}
}

// SetLogger sets the logger that the registry will use to write log messages. This is mandatory.
func (b *RegistryServerBuilder) SetLogger(value logr.Logger) *RegistryServerBuilder {
	b.logger = value
	return b
}

// SetListenAddr sets the address where the registry server will listen. This is mandatory.
func (b *RegistryServerBuilder) SetListenAddr(value string) *RegistryServerBuilder {
	b.listenAddr = value
	return b
}

// SetRootDir sets the root of the directory tree where the registry will store the images. This is
// mandatory.
func (b *RegistryServerBuilder) SetRootDir(value string) *RegistryServerBuilder {
	b.rootDir = value
	return b
}

// SetCertificate sets the TLS certificate and key (in PEM format) that will be used by the server.
// This is optional. If not set then a self signed certificate will be generated.
func (b *RegistryServerBuilder) SetCertificate(cert, key []byte) *RegistryServerBuilder {
	b.certBytes = slices.Clone(cert)
	b.keyBytes = slices.Clone(key)
	return b
}

// Build uses the data stored in the builder to create a new registry.
func (b *RegistryServerBuilder) Build() (result *RegistryServer, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.listenAddr == "" {
		err = errors.New("listen address is mandatory")
		return
	}
	if b.rootDir == "" {
		err = errors.New("root directory is mandatory")
		return
	}
	if b.certBytes != nil && b.keyBytes == nil {
		err = errors.New("key is mandatory when certificate is set")
		return
	}
	if b.keyBytes != nil && b.certBytes == nil {
		err = errors.New("certificate is mandatory when key is set")
		return
	}

	// Create the temporary directory:
	tmpDir, err := os.MkdirTemp("", "*.registry")
	if err != nil {
		return
	}

	// Generate the TLS certificate and keyBytes if needed:
	certBytes, keyBytes := b.certBytes, b.keyBytes
	if b.certBytes == nil && b.keyBytes == nil {
		certBytes, keyBytes, err = b.makeSelfSignedCert()
		if err != nil {
			return
		}
	}

	// Create the logrus log entry that will be passed to the underlying registry server so that
	// all the log messages it generates will be mapped to our logr logger and to the debug
	// level:
	logrusLogger := logrus.New()
	logrusLogger.Out = io.Discard
	logrusLogger.Formatter = &registryServerLogFormatter{}
	logrusLogger.AddHook(&registryServerLogHook{
		logger: b.logger,
	})
	logrusEntry := logrus.NewEntry(logrusLogger)

	// Create and populate the object:
	result = &RegistryServer{
		logger:      b.logger,
		logrusEntry: logrusEntry,
		listenAddr:  b.listenAddr,
		rootDir:     b.rootDir,
		tmpDir:      tmpDir,
		certBytes:   certBytes,
		keyBytes:    keyBytes,
	}
	return
}

func (b *RegistryServerBuilder) makeSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	listenHost, _, err := net.SplitHostPort(b.listenAddr)
	if err != nil {
		return
	}
	listenAddrs, err := net.LookupHost(listenHost)
	if err != nil {
		return
	}
	listenIPs := make([]net.IP, len(listenAddrs))
	for i, listenAddr := range listenAddrs {
		listenIPs[i] = net.ParseIP(listenAddr)
	}
	keyBytes, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return
	}
	timeNow := time.Now()
	certSpec := x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			CommonName: listenHost,
		},
		DNSNames: []string{
			listenHost,
		},
		IPAddresses: listenIPs,
		NotBefore:   timeNow,
		NotAfter:    timeNow.Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}
	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		&certSpec,
		&certSpec,
		&keyBytes.PublicKey,
		keyBytes,
	)
	if err != nil {
		return
	}
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(keyBytes),
	})
	return
}

// ListenAddr returns the address where the registry is listening.
func (s *RegistryServer) ListenAddr() string {
	return s.listener.Addr().String()
}

// RootDir returns the root directory of the registry.
func (s *RegistryServer) RootDir() string {
	return s.rootDir
}

// Certificate returns the TLS certificate and key used by the registry, in PEM format.
func (s *RegistryServer) Certificate() (certBytes, keyBytes []byte) {
	certBytes = slices.Clone(s.certBytes)
	keyBytes = slices.Clone(s.keyBytes)
	return
}

// Start starts the registry.
func (s *RegistryServer) Start(ctx context.Context) error {
	// Start the registry server:
	certFile := filepath.Join(s.tmpDir, "tls.crt")
	err := os.WriteFile(certFile, s.certBytes, 0400)
	if err != nil {
		return err
	}
	keyFile := filepath.Join(s.tmpDir, "tls.key")
	err = os.WriteFile(keyFile, s.keyBytes, 0400)
	if err != nil {
		return err
	}
	s.listener, err = net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	handlerCfg := &dconfiguration.Configuration{}
	handlerCfg.Storage = dconfiguration.Storage{
		"filesystem": dconfiguration.Parameters{
			"rootdirectory": s.rootDir,
		},
	}
	handlerCfg.HTTP.Secret = "42"
	handlerCfg.HTTP.Addr = s.listener.Addr().String()
	handlerCfg.HTTP.TLS.Certificate = certFile
	handlerCfg.HTTP.TLS.Key = keyFile
	handlerCfg.Catalog.MaxEntries = 100
	handlerCtx := dcontext.WithLogger(ctx, s.logrusEntry)
	handlerApp := dhandlers.NewApp(handlerCtx, handlerCfg)
	s.server = &http.Server{
		Handler: handlerApp,
		BaseContext: func(l net.Listener) context.Context {
			return dcontext.WithLogger(context.Background(), s.logrusEntry)
		},
	}
	if err != nil {
		return err
	}
	go func() {
		err = s.server.ServeTLS(s.listener, certFile, keyFile)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error(err, "Failed to serve")
		}
	}()

	return nil
}

// Stop stops the registry.
func (s *RegistryServer) Stop(ctx context.Context) error {
	// Shutdown the server:
	err := s.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	// Remore the temporary directory:
	err = os.RemoveAll(s.tmpDir)
	if err != nil {
		return err
	}

	return nil
}

// registryServerLogFormatter is a log formatter that does nothing. This is used to avoid wasting
// CPU cycles generating the log messages that will be discarded. These messages will instead be
// redirected to our logging framework where they will be generated again.
type registryServerLogFormatter struct {
}

// Make sure we implement the log formatter interface.
var _ logrus.Formatter = (*registryServerLogFormatter)(nil)

func (f *registryServerLogFormatter) Format(entry *logrus.Entry) (result []byte, err error) {
	return
}

// registryServerLogHook is a logrus hook that sends the log messages to our logger while
// transforming all the messages to the debug level.
type registryServerLogHook struct {
	logger logr.Logger
}

// Make sure we implement the log hook interface.
var _ logrus.Hook = (*registryServerLogHook)(nil)

func (m *registryServerLogHook) Fire(entry *logrus.Entry) error {
	fields := make([]any, 2*len(entry.Data))
	i := 0
	for name, value := range entry.Data {
		fields[2*i] = name
		fields[2*i+1] = value
		i++
	}
	m.logger.V(1).Info(entry.Message, fields...)
	return nil
}

func (h *registryServerLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
