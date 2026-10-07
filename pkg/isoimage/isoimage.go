package isoimage

import (
	"fmt"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

// Create builds a data ISO at outPath from the contents of workDir.
// Unlike isoeditor.Create, this enables DeepDirectories to allow
// directory trees deeper than 8 levels without Rock Ridge relocation.
// This is required for Docker registry V2 directory structures whose
// paths can exceed 8 levels.
func Create(outPath, workDir, volumeLabel string) error {
	minISOSize := 38 * 1024
	d, err := diskfs.Create(outPath, int64(minISOSize), diskfs.SectorSizeDefault)
	if err != nil {
		return fmt.Errorf("failed to create disk image: %w", err)
	}

	d.LogicalBlocksize = 2048
	fspec := disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: volumeLabel,
		WorkDir:     workDir,
	}
	fs, err := d.CreateFilesystem(fspec)
	if err != nil {
		return fmt.Errorf("failed to create ISO filesystem: %w", err)
	}

	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		return fmt.Errorf("not an iso9660 filesystem")
	}

	options := iso9660.FinalizeOptions{
		RockRidge:        true,
		DeepDirectories:  true,
		VolumeIdentifier: volumeLabel,
	}

	return iso.Finalize(options)
}
