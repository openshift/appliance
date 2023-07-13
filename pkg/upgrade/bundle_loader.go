package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/upgrade/annotations"
	"github.com/openshift/appliance/pkg/upgrade/labels"
)

// BundleLoaderBuilder contains the data and logic needed to create bundle loaders. Don't create
// instances of this type directly, use the NewBundleLoader function instead.
type BundleLoaderBuilder struct {
	logger    logr.Logger
	client    clnt.Client
	node      string
	rootDir   string
	bundleDir string
}

// BundleLoader loads the images from the bundle into the CRI-O container storage directory. Don't
// create instances of this type directly, use the NewBundleLoader function instead.
type BundleLoader struct {
	logger    logr.Logger
	client    clnt.Client
	node      string
	rootDir   string
	bundleDir string
	crioTool  *CRIOTool
}

// NewBundleLoader creates a builder that can then be used to configure and create bundle
// extractors.
func NewBundleLoader() *BundleLoaderBuilder {
	return &BundleLoaderBuilder{}
}

// SetLogger sets the logger that the loader will use to write log messages. This is mandatory.
func (b *BundleLoaderBuilder) SetLogger(value logr.Logger) *BundleLoaderBuilder {
	b.logger = value
	return b
}

// SetClient sets the Kubernetes API client that the loader will use to write the annotations and
// labels used to report progress and to update the state of the loading process. This is mandatory.
func (b *BundleLoaderBuilder) SetClient(value clnt.Client) *BundleLoaderBuilder {
	b.client = value
	return b
}

// SetNode sets the name of the node where the loader is running. The loader will add to this node
// the annotations and labels that indicate the progress and state of the loading process. This is
// mandatory.
func (b *BundleLoaderBuilder) SetNode(value string) *BundleLoaderBuilder {
	b.node = value
	return b
}

// SetRootDir sets the root directory. This is optional, and when specified all the other
// directories are relative to it. This is intended for running the loader in a privileged pod
// with the node root filesystem mounted in a regular directory.
func (b *BundleLoaderBuilder) SetRootDir(value string) *BundleLoaderBuilder {
	b.rootDir = value
	return b
}

// SetBundleDir sets the directory where the bundle has been extracted. If the directory doesn't
// exist the loader will finish with an error. This is mandatory.
func (b *BundleLoaderBuilder) SetBundleDir(value string) *BundleLoaderBuilder {
	b.bundleDir = value
	return b
}

// Build uses the data stored in the builder to create and configure a new bundle loader.
func (b *BundleLoaderBuilder) Build() (result *BundleLoader, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}
	if b.client == nil {
		err = errors.New("client is mandatory")
		return
	}
	if b.node == "" {
		err = errors.New("node name is mandatory")
		return
	}
	if b.bundleDir == "" {
		err = errors.New("bundle directory is mandatory")
		return
	}

	// Create the CRI-O tool:
	crioTool, err := NewCRIOTool().
		SetLogger(b.logger).
		SetRootDir(b.rootDir).
		Build()
	if err != nil {
		err = fmt.Errorf("failed to create CRI-O tool: %w", err)
		return
	}

	// Create and populate the object:
	result = &BundleLoader{
		logger:    b.logger,
		client:    b.client,
		node:      b.node,
		rootDir:   b.rootDir,
		bundleDir: b.bundleDir,
		crioTool:  crioTool,
	}
	return
}

func (l *BundleLoader) Run(ctx context.Context) error {
	// Check that the bundle directory bundleDirExists:
	bundleDirExists, err := l.checkBundleDir(ctx)
	if err != nil {
		return err
	}
	if !bundleDirExists {
		return fmt.Errorf("bundle directory '%s' doesn't exist", l.bundleDir)
	}

	// Read the bundleMetadata:
	bundleMetadata, err := l.readMetadata(ctx)
	if err != nil {
		return err
	}

	// Start the registryServer server:
	registryServer, err := l.startRegistry(ctx)
	if err != nil {
		return err
	}

	// Write the CRI-O configuration and then ask it reload and pull the images:
	l.logger.Info("Populating CRI-O")
	err = l.configureCRIO(ctx, registryServer.ListenAddr(), bundleMetadata.Images)
	if err != nil {
		return err
	}
	err = l.populateCRIO(ctx, bundleMetadata.Images)
	if err != nil {
		return err
	}
	err = l.deconfigureCRIO(ctx)
	if err != nil {
		return err
	}
	l.logger.Info("Populated CRI-O")

	// Stop the registry server:
	err = registryServer.Stop(ctx)
	if err != nil {
		return err
	}
	l.logger.Info("Stopped registry server")

	// Delete the bundle directory:
	err = l.deleteBundle(ctx)
	if err != nil {
		return err
	}

	// Write the node annotations and labels that indicate the result:
	err = l.writeResult(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (l *BundleLoader) checkBundleDir(ctx context.Context) (exists bool, err error) {
	dir := l.absolutePath(l.bundleDir)
	_, err = os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}
	exists = true
	return
}

func (l *BundleLoader) absolutePath(relPath string) string {
	absPath := relPath
	if l.rootDir != "" {
		absPath = filepath.Join(l.rootDir, relPath)
	}
	return absPath
}

func (l *BundleLoader) configureCRIO(ctx context.Context, addr string, refs []string) error {
	// Create the configuration files:
	err := l.crioTool.CreatePinConf(refs)
	if err != nil {
		return err
	}
	err = l.crioTool.CreateMirrorConf(addr, refs)
	if err != nil {
		return err
	}

	// Reload the service:
	return l.crioTool.ReloadService(ctx)
}

func (l *BundleLoader) deconfigureCRIO(ctx context.Context) error {
	// Remove the configuration files. Note that the pinning configuration can't be removed at
	// this point, it will be removed only when the upgrade has been completed.
	err := l.crioTool.RemoveMirrorConf()
	if err != nil {
		return err
	}

	// Reload the service:
	return l.crioTool.ReloadService(ctx)
}

func (l *BundleLoader) populateCRIO(ctx context.Context, refs []string) error {
	for i, ref := range refs {
		err := l.crioTool.PullImage(ctx, ref)
		if err != nil {
			return err
		}
		l.reportProgress(ctx, "Pulled %d of %d images", i+1, len(refs))
	}
	return nil
}

func (l *BundleLoader) readMetadata(ctx context.Context) (result *Metadata, err error) {
	dir := l.absolutePath(l.bundleDir)
	file := filepath.Join(dir, "metadata.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return
	}
	l.logger.Info(
		"Read metadata",
		"file", file,
		"version", result.Version,
		"arch", result.Arch,
		"images", len(result.Images),
	)
	return
}

func (l *BundleLoader) startRegistry(ctx context.Context) (registryServer *registry.RegistryServer,
	err error) {
	dir := l.absolutePath(l.bundleDir)
	registryServer, err = registry.NewRegistryServer().
		SetLogger(l.logger).
		SetListenAddr("localhost:0").
		SetRootDir(dir).
		Build()
	if err != nil {
		return
	}
	err = registryServer.Start(ctx)
	if err != nil {
		return
	}
	l.logger.Info(
		"Started registry server",
		"address", registryServer.ListenAddr(),
		"root", registryServer.RootDir(),
	)
	return
}

func (l *BundleLoader) deleteBundle(ctx context.Context) error {
	dir := l.absolutePath(l.bundleDir)
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	l.logger.Info(
		"Deleted bundle",
		"dir", dir,
	)
	return nil
}

func (l *BundleLoader) writeResult(ctx context.Context) error {
	// Fetch the node:
	nodeObject := &corev1.Node{}
	nodeKey := clnt.ObjectKey{
		Name: l.node,
	}
	err := l.client.Get(ctx, nodeKey, nodeObject)
	if err != nil {
		return err
	}

	// Apply the patch:
	loadedText := strconv.FormatBool(true)
	nodeUpdate := nodeObject.DeepCopy()
	if nodeUpdate.Labels == nil {
		nodeUpdate.Labels = map[string]string{}
	}
	nodeUpdate.Labels[labels.BundleLoaded] = loadedText
	nodePatch := clnt.MergeFrom(nodeObject)
	err = l.client.Patch(ctx, nodeUpdate, nodePatch)
	if err != nil {
		return err
	}
	l.logger.V(1).Info(
		"Wrote success",
		"node", l.node,
	)
	return nil
}

func (l *BundleLoader) reportProgress(ctx context.Context, format string, args ...any) {
	// Render the progress message text:
	text := fmt.Sprintf(format, args...)

	// Create a patch to add the annotation containing the rendered message:
	data, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				annotations.Progress: text,
			},
		},
	})
	if err != nil {
		l.logger.Error(
			err,
			"Failed to create progress patch",
			"node", l.node,
			"text", text,
		)
		return
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: l.node,
		},
	}
	patch := clnt.RawPatch(types.MergePatchType, data)

	// Apply the patch:
	err = l.client.Patch(ctx, node, patch)
	if err != nil {
		l.logger.Error(
			err,
			"Failed to apply progress patch",
			"node", l.node,
			"text", text,
		)
		return
	}
	l.logger.V(1).Info(
		"Reported progress",
		"node", l.node,
		"text", text,
	)
}
