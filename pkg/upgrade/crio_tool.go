package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	dreference "github.com/distribution/distribution/v3/reference"
	"github.com/go-logr/logr"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// CRIOToolBuilder contains the data and logic needed to create a tool that helps with management of
// CRI-O. Don't create instances of this type directly, use the NewCRIOTool function instead.
type CRIOToolBuilder struct {
	logger  logr.Logger
	rootDir string
}

// CRIOTool knows how to do certain CRI-O operations, like reloading it and manipulationg
// configuration files. Don't create instances of this type directly, use the NewCRIOTool function
// instead.
type CRIOTool struct {
	logger      logr.Logger
	rootDir     string
	grpcConn    *grpc.ClientConn
	imageClient criv1.ImageServiceClient
}

// NewCRIOTool creates a builder that can then be used to configure and create a CRI-O tool.
func NewCRIOTool() *CRIOToolBuilder {
	return &CRIOToolBuilder{}
}

// SetLogger sets the logger that the tool will use to write log messages. This is mandatory.
func (b *CRIOToolBuilder) SetLogger(value logr.Logger) *CRIOToolBuilder {
	b.logger = value
	return b
}

// SetRootDir sets the root directory. This is optional, and when specified all the other
// directories are relative to it. This is intended for running the cleaner in a privileged pod with
// the node root filesystem mounted in a regular directory.
func (b *CRIOToolBuilder) SetRootDir(value string) *CRIOToolBuilder {
	b.rootDir = value
	return b
}

// Build uses the data stored in the builder to create and configure a new CRI-O tool.
func (b *CRIOToolBuilder) Build() (result *CRIOTool, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}

	// Create the gRPC connection:
	grpcSocket := crioSocket
	if b.rootDir != "" {
		grpcSocket = filepath.Join(b.rootDir, grpcSocket)
	}
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	grpcConn, err := grpc.Dial("unix:"+grpcSocket, grpcOpts...)
	if err != nil {
		return
	}

	// Create the client for the image service:
	imageClient := criv1.NewImageServiceClient(grpcConn)

	// Create and populate the object:
	result = &CRIOTool{
		logger:      b.logger,
		rootDir:     b.rootDir,
		grpcConn:    grpcConn,
		imageClient: imageClient,
	}
	return
}

// Close releases the resources used by the tool, in particular it closes the gRPC connection.
func (t *CRIOTool) Close() error {
	return t.grpcConn.Close()
}

// CreatePinConif creates the configuration file that instructs CRI-O to not garbage collect the
// images corresponding to the given image references.
func (t *CRIOTool) CreatePinConf(refs []string) error {
	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "pinned_images = [\n")
	for i, ref := range refs {
		fmt.Fprintf(buffer, "  \"%s\"", ref)
		if i < len(refs)-1 {
			fmt.Fprintf(buffer, ",")
		}
		fmt.Fprintf(buffer, "\n")
	}
	fmt.Fprintf(buffer, "]\n")
	file := t.absolutePath(crioPinConf)
	data := buffer.Bytes()
	err := os.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	t.logger.Info(
		"Created pinning configuration",
		"file", file,
		"data", string(data),
	)
	return nil
}

// RemovePinConf removes the configuration file that instruct CRI-O to not garbage collect the
// images.
func (t *CRIOTool) RemovePinConf() error {
	file := t.absolutePath(crioPinConf)
	err := os.Remove(file)
	if err != nil {
		return err
	}
	t.logger.Info(
		"Removed mirroring configuration",
		"file", file,
	)
	return nil
}

// CreateMirrorConf creates the configuratoin file that that instructs CRI-O to go to the given
// mirror for the given set of image references.
func (t *CRIOTool) CreateMirrorConf(mirror string, refs []string) error {
	buffer := &bytes.Buffer{}
	index := map[string]dreference.Named{}
	for _, ref := range refs {
		parsed, err := dreference.ParseAnyReference(ref)
		if err != nil {
			return err
		}
		named, ok := parsed.(dreference.Named)
		if !ok {
			return fmt.Errorf("image reference '%s' doesn't contain a name", ref)
		}
		index[named.Name()] = named
	}
	names := maps.Keys(index)
	slices.Sort(names)
	for _, name := range names {
		named := index[name]
		path := dreference.Path(named)
		fmt.Fprintf(buffer, "[[registry]]\n")
		fmt.Fprintf(buffer, "prefix = \"%s\"\n", name)
		fmt.Fprintf(buffer, "location = \"%s\"\n", name)
		fmt.Fprintf(buffer, "\n")
		fmt.Fprintf(buffer, "[[registry.mirror]]\n")
		fmt.Fprintf(buffer, "location = \"%s/%s\"\n", mirror, path)
		fmt.Fprintf(buffer, "insecure = true\n")
		fmt.Fprintf(buffer, "\n")
	}
	file := t.absolutePath(crioMirrorConf)
	data := buffer.Bytes()
	err := os.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	t.logger.Info(
		"Created mirroring configuration",
		"file", file,
		"data", string(data),
	)
	return nil
}

// RemoveMirrorConf removes the configuration file that we use to configure mirroring.
func (l *CRIOTool) RemoveMirrorConf() error {
	file := l.absolutePath(crioMirrorConf)
	err := os.Remove(file)
	if err != nil {
		return err
	}
	l.logger.Info(
		"Removed mirroring configuration",
		"file", file,
	)
	return nil
}

// ReloadService reloads the CRI-O configuration with the equivalent of 'systemctl reload
// crio.service'.
func (t *CRIOTool) ReloadService(ctx context.Context) error {
	before, ok := os.LookupEnv(dbusSystemEnv)
	if ok {
		defer func() {
			err := os.Setenv(dbusSystemEnv, before)
			if err != nil {
				t.logger.Error(
					err,
					"Failed to restore D-Bus environment",
					"var", dbusSystemSocket,
					"value", before,
				)
			}
		}()
	} else {
		defer func() {
			err := os.Unsetenv(dbusSystemEnv)
			if err != nil {
				t.logger.Error(
					err,
					"Failed to clear D-Bus environment",
					"var", dbusSystemEnv,
				)
			}
		}()
	}
	os.Setenv(dbusSystemEnv, "unix:path="+t.absolutePath(dbusSystemSocket))
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	results := make(chan string)
	job, err := conn.ReloadUnitContext(ctx, crioService, "replace", results)
	if err != nil {
		return fmt.Errorf("failed to reload CRI-O: %v", err)
	}
	result := <-results
	if result != "done" {
		return fmt.Errorf(
			"job %d failed to reload CRI-O with result '%s': %v",
			job, result, err,
		)
	}
	t.logger.Info("Reloaded CRI-O")
	return nil
}

// Pull image asks CRI-O to pull the given image references.
func (t *CRIOTool) PullImage(ctx context.Context, ref string) error {
	start := time.Now()
	request := &criv1.PullImageRequest{
		Image: &criv1.ImageSpec{
			Image: ref,
		},
	}
	response, err := t.imageClient.PullImage(ctx, request)
	if err != nil {
		return err
	}
	duration := time.Since(start)
	t.logger.Info(
		"Pulled image",
		"ref", response.ImageRef,
		"duration", duration.String(),
	)
	return nil
}

func (t *CRIOTool) absolutePath(relPath string) string {
	absPath := relPath
	if t.rootDir != "" {
		absPath = filepath.Join(t.rootDir, relPath)
	}
	return absPath
}

const (
	crioService    = "crio.service"
	crioSocket     = "/var/run/crio/crio.sock"
	crioMirrorConf = "/etc/containers/registries.conf.d/999-upgrade-mirror.conf"
	crioPinConf    = "/etc/crio/crio.conf.d/99-upgrade-pin"

	dbusSystemSocket = "/var/run/dbus/system_bus_socket"
	dbusSystemEnv    = "DBUS_SYSTEM_BUS_ADDRESS"
)
