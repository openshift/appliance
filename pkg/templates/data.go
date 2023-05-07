package templates

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/asset/registry"
	"github.com/danielerez/openshift-appliance/pkg/types"
	"github.com/go-openapi/swag"
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

// formatVersion trims the '.z' portion from  aversion if it came in a x.y.z format
// Otherwise, return version with no changes.
func formatVersion(version string) string {
	if strings.Count(version, ".") == 2 {
		lastIdx := strings.LastIndex(version, ".")
		return version[:lastIdx]
	}
	return version
}

func GetImageSetTemplateData(applianceConfig *config.ApplianceConfig, blockedImages string, additionalImages string) interface{} {
	version := formatVersion(applianceConfig.Config.OcpRelease.Version)
	return struct {
		Architectures    string
		ChannelName      string
		MinVersion       string
		MaxVersion       string
		BlockedImages    string
		AdditionalImages string
	}{
		Architectures:    swag.StringValue(applianceConfig.Config.OcpRelease.CpuArchitecture),
		ChannelName:      fmt.Sprintf("%s-%s", swag.StringValue(applianceConfig.Config.OcpRelease.Channel), version),
		MinVersion:       applianceConfig.Config.OcpRelease.Version,
		MaxVersion:       applianceConfig.Config.OcpRelease.Version,
		BlockedImages:    blockedImages,
		AdditionalImages: additionalImages,
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
