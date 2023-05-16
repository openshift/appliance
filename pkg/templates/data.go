package templates

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/registry"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/assisted-service/pkg/conversions"
)

func GetUserCfgTemplateData(grubMenuEntryName string, grubDefault int) interface{} {
	return struct {
		GrubTimeout, GrubDefault                 int
		GrubMenuEntryName, RecoveryPartitionName string
	}{
		GrubTimeout:           GrubTimeout,
		GrubDefault:           grubDefault,
		GrubMenuEntryName:     grubMenuEntryName,
		RecoveryPartitionName: RecoveryPartitionName,
	}
}

func GetGuestfishScriptTemplateData(diskSize, recoveryIsoSize, dataPartitionSize int64, baseImageFile, applianceImageFile, recoveryIsoFile, dataIsoFile, cfgFile string) interface{} {
	sectorSize := int64(512)
	// ext4 filesystem has a larger overhead compared to ISO
	// (an inode table for storing metadata, etc. See: https://ext4.wiki.kernel.org/index.php/Ext4_Disk_Layout#Inode_Table)
	ext4FsOverheadPercentage := 1.1

	dataPartitionEndSector := (diskSize*conversions.GibToBytes(1) - conversions.MibToBytes(1)) / sectorSize
	dataPartitionStartSector := dataPartitionEndSector - (dataPartitionSize / sectorSize)
	recoveryPartitionSize := int64(float64(recoveryIsoSize) * ext4FsOverheadPercentage)
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

func GetBootstrapIgnitionTemplateData(ocpReleaseImage types.ReleaseImage, registryDataPath, installIgnitionConfig string) interface{} {
	releaseImageArr := []map[string]any{
		{
			"openshift_version": ocpReleaseImage.Version,
			"version":           ocpReleaseImage.Version,
			"cpu_architecture":  NormalizeCPUArchitecture(*ocpReleaseImage.CpuArchitecture),
			"url":               ocpReleaseImage.URL,
		},
	}
	releaseImages, _ := json.Marshal(releaseImageArr)

	osImageArr := []map[string]any{
		{
			"openshift_version": ocpReleaseImage.Version,
			"cpu_architecture":  NormalizeCPUArchitecture(*ocpReleaseImage.CpuArchitecture),
			"version":           "n/a",
			"url":               "n/a",
		},
	}
	osImages, _ := json.Marshal(osImageArr)

	return struct {
		IsBootstrapStep       bool
		InstallIgnitionConfig string

		ReleaseImages, ReleaseImage, OsImages                             string
		RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage string
	}{
		IsBootstrapStep:       true,
		InstallIgnitionConfig: installIgnitionConfig,

		// Images
		ReleaseImages: string(releaseImages),
		ReleaseImage:  swag.StringValue(ocpReleaseImage.URL),
		OsImages:      string(osImages),

		// Registry
		RegistryDataPath: registryDataPath,
		RegistryDomain:   registry.RegistryDomain,
		RegistryFilePath: RegistryFilePath,
		RegistryImage:    RegistryImage,
	}
}

func GetInstallIgnitionTemplateData(registryDataPath string) interface{} {
	return struct {
		IsBootstrapStep bool

		RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage string
	}{
		IsBootstrapStep: false,

		// Registry
		RegistryDataPath: registryDataPath,
		RegistryDomain:   registry.RegistryDomain,
		RegistryFilePath: RegistryFilePath,
		RegistryImage:    RegistryImage,
	}
}
