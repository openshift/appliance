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

func GetGuestfishScriptTemplateData(diskSize, recoveryPartitionSize int64, baseImageFile, applianceImageFile, recoveryIsoFile, cfgFile string) interface{} {
	sectorSize := int64(512)
	recoveryPartitionEndSector := (diskSize*conversions.GibToBytes(1) - conversions.MibToBytes(1)) / sectorSize
	recoveryPartitionStartSector := recoveryPartitionEndSector - (recoveryPartitionSize / sectorSize)

	return struct {
		ApplianceFile, RecoveryIsoFile, CoreOSImage, RecoveryPartitionName, ReservedPartitionGUID, CfgFile string
		DiskSize, StartSector, EndSector                                                                   int64
	}{
		ApplianceFile:         applianceImageFile,
		RecoveryIsoFile:       recoveryIsoFile,
		DiskSize:              diskSize,
		CoreOSImage:           baseImageFile,
		StartSector:           recoveryPartitionStartSector,
		EndSector:             recoveryPartitionEndSector,
		RecoveryPartitionName: RecoveryPartitionName,
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
