package templates

import (
	"math"

	"github.com/diskfs/go-diskfs"
	"github.com/openshift/assisted-service/pkg/conversions"
)

const (
	sectorSize    = int64(512)
	sectorSize64K = int64(64 * 1024)

	// We align the partitions to block size of 64K, as suggested for best performance:
	// https://libguestfs.org/virt-alignment-scan.1.html
	sectorAlignmentFactor = int64(sectorSize64K / sectorSize)
)

type Partitions interface {
	GetAgentPartitions(diskSize, recoveryIsoSize, dataIsoSize int64) *AgentPartitions
	GetCoreOSPartitions(coreosImagePath string) ([]Partition, error)
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

func (p *partitions) GetAgentPartitions(diskSize, recoveryIsoSize, dataIsoSize int64) *AgentPartitions {
	// Calc data partition start/end sectors
	dataEndSector := (conversions.GibToBytes(diskSize) - conversions.MibToBytes(1)) / sectorSize
	dataStartSector := dataEndSector - (dataIsoSize / sectorSize)
	dataStartSector = roundToNearestSector(dataStartSector, sectorAlignmentFactor)

	// Calc recovery partition start/end sectors
	recoveryPartitionSize := int64(float64(recoveryIsoSize))
	recoveryEndSector := dataStartSector - sectorAlignmentFactor
	recoveryStartSector := recoveryEndSector - (recoveryPartitionSize / sectorSize)
	recoveryStartSector = roundToNearestSector(recoveryStartSector, sectorAlignmentFactor)

	// Calc root partition start/end sectors
	rootPartitionSize := sectorSize64K
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

// Returns the nearest (and lowest) sector according to a specified alignment factor
// E.g. for 'sector: 19' and 'alignmentFactor: 8' -> returns 16
func roundToNearestSector(sector int64, alignmentFactor int64) int64 {
	sectorFloat := float64(sector)
	alignmentFactorFloat := float64(alignmentFactor)
	return int64(math.Floor(sectorFloat/alignmentFactorFloat) * alignmentFactorFloat)
}
