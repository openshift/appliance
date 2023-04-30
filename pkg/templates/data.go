package templates

import (
	"encoding/json"

	"github.com/danielerez/openshift-appliance/pkg/asset/registry"
	"github.com/danielerez/openshift-appliance/pkg/types"
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

func GetImageRegistryTemplateData(registryDataPath string) interface{} {
	return struct {
		RegistryDataPath string
	}{
		RegistryDataPath: registryDataPath,
	}
}

func GetBootstrapIgnitionTemplateData(ocpReleaseImage types.ReleaseImage, registryDataPath string) interface{} {
	releaseImageObj := map[string]any{
		"openshift_version": ocpReleaseImage.Version,
		"version":           ocpReleaseImage.Version,
		"cpu_architecture":  ocpReleaseImage.CpuArchitecture,
		"url":               ocpReleaseImage.URL,
	}
	releaseImageArr := []map[string]any{releaseImageObj}
	releaseImages, _ := json.Marshal(releaseImageArr)

	return struct {
		IsBootstrapStep bool

		ReleaseImages, ReleaseImageUrl, RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage           string
		AssistedServiceImage, AssistedInstallerAgentImage, AssistedInstallerControllerImage, AssistedInstallerImage string
	}{
		IsBootstrapStep: true,

		// Registry
		ReleaseImages:    string(releaseImages),
		ReleaseImageUrl:  *ocpReleaseImage.URL,
		RegistryDataPath: registryDataPath,
		RegistryDomain:   registry.RegistryDomain,
		RegistryFilePath: RegistryFilePath,
		RegistryImage:    RegistryImage,

		// AI images
		AssistedServiceImage:             AssistedServiceImage,
		AssistedInstallerAgentImage:      AssistedInstallerAgentImage,
		AssistedInstallerControllerImage: AssistedInstallerControllerImage,
		AssistedInstallerImage:           AssistedInstallerControllerImage,
	}
}
