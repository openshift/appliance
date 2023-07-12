package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/appliance/pkg/upgrade/annotations"
	"github.com/openshift/appliance/pkg/upgrade/labels"
)

// BundleExtractorBuilder contains the data and logic needed to create bundle extractors. Don't
// create instances of this type directly, use the NewBundleExtractor function instead.
type BundleExtractorBuilder struct {
	logger     logr.Logger
	client     clnt.Client
	node       string
	rootDir    string
	bundleFile string
	bundleDir  string
	serverAddr string
}

// BundleExtractor obtains the upgrade bundle, from a file or from the bundle server, extracts it to
// a directory and marks the node with a label when it finishes. Don't create instances of this type
// directly, use the NewBundleExtractor function instead.
type BundleExtractor struct {
	logger     logr.Logger
	client     clnt.Client
	node       string
	rootDir    string
	bundleFile string
	bundleDir  string
	serverAddr string
}

// NewBundleExtractor creates a builder that can then be used to configure and create bundle
// extractors.
func NewBundleExtractor() *BundleExtractorBuilder {
	return &BundleExtractorBuilder{}
}

// SetLogger sets the logger that the extractor will use to write log messages. This is mandatory.
func (b *BundleExtractorBuilder) SetLogger(value logr.Logger) *BundleExtractorBuilder {
	b.logger = value
	return b
}

// SetClient sets the Kubernetes API client that the extractor will use to write the annotations and
// labels used to report progress and to update the state of the extraction process. This is
// mandatory.
func (b *BundleExtractorBuilder) SetClient(value clnt.Client) *BundleExtractorBuilder {
	b.client = value
	return b
}

// SetNode sets the name of the node where the extractor is running. The extractor will add to
// this node the annotations and labels that indicate the progress and state of the extraction
// process. This is mandatory.
func (b *BundleExtractorBuilder) SetNode(value string) *BundleExtractorBuilder {
	b.node = value
	return b
}

// SetRootDir sets the root directory. This is optional, and when specified all the other
// directories are relative to it. This is intended for running the extractor in a privileged pod
// with the node root filesystem mounted in a regular directory.
func (b *BundleExtractorBuilder) SetRootDir(value string) *BundleExtractorBuilder {
	b.rootDir = value
	return b
}

// SetBundleFile sets the location of the bundle file. If that file exists the extractor will read
// it and will not try to download the bundle from the bundle server. Note that this is mandatory
// even if the file doesn't exist.
func (b *BundleExtractorBuilder) SetBundleFile(value string) *BundleExtractorBuilder {
	b.bundleFile = value
	return b
}

// SetBundleDir sets the directory where the bundle will be extracted. If the directory exists its
// contents will be completely removed and replaced with the new bundle. This is mandatory.
func (b *BundleExtractorBuilder) SetBundleDir(value string) *BundleExtractorBuilder {
	b.bundleDir = value
	return b
}

// SetServerAddr sets the address of the server where the extractor will try to download the bundle
// if the bundle file doesn't exist. This is mandatory.
func (b *BundleExtractorBuilder) SetServerAddr(value string) *BundleExtractorBuilder {
	b.serverAddr = value
	return b
}

// Build uses the data stored in the builder to create and configure a new bundle extractor.
func (b *BundleExtractorBuilder) Build() (result *BundleExtractor, err error) {
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
	if b.bundleFile == "" {
		err = errors.New("bundle file is mandatory")
		return
	}
	if b.bundleDir == "" {
		err = errors.New("bundle directory is mandatory")
		return
	}
	if b.serverAddr == "" {
		err = errors.New("server address is mandatory")
		return
	}

	// Create and populate the object:
	result = &BundleExtractor{
		logger:     b.logger,
		client:     b.client,
		node:       b.node,
		rootDir:    b.rootDir,
		bundleFile: b.bundleFile,
		bundleDir:  b.bundleDir,
		serverAddr: b.serverAddr,
	}
	return
}

func (e *BundleExtractor) Run(ctx context.Context) error {
	// Nothing to do if the bundle directory already exists:
	exists, err := e.checkBundleDir(ctx)
	if err != nil {
		return err
	}
	if exists {
		e.logger.Info(
			"Bundle directory already exists",
			"dir", e.bundleDir,
		)
		return nil
	}

	// Obtain and extract the bundle:
	var reader io.ReadCloser
	reader, err = e.openBundle(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := reader.Close()
		if err != nil {
			e.logger.Error(err, "Failed to close bundle")
		}
	}()
	err = e.extractBundle(ctx, reader)
	if err != nil {
		return err
	}

	// Write the node annotations and labels that indicate the result. The annotation containin
	// the metadata won't contain the full list of images, only the version, architecture and
	// release image. The list of images is very long and not really necessary.
	metadata, err := e.readMetadata(ctx)
	if err != nil {
		return err
	}
	metadata.Images = nil
	err = e.writeResult(ctx, metadata)
	if err != nil {
		return err
	}

	return nil
}

func (e *BundleExtractor) openBundle(ctx context.Context) (reader io.ReadCloser, err error) {
	for {
		reader, err = e.openBundleAttempt(ctx)
		if err == nil && reader != nil {
			return
		}
		if err != nil {
			e.logger.Error(err, "Failed to open bundle, will try again later")
		} else {
			e.logger.Info("Bundle is not yet available, will try again later")
		}
		time.Sleep(10 * time.Second)
	}
}

func (e *BundleExtractor) openBundleAttempt(ctx context.Context) (reader io.ReadCloser, err error) {
	reader, err = e.openBundleFile(ctx)
	if err != nil || reader != nil {
		return
	}
	reader, err = e.openBundleURL(ctx)
	return
}

func (e *BundleExtractor) checkBundleDir(ctx context.Context) (exists bool, err error) {
	dir := e.absolutePath(e.bundleDir)
	_, err = os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
		return
	}
	if err != nil {
		e.logger.Error(
			err,
			"Failed to check if bundle directory exists",
			"dir", dir,
		)
	}
	exists = true
	return
}

func (e *BundleExtractor) openBundleFile(ctx context.Context) (reader io.ReadCloser,
	err error) {
	file := e.absolutePath(e.bundleFile)
	reader, err = os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		reader = nil
		err = nil
	}
	if reader != nil {
		e.logger.Info(
			"Reading bundle from file",
			"file", file,
		)
	}
	return
}

func (e *BundleExtractor) openBundleURL(ctx context.Context) (stream io.ReadCloser, err error) {
	var url string
	url, err = e.selectBundleURL(ctx)
	if err != nil || url == "" {
		return
	}
	e.logger.Info(
		"Selected bundle URL",
		"url", url,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	request.Header.Set("Accept", "application/octet-stream")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	switch response.StatusCode {
	case http.StatusOK:
		e.logger.Info(
			"Reading bundle from URL",
			"url", url,
		)
		stream = response.Body
	default:
		e.logger.Info(
			"Bundle download failed",
			"url", url,
			"status", response.StatusCode,
		)
	}
	return
}

func (e *BundleExtractor) selectBundleURL(ctx context.Context) (result string, err error) {
	// Find the addresses of the servers:
	host, port, err := net.SplitHostPort(e.serverAddr)
	if err != nil {
		return
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return
	}
	urls := make([]string, len(addrs))
	for i, addr := range addrs {
		urls[i] = fmt.Sprintf("http://%s:%s", addr, port)
	}
	e.logger.Info(
		"Server URLs",
		"server", e.serverAddr,
		"urls", urls,
	)

	// Find all the URLs that have the bundle file available:
	var good []string
	for _, url := range urls {
		ok, err := e.checkBundleURL(ctx, url)
		if err != nil {
			e.logger.Error(
				err,
				"Failed to check if bundle file is available",
				"url", url,
			)
			continue
		}
		if ok {
			e.logger.Info(
				"Bundle file is available",
				"url", url,
			)
			good = append(good, url)
		} else {
			e.logger.Info(
				"Bundle file isn't available",
				"url", url,
			)
		}
	}
	if len(good) == 0 {
		e.logger.Info("Bundle file isn't available")
		return
	}
	e.logger.Info(
		"Bundle file is available",
		"urls", good,
	)

	// Randomly select one of the good URLs:
	result = good[rand.Intn(len(good))]
	return
}

func (e *BundleExtractor) checkBundleURL(ctx context.Context, url string) (ok bool, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			e.logger.Error(
				err,
				"Failed to close bundle file check response body",
				"url", url,
			)
		}
	}()
	e.logger.Info(
		"Received response from bundle server",
		"status", response.StatusCode,
	)
	if response.StatusCode != http.StatusOK {
		return
	}
	ok = true
	return
}

func (e *BundleExtractor) extractBundle(ctx context.Context, reader io.ReadCloser) error {
	// Clean the bundle directory:
	dir := e.absolutePath(e.bundleDir)
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	e.logger.Info(
		"Cleaned bundle directory",
		"dir", dir,
	)

	// Create the temporary directory:
	tmp := fmt.Sprintf("%s.tmp", dir)
	err = os.RemoveAll(tmp)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err = os.MkdirAll(tmp, 0755)
	if err != nil {
		return err
	}
	e.logger.Info(
		"Created temporary directory",
		"dir", tmp,
	)

	// Wrap the reader so that we can report the progress:
	reader = &bundleExtractorProgressReader{
		logger:   e.logger,
		client:   e.client,
		nodeName: e.node,
		reader:   reader,
	}

	// Execute the tar command to expand the bundle to the temporary directory:
	path, err := exec.LookPath("tar")
	if err != nil {
		return err
	}
	stdout := os.Stdout
	stderr := os.Stderr
	cmd := &exec.Cmd{
		Path: path,
		Args: []string{
			"tar",
			"--extract",
			"--file=-",
		},
		Dir:    tmp,
		Stdin:  reader,
		Stdout: stdout,
		Stderr: stderr,
	}
	e.logger.Info(
		"Starting bundle extraction",
		"args", cmd.Args,
	)
	err = cmd.Run()
	e.logger.Info(
		"Finished bundle extraction",
		"args", cmd.Args,
	)
	if err != nil {
		return err
	}

	// Now that we finished downloading and extracting the bundle to the temporary directory we
	// can rename it:
	err = os.Rename(tmp, dir)
	if err != nil {
		return err
	}
	e.logger.Info(
		"Renamed temporary directory",
		"from", tmp,
		"to", dir,
	)
	e.logger.Info(
		"Successfully extracted bundle",
		"dir", dir,
	)

	return nil
}

func (c *BundleExtractor) readMetadata(ctx context.Context) (result *Metadata, err error) {
	dir := c.absolutePath(c.bundleDir)
	file := filepath.Join(dir, "metadata.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return
	}
	c.logger.Info(
		"Read metadata",
		"file", file,
		"version", result.Version,
		"arch", result.Arch,
		"images", len(result.Images),
	)
	return
}

func (c *BundleExtractor) writeResult(ctx context.Context, metadata *Metadata) error {
	// Fetch the nodeObject:
	nodeObject := &corev1.Node{}
	nodeKey := clnt.ObjectKey{
		Name: c.node,
	}
	err := c.client.Get(ctx, nodeKey, nodeObject)
	if err != nil {
		return err
	}

	// Apply the patch:
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	metadataText := string(metadataBytes)
	extractedText := strconv.FormatBool(true)
	nodeUpdate := nodeObject.DeepCopy()
	if nodeUpdate.Annotations == nil {
		nodeUpdate.Annotations = map[string]string{}
	}
	nodeUpdate.Annotations[annotations.BundleMetadata] = metadataText
	if nodeUpdate.Labels == nil {
		nodeUpdate.Labels = map[string]string{}
	}
	nodeUpdate.Labels[labels.BundleExtracted] = extractedText
	nodePatch := clnt.MergeFrom(nodeObject)
	err = c.client.Patch(ctx, nodeUpdate, nodePatch)
	if err != nil {
		return err
	}
	c.logger.V(1).Info(
		"Wrote success",
		"node", c.node,
		"metadata", metadataText,
	)
	return nil
}

func (c *BundleExtractor) absolutePath(relPath string) string {
	absPath := relPath
	if c.rootDir != "" {
		absPath = filepath.Join(c.rootDir, relPath)
	}
	return absPath
}

type bundleExtractorProgressReader struct {
	logger   logr.Logger
	client   clnt.Client
	nodeName string
	reader   io.ReadCloser
	last     time.Time
	total    uint64
}

func (r *bundleExtractorProgressReader) Read(p []byte) (n int, err error) {
	if r.total == 0 {
		r.reportProgress("Extraction started")
	}
	n, err = r.reader.Read(p)
	switch {
	case err == io.EOF:
		r.reportProgress("Extraction finished")
	case err != nil:
		r.reportProgress("Extraction failed")
	default:
		r.total += uint64(n)
		if time.Since(r.last) > time.Minute {
			r.reportProgress("Extracted %s", humanize.IBytes(r.total))
		}
	}
	return
}

func (r *bundleExtractorProgressReader) Close() error {
	return r.reader.Close()
}

func (r *bundleExtractorProgressReader) reportProgress(format string, args ...any) {
	// Render the progress message progressMsg:
	progressMsg := fmt.Sprintf(format, args...)

	// Create a patch to add the annotation containing the rendered message:
	patchData, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				annotations.Progress: progressMsg,
			},
		},
	})
	if err != nil {
		r.logger.Error(
			err,
			"Failed to create progress patch",
			"node", r.nodeName,
			"text", progressMsg,
		)
		return
	}
	nodeObj := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.nodeName,
		},
	}
	patchObj := clnt.RawPatch(types.MergePatchType, patchData)

	// Apply the patch:
	err = r.client.Patch(context.Background(), nodeObj, patchObj)
	if err != nil {
		r.logger.Error(
			err,
			"Failed to apply progress patch",
			"node", r.nodeName,
			"text", progressMsg,
		)
		return
	}
	r.logger.V(1).Info(
		"Reported progress",
		"node", r.nodeName,
		"text", progressMsg,
	)

	// Update the last report time:
	r.last = time.Now()
}
