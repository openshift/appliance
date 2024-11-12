package templates

import (
	"math"

	"github.com/diskfs/go-diskfs"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/conversions"
	"github.com/sirupsen/logrus"
)

const (
	sectorSize    = int64(512)
	sectorSize64K = int64(64 * 1024)

	// We align the partitions to block size of 64K, as suggested for best performance:
	// https://libguestfs.org/virt-alignment-scan.1.html
	sectorAlignmentFactor = sectorSize64K / sectorSize
)

type Partitions interface {
	GetAgentPartitions(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64) *AgentPartitions
	GetCoreOSPartitions(coreosImagePath string) ([]Partition, error)
	GetBootPartitionsSize(baseImageFile string) int64
}

type Partition struct {
	StartSector, EndSector, Size int64
}

type AgentPartitions struct {
	RecoveryPartition, DataPartition, RootPartition *Partition
}

type partitions struct {
}

func NewPartitions() Partitions {
	return &partitions{}
}

func (p *partitions) GetAgentPartitions(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64) *AgentPartitions {
	// Calc data partition start/end sectors
	dataEndSector := (conversions.GibToBytes(diskSize) - conversions.MibToBytes(1)) / sectorSize
	dataStartSector := dataEndSector - (dataIsoSize / sectorSize)
	dataStartSector = roundToNearestSector(dataStartSector, sectorAlignmentFactor)

	// Calc recovery partition start/end sectors
	recoveryEndSector := dataStartSector - sectorAlignmentFactor
	recoveryStartSector := recoveryEndSector - (recoveryIsoSize / sectorSize)
	recoveryStartSector = roundToNearestSector(recoveryStartSector, sectorAlignmentFactor)

	// Calc root partition start/end sectors
	rootPartitionSize := p.getRootPartitionSize(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize)
	rootEndSector := recoveryStartSector - sectorAlignmentFactor
	rootStartSector := rootEndSector - (rootPartitionSize / sectorSize)
	rootStartSector = roundToNearestSector(rootStartSector, sectorAlignmentFactor)

	return &AgentPartitions{
		RecoveryPartition: &Partition{StartSector: recoveryStartSector, EndSector: recoveryEndSector},
		DataPartition:     &Partition{StartSector: dataStartSector, EndSector: dataEndSector},
		RootPartition:     &Partition{StartSector: rootStartSector, EndSector: rootEndSector},
	}
}

func (p *partitions) GetCoreOSPartitions(coreosImagePath string) ([]Partition, error) {
	partitionsInfo := []Partition{}

	disk, err := diskfs.Open(coreosImagePath)
	if err != nil {
		return nil, err
	}
	partitionTable, err := disk.GetPartitionTable()
	if err != nil {
		return nil, err
	}

	partitions := partitionTable.GetPartitions()
	for _, partition := range partitions {
		partitionsInfo = append(partitionsInfo, Partition{
			StartSector: partition.GetStart() / sectorSize,
			Size:        partition.GetSize() / sectorSize,
		})
	}

	// Root partition should be at least 8GiB
	// (https://docs.fedoraproject.org/en-US/fedora-coreos/storage/)
	partitionsInfo[3].Size = conversions.GibToBytes(8) / sectorSize

	return partitionsInfo, nil
}

func (p *partitions) GetBootPartitionsSize(baseImageFile string) int64 {
	partitions, err := p.GetCoreOSPartitions(baseImageFile)
	if err != nil {
		logrus.Fatal(err)
	}

	// Calc base disk image size in bytes (including an additional overhead for alignment)
	return sectorSize*(partitions[0].Size+partitions[1].Size+partitions[2].Size) + conversions.MibToBytes(1)
}

func (p *partitions) getRootPartitionSize(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64) int64 {
	if diskSize < config.MinDiskSize {
		// When using a compact disk image, the root partition is resized during cloning
		return sectorSize64K
	}

	// Calc root partition size
	return conversions.GbToBytes(diskSize) - (baseIsoSize + recoveryIsoSize + dataIsoSize)
}

// Returns the nearest (and lowest) sector according to a specified alignment factor
// E.g. for 'sector: 19' and 'alignmentFactor: 8' -> returns 16
func roundToNearestSector(sector int64, alignmentFactor int64) int64 {
	sectorFloat := float64(sector)
	alignmentFactorFloat := float64(alignmentFactor)
	return int64(math.Floor(sectorFloat/alignmentFactorFloat) * alignmentFactorFloat)
}
