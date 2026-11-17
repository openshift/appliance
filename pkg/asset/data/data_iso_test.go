package data

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/release"
)

func TestBundleManifests(t *testing.T) {
	cases := []struct {
		name         string
		mirrorPath   string
		mapping      []byte
		mappingErr   error
		expectDryRun bool
		expected     []string
	}{
		{
			name:         "oc-mirror run by the appliance",
			mapping:      []byte("docker://quay.io/a@sha256:1=docker://127.0.0.1:5005/a\n"),
			expectDryRun: true,
			expected:     []string{"cluster-resources", "scripts/mirror/imageset.yaml", "mapping.txt"},
		},
		{
			name:         "dry-run failure skips the mapping",
			mappingErr:   errors.New("dry-run boom"),
			expectDryRun: true,
			expected:     []string{"cluster-resources", "scripts/mirror/imageset.yaml"},
		},
		{
			name:       "pre-mirrored path provided",
			mirrorPath: "/mirror",
			expected:   []string{"cluster-resources"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, cacheDir := t.TempDir(), t.TempDir()
			envConfig := &config.EnvConfig{TempDir: tempDir, CacheDir: cacheDir}
			mockRelease := release.NewMockRelease(gomock.NewController(t))
			if tc.expectDryRun {
				mockRelease.EXPECT().GetMappingFile().Return(tc.mapping, tc.mappingErr)
			}

			got := bundleManifests(mockRelease, envConfig, tc.mirrorPath)

			var expected []string
			for _, rel := range tc.expected {
				base := tempDir
				if rel == "cluster-resources" {
					base = cacheDir
				}
				expected = append(expected, filepath.Join(base, rel))
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("bundleManifests() = %v, want %v", got, expected)
			}
			if tc.mapping != nil {
				content, err := os.ReadFile(filepath.Join(tempDir, "mapping.txt"))
				if err != nil || string(content) != string(tc.mapping) {
					t.Fatalf("mapping.txt = %q (err %v), want %q", content, err, tc.mapping)
				}
			}
		})
	}
}
