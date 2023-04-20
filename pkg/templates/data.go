package templates

import (
	"github.com/openshift/assisted-service/pkg/conversions"
)

func GetUserCfgTemplateData() interface{} {
	return struct {
		GrubTimeout, GrubDefault                 int
		GrubMenuEntryName, RecoveryPartitionName string
	}{
		GrubTimeout:           GrubTimeout,
		GrubDefault:           GrubDefault,
		GrubMenuEntryName:     GrubMenuEntryName,
		RecoveryPartitionName: RecoveryPartitionName,
	}
}

func GetGuestfishScriptTemplateData(diskSize, recoveryPartitionSize, dataPartitionSize int64, baseImageFile, applianceImageFile, recoveryIsoFile, dataIsoFile, cfgFile string) interface{} {
	sectorSize := int64(512)
	dataPartitionEndSector := (diskSize*conversions.GibToBytes(1) - conversions.MibToBytes(1)) / sectorSize
	dataPartitionStartSector := dataPartitionEndSector - (dataPartitionSize / sectorSize)

	recoveryPartitionEndSector := dataPartitionStartSector - 1
	recoveryPartitionStartSector := recoveryPartitionEndSector - (recoveryPartitionSize / sectorSize)

	return struct {
		ApplianceFile, RecoveryIsoFile, DataIsoFile, CoreOSImage, RecoveryPartitionName, DataPartitionName, ReservedPartitionGUID, CfgFile string
		DiskSize, RecoveryStartSector, RecoveryEndSector, DataStartSector, DataEndSector                                                   int64
	}{
		ApplianceFile:         applianceImageFile,
		RecoveryIsoFile:       recoveryIsoFile,
		DataIsoFile:           dataIsoFile,
		DiskSize:              diskSize,
		CoreOSImage:           baseImageFile,
		RecoveryStartSector:   recoveryPartitionStartSector,
		RecoveryEndSector:     recoveryPartitionEndSector,
		DataStartSector:       dataPartitionStartSector,
		DataEndSector:         dataPartitionEndSector,
		RecoveryPartitionName: RecoveryPartitionName,
		DataPartitionName:     DataPartitionName,
		ReservedPartitionGUID: ReservedPartitionGUID,
		CfgFile:               cfgFile,
	}
}

func GetBootstrapIgnitionTemplateData(registryDataPath string) interface{} {
	return struct {
		RegistryDataPath string
	}{
		RegistryDataPath: registryDataPath,
	}
}
