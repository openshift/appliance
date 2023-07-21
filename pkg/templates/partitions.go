package templates

import (
	"math"

	"github.com/openshift/assisted-service/pkg/conversions"
)

const (
	sectorSize   = int64(512)
	sectorSize64K = int64(64 * 1024)

	// We align the partitions to block size of 64K, as suggested for best performance:
	// https://libguestfs.org/virt-alignment-scan.1.html
	sectorAlignmentFactor = int64(sectorSize64K / sectorSize)
)

type Partitions struct {
	RecoveryStartSector int64
	RecoveryEndSector   int64
	DataStartSector     int64
	DataEndSector       int64
}

func NewPartitions(diskSize, recoveryIsoSize, dataIsoSize int64) *Partitions {
	// ext4 filesystem has a larger overhead compared to ISO
	// (an inode table for storing metadata, etc. See: https://ext4.wiki.kernel.org/index.php/Ext4_Disk_Layout#Inode_Table)
	ext4FsOverheadPercentage := 1.1

	// Calc data partition start/end sectors
	dataEndSector := (conversions.GibToBytes(diskSize) - conversions.MibToBytes(1)) / sectorSize
	dataStartSector := dataEndSector - (dataIsoSize / sectorSize)
	dataStartSector = roundToNearestSector(dataStartSector, sectorAlignmentFactor)

	// Calc recovery partition start/end sectors
	recoveryPartitionSize := int64(float64(recoveryIsoSize) * ext4FsOverheadPercentage)
	recoveryEndSector := dataStartSector - sectorAlignmentFactor
	recoveryStartSector := recoveryEndSector - (recoveryPartitionSize / sectorSize)
	recoveryStartSector = roundToNearestSector(recoveryStartSector, sectorAlignmentFactor)

	return &Partitions{
		RecoveryStartSector: recoveryStartSector,
		RecoveryEndSector:   recoveryEndSector,
		DataStartSector:     dataStartSector,
		DataEndSector:       dataEndSector,
	}
}

// Returns the nearest (and lowest) sector according to a specified alignment factor
// E.g. for 'sector: 19' and 'alignmentFactor: 8' -> returns 16
func roundToNearestSector(sector int64, alignmentFactor int64) int64 {
	sectorFloat := float64(sector)
	alignmentFactorFloat := float64(alignmentFactor)
	return int64(math.Floor(sectorFloat/alignmentFactorFloat) * alignmentFactorFloat)
}
