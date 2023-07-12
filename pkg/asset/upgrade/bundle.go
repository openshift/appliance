package upgrade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bombsimon/logrusr/v4"
	dreference "github.com/distribution/distribution/v3/reference"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/jq"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/upgrade"
	"github.com/openshift/installer/pkg/asset"
)

// Bundle generates the upgrade bundle.
type Bundle struct {
	Tar      *asset.File
	Digest   *asset.File
	Manifest *asset.File
}

var _ asset.Asset = (*Bundle)(nil)

// Name returns a human friendly name for the asset.
func (b *Bundle) Name() string {
	return "Upgrade bundle"
}

// Dependencies returns all the dependencies directly needed to generate the asset.
func (b *Bundle) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate generates the upgrade bundle.
func (b *Bundle) Generate(dependencies asset.Parents) error {
	// Get the logger:
	logger := logrus.StandardLogger()

	// Get the dependencies:
	envCfg := &config.EnvConfig{}
	appCfg := &config.ApplianceConfig{}
	dependencies.Get(envCfg, appCfg)

	// Verify the pieces of the configuration that we need:
	if appCfg.Config.OcpRelease.Version == "" {
		return errors.New("version is mandatory")
	}
	if appCfg.Config.OcpRelease.CpuArchitecture == nil || *appCfg.Config.OcpRelease.CpuArchitecture == "" {
		return errors.New("architecture is mandatory")
	}
	if appCfg.Config.PullSecret == "" {
		return errors.New("pull secret is mandatory")
	}

	// Create the JSON query tool:
	jqTool, err := jq.NewQueryTool().
		SetLogger(logrusr.New(logger)).
		Build()
	if err != nil {
		return err
	}

	// Create the temporary directory for the bundle contents:
	tmpDir := filepath.Join(envCfg.TempDir, "upgrade")
	err = os.Mkdir(tmpDir, 0700)
	if errors.Is(err, os.ErrExist) {
		err = nil
	}
	if err != nil {
		return err
	}

	// Create and run the generator:
	generator := &bundleGenerator{
		logger: logrus.StandardLogger(),
		tmpDir: tmpDir,
		jqTool: jqTool,
		envCfg: envCfg,
		appCfg: appCfg,
	}
	err = generator.run(context.Background())
	if err != nil {
		return err
	}

	// Get the locations of the generated files:
	b.Tar = &asset.File{
		Filename: generator.bundleFile(),
	}
	b.Digest = &asset.File{
		Filename: generator.digestFile(),
	}
	b.Manifest = &asset.File{
		Filename: generator.manifestFile(),
	}

	return nil
}

type bundleGenerator struct {
	logger *logrus.Logger
	tmpDir string
	jqTool *jq.QueryTool
	envCfg *config.EnvConfig
	appCfg *config.ApplianceConfig
}

func (g *bundleGenerator) run(ctx context.Context) error {
	// Find the images:
	g.logger.Infof("Finding images ...")
	releaseImageRef, imageMap, err := g.findImages(ctx)
	if err != nil {
		return err
	}

	// Create the registry:
	g.logger.Infof("Starting registry ...")
	registryServer, err := g.startRegistry(ctx)
	if err != nil {
		return err
	}

	// Download the images:
	err = g.downloadImages(ctx, registryServer, imageMap)
	if err != nil {
		return err
	}

	// Stop the registry:
	g.logger.Infof("Stopping registry ...")
	err = registryServer.Stop(ctx)
	if err != nil {
		return err
	}

	// Write the metadata:
	g.logger.Infof("Writing metadata ...")
	if g.appCfg.Config.OcpRelease.CpuArchitecture == nil {
		return errors.New("CPU architecture is mandatory")
	}
	metadata := &upgrade.Metadata{
		Version: g.appCfg.Config.OcpRelease.Version,
		Arch:    *g.appCfg.Config.OcpRelease.CpuArchitecture,
		Release: releaseImageRef,
		Images:  maps.Values(imageMap),
	}
	err = g.writeMetadata(ctx, metadata)
	if err != nil {
		return err
	}

	// Write the bundle:
	g.logger.Infof("Writing bundle to '%s' ...", g.bundleFile())
	err = g.writeBundle(ctx)
	if err != nil {
		return err
	}

	// Write the digest:
	g.logger.Infof("Writing digest to '%s' ...", g.digestFile())
	err = g.writeDigest(ctx)
	if err != nil {
		return err
	}

	// Write the manifest:
	g.logger.Infof("Writing manifest to '%s' ...", g.manifestFile())
	err = g.writeManifest(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (g *bundleGenerator) startRegistry(ctx context.Context) (result *registry.RegistryServer,
	err error) {
	result, err = registry.NewRegistryServer().
		SetLogger(logrusr.New(g.logger)).
		SetListenAddr("localhost:0").
		SetRootDir(g.tmpDir).
		Build()
	if err != nil {
		return
	}
	err = result.Start(ctx)
	return
}

// findImages finds the release metadata image and from that the rest of the release images. It
// returns a map where the keys are the tags and the values are the full image references.
func (g *bundleGenerator) findImages(ctx context.Context) (releaseImageRef string,
	imageMap map[string]string, err error) {
	// Calculate the release metadata image tag and reference:
	releaseImageTag := fmt.Sprintf(
		"%s-%s",
		g.appCfg.Config.OcpRelease.Version,
		*g.appCfg.Config.OcpRelease.CpuArchitecture,
	)
	releaseImageRef = fmt.Sprintf(
		"%s/%s:%s",
		bundleReleaseImageDomain, bundleReleaseImagePath, releaseImageTag,
	)

	// Use the 'oc adm release info' command to extract the release metadata JSON from the
	// release metadata image:
	ocPath, err := exec.LookPath("oc")
	if err != nil {
		return
	}
	ocStdout := &bytes.Buffer{}
	ocStderr := &bytes.Buffer{}
	ocCmd := exec.Cmd{
		Path: ocPath,
		Args: []string{
			"oc",
			"adm",
			"release",
			"info",
			"--output=json",
			releaseImageRef,
		},
		Stdout: ocStdout,
		Stderr: ocStderr,
	}
	err = ocCmd.Run()
	g.logger.WithFields(logrus.Fields{
		"args":   ocCmd.Args,
		"stdout": ocCmd.String(),
		"stderr": ocCmd.String(),
		"code":   ocCmd.ProcessState.ExitCode(),
	}).Debug("Executed 'oc' command")
	if err != nil {
		return
	}

	// Extract from the release metadata JSON the digest of the release metadata image, so that
	// we can later download it using the digest instead of the tag that we calculated earlier:
	var releaseImageDigest string
	err = g.jqTool.QueryBytes(
		`.digest`,
		ocStdout.Bytes(), &releaseImageDigest,
	)
	if err != nil {
		return
	}
	releaseImageRef = fmt.Sprintf(
		"%s/%s@%s",
		bundleReleaseImageDomain, bundleReleaseImagePath, releaseImageDigest,
	)

	// Extract from the release metadata JSON the references of the rest of the images that are
	// part of the release:
	type ImageInfo struct {
		Tag string `json:"tag"`
		Ref string `json:"ref"`
	}
	var imageInfos []ImageInfo
	err = g.jqTool.QueryBytes(
		`[.references.spec.tags[] | {
			"tag": .name,
			"ref": .from.name
		}]`,
		ocStdout.Bytes(), &imageInfos,
	)
	if err != nil {
		return
	}

	// Build the result map:
	imageMap = map[string]string{
		releaseImageTag: releaseImageRef,
	}
	for _, imageInfo := range imageInfos {
		imageMap[imageInfo.Tag] = imageInfo.Ref
	}

	return
}

// downloadImages copies a set of images into the given registry server. The images are specified
// with a map where the keys are the full image references and the values are the tags.
func (g *bundleGenerator) downloadImages(ctx context.Context,
	registryServer *registry.RegistryServer, imageMap map[string]string) error {
	// Create a temporary directory to store the certificates and the pull secret that we will
	// need to pass to the skopeo command:
	tmpDir, err := os.MkdirTemp("", "*.skopeo")
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			g.logger.WithError(err).WithFields(logrus.Fields{
				"dir": tmpDir,
			}).Error("Failed to remove skopeo temporary directory")
		}
	}()

	// Save the TLS certificate of the registry to the temporary directory, so that we can later
	// pass it to the '--dest-cert-dir' file of the skopeo command.
	certBytes, _ := registryServer.Certificate()
	certDir := filepath.Join(tmpDir, "certs")
	err = os.Mkdir(certDir, 0700)
	if err != nil {
		return err
	}
	certFile := filepath.Join(certDir, "tls.crt")
	err = os.WriteFile(certFile, certBytes, 0400)
	if err != nil {
		return err
	}

	// Save the pull secret to a file in the temporary directory, so that we can later pass it
	// to the '--src-authfile` of the skopeo command:
	authFile := filepath.Join(tmpDir, "auth.json")
	err = os.WriteFile(authFile, []byte(g.appCfg.Config.PullSecret), 0400)
	if err != nil {
		return err
	}

	// Download the images:
	imageTags := maps.Keys(imageMap)
	slices.Sort(imageTags)
	registryAddr := registryServer.ListenAddr()
	for i, srcTag := range imageTags {
		srcRef := imageMap[srcTag]
		srcNamed, err := dreference.ParseNamed(srcRef)
		if err != nil {
			return fmt.Errorf("failed to parse image reference '%s': %w", srcRef, err)
		}
		srcPath := dreference.Path(srcNamed)
		g.logger.Infof(
			"Downloading image '%s:%s' (%d of %d) ...",
			srcPath, srcTag, i+1, len(imageTags),
		)
		dstRef := fmt.Sprintf("%s/%s:%s", registryAddr, srcPath, srcTag)
		err = g.downloadImage(ctx, certDir, authFile, srcRef, dstRef)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *bundleGenerator) downloadImage(ctx context.Context, certDir string, authFile string,
	srcRef, dstRef string) error {
	skopeoPath, err := exec.LookPath("skopeo")
	if err != nil {
		return err
	}
	skopeoStdout := &bytes.Buffer{}
	skopeoStderr := &bytes.Buffer{}
	skopeoCmd := exec.Cmd{
		Path: skopeoPath,
		Args: []string{
			"skopeo",
			"copy",
			fmt.Sprintf("--src-authfile=%s", authFile),
			fmt.Sprintf("--dest-cert-dir=%s", certDir),
			fmt.Sprintf("docker://%s", srcRef),
			fmt.Sprintf("docker://%s", dstRef),
		},
		Stdout: skopeoStdout,
		Stderr: skopeoStderr,
	}
	err = skopeoCmd.Run()
	var skopeoCode int
	if skopeoCmd.ProcessState != nil {
		skopeoCode = skopeoCmd.ProcessState.ExitCode()
	}
	g.logger.WithFields(logrus.Fields{
		"args":   skopeoCmd.Args,
		"stdout": skopeoStdout.String(),
		"stderr": skopeoStderr.String(),
		"code":   skopeoCode,
	}).Debug("Executed 'skopeo' command")
	return err
}

func (g *bundleGenerator) writeMetadata(ctx context.Context, metadata *upgrade.Metadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	file := filepath.Join(g.tmpDir, "metadata.json")
	return os.WriteFile(file, data, 0644)
}

func (g *bundleGenerator) writeBundle(ctx context.Context) error {
	bundleFile := g.bundleFile()
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return err
	}
	tarStdout := &bytes.Buffer{}
	tarStderr := &bytes.Buffer{}
	tarCmd := exec.Cmd{
		Path: tarPath,
		Args: []string{
			"tar",
			fmt.Sprintf("--directory=%s", g.tmpDir),
			"--create",
			fmt.Sprintf("--file=%s", bundleFile),
			"metadata.json",
			"docker",
		},
		Stdout: tarStdout,
		Stderr: tarStderr,
	}
	err = tarCmd.Run()
	var tarCode int
	if tarCmd.ProcessState != nil {
		tarCode = tarCmd.ProcessState.ExitCode()
	}
	g.logger.WithFields(logrus.Fields{
		"args":   tarCmd.Args,
		"stdout": tarStdout.String(),
		"stderr": tarStderr.String(),
		"code":   tarCode,
	}).Debug("Executed 'tar' command")
	return err
}

func (g *bundleGenerator) writeDigest(ctx context.Context) error {
	bundleFile := g.bundleFile()
	digestFile := g.digestFile()
	bundleHash := sha256.New()
	bundleReader, err := os.Open(bundleFile)
	if err != nil {
		return err
	}
	defer func() {
		err := bundleReader.Close()
		if err != nil {
			g.logger.WithFields(logrus.Fields{
				"file": bundleFile,
			}).Error("Failed to close bundle file")
		}
	}()
	_, err = io.Copy(bundleHash, bundleReader)
	if err != nil {
		return err
	}
	bundleSum := hex.EncodeToString(bundleHash.Sum(nil))
	digestWriter, err := os.OpenFile(digestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err := digestWriter.Close()
		if err != nil {
			g.logger.WithError(err).WithFields(logrus.Fields{
				"file": digestFile,
			}).Error("Failed to close digest file")
		}
	}()
	_, err = fmt.Fprintf(digestWriter, "%s  %s\n", bundleSum, filepath.Base(bundleFile))
	if err != nil {
		return err
	}
	return nil
}

func (g *bundleGenerator) writeManifest(context.Context) error {
	content, err := upgrade.TemplatesFS.ReadFile("templates/manifest.yaml")
	if err != nil {
		return err
	}
	manifest := g.manifestFile()
	err = os.WriteFile(manifest, content, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (g *bundleGenerator) bundleFile() string {
	return g.outputBase() + ".tar"
}

func (g *bundleGenerator) digestFile() string {
	return g.outputBase() + ".sha256"
}

func (c *bundleGenerator) manifestFile() string {
	return c.outputBase() + ".yaml"
}

func (g *bundleGenerator) outputBase() string {
	name := fmt.Sprintf(
		"upgrade-%s-%s",
		g.appCfg.Config.OcpRelease.Version, *g.appCfg.Config.OcpRelease.CpuArchitecture,
	)
	return filepath.Join(g.envCfg.AssetsDir, name)
}

const (
	bundleReleaseImageDomain = "quay.io"
	bundleReleaseImagePath   = "openshift-release-dev/ocp-release"
)
