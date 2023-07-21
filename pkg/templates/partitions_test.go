package templates

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/pkg/conversions"
)

var _ = Describe("Test Partitions", func() {
	var (
		testPartitions                         *Partitions
		diskSize, recoveryIsoSize, dataIsoSize int64
	)

	BeforeEach(func() {
		diskSize = 200
		recoveryIsoSize = conversions.GibToBytes(5)
		dataIsoSize = conversions.GibToBytes(30)
		testPartitions = NewPartitions(diskSize, recoveryIsoSize, dataIsoSize)
	})

	It("partitions are aligned to 4K", func() {
		Expect(testPartitions.RecoveryStartSector % sectorAlignmentFactor).To(Equal(int64(0)))
		Expect(testPartitions.DataStartSector % sectorAlignmentFactor).To(Equal(int64(0)))
	})

	It("partitions are not overlapping", func() {
		Expect(testPartitions.RecoveryEndSector < testPartitions.DataStartSector).To(BeTrue())
	})

	It("recovery partition is large enough", func() {
		partitionSize := (testPartitions.RecoveryEndSector - testPartitions.RecoveryStartSector) * sectorSize
		Expect(partitionSize >= recoveryIsoSize).To(BeTrue())
	})

	It("data partition is large enough", func() {
		partitionSize := (testPartitions.DataEndSector - testPartitions.DataStartSector) * sectorSize
		Expect(partitionSize >= dataIsoSize).To(BeTrue())
	})

	It("end of disk image has an empty 1MiB", func() {
		diskSizeInSectors := int64(conversions.GibToBytes(diskSize) / sectorSize)
		emptyBytes := (diskSizeInSectors - testPartitions.DataEndSector) * sectorSize
		Expect(emptyBytes).To(Equal(conversions.MibToBytes(1)))
	})
})
