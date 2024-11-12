package templates

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/conversions"
)

var _ = Describe("Test Partitions", func() {
	var (
		testPartitions                                      *AgentPartitions
		diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64
	)

	BeforeEach(func() {
		diskSize = 200
		baseIsoSize = conversions.GibToBytes(2)
		recoveryIsoSize = conversions.GibToBytes(5)
		dataIsoSize = conversions.GibToBytes(30)
		testPartitions = NewPartitions().GetAgentPartitions(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize)
	})

	It("partitions are aligned to 4K", func() {
		Expect(testPartitions.RecoveryPartition.StartSector % sectorAlignmentFactor).To(Equal(int64(0)))
		Expect(testPartitions.RecoveryPartition.StartSector % sectorAlignmentFactor).To(Equal(int64(0)))
	})

	It("partitions are not overlapping", func() {
		Expect(testPartitions.RecoveryPartition.EndSector < testPartitions.DataPartition.StartSector).To(BeTrue())
	})

	It("recovery partition is large enough", func() {
		partitionSize := (testPartitions.RecoveryPartition.EndSector - testPartitions.RecoveryPartition.StartSector) * sectorSize
		Expect(partitionSize >= recoveryIsoSize).To(BeTrue())
	})

	It("data partition is large enough", func() {
		partitionSize := (testPartitions.DataPartition.EndSector - testPartitions.RecoveryPartition.StartSector) * sectorSize
		Expect(partitionSize >= dataIsoSize).To(BeTrue())
	})

	It("end of disk image has an empty 1MiB", func() {
		diskSizeInSectors := conversions.GibToBytes(diskSize) / sectorSize
		emptyBytes := (diskSizeInSectors - testPartitions.DataPartition.EndSector) * sectorSize
		Expect(emptyBytes).To(Equal(conversions.MibToBytes(1)))
	})
})
