package templates

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/registry"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/types"
	"github.com/sirupsen/logrus"
)

func GetUserCfgTemplateData(grubMenuEntryName string, enableFips bool) interface{} {
	var fipsArg string
	if enableFips {
		fipsArg = "fips=1"
	}

	return struct {
		GrubTimeout                              int
		GrubMenuEntryName, RecoveryPartitionName string
		FipsArg                                  string
	}{
		GrubTimeout:           consts.GrubTimeout,
		RecoveryPartitionName: consts.RecoveryPartitionName,
		GrubMenuEntryName:     grubMenuEntryName,
		FipsArg:               fipsArg,
	}
}

func GetGuestfishScriptTemplateData(isCompact bool, diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64,
	baseImageFile, applianceImageFile, recoveryIsoFile, dataIsoFile, userCfgFile, grubCfgFile, tempDir string) interface{} {

	partitionsInfo := NewPartitions().GetAgentPartitions(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize, isCompact)

	return struct {
		ApplianceFile, RecoveryIsoFile, DataIsoFile, CoreOSImage, RecoveryPartitionName, DataPartitionName, ReservedPartitionGUID string
		UserCfgFile, GrubCfgFile, GrubTempDir                                                                                     string
		DiskSize, RecoveryStartSector, RecoveryEndSector, DataStartSector, DataEndSector, RootStartSector, RootEndSector          int64
	}{
		ApplianceFile:         applianceImageFile,
		RecoveryIsoFile:       recoveryIsoFile,
		DataIsoFile:           dataIsoFile,
		DiskSize:              diskSize,
		CoreOSImage:           baseImageFile,
		RecoveryStartSector:   partitionsInfo.RecoveryPartition.StartSector,
		RecoveryEndSector:     partitionsInfo.RecoveryPartition.EndSector,
		DataStartSector:       partitionsInfo.DataPartition.StartSector,
		DataEndSector:         partitionsInfo.DataPartition.EndSector,
		RootStartSector:       partitionsInfo.RootPartition.StartSector,
		RootEndSector:         partitionsInfo.RootPartition.EndSector,
		RecoveryPartitionName: consts.RecoveryPartitionName,
		DataPartitionName:     consts.DataPartitionName,
		ReservedPartitionGUID: consts.ReservedPartitionGUID,
		UserCfgFile:           userCfgFile,
		GrubCfgFile:           grubCfgFile,
		GrubTempDir:           filepath.Join(tempDir, "scripts/grub"),
	}
}

func GetImageSetTemplateData(applianceConfig *config.ApplianceConfig, blockedImages, additionalImages, operators string) interface{} {
	return struct {
		ReleaseImage     string
		BlockedImages    string
		AdditionalImages string
		Operators        string
	}{
		ReleaseImage:     swag.StringValue(applianceConfig.Config.OcpRelease.URL),
		BlockedImages:    blockedImages,
		AdditionalImages: additionalImages,
		Operators:        operators,
	}
}

func GetPinnedImageSetTemplateData(images, role string) interface{} {
	return struct {
		Role   string
		Images string
	}{
		Role:   role,
		Images: images,
	}
}

func GetBootstrapIgnitionTemplateData(isLiveISO, enableInteractiveFlow bool, ocpReleaseImage types.ReleaseImage, installIgnitionConfig, coreosImagePath, rendezvousHostEnvPlaceholder string) interface{} {
	releaseImageArr := []map[string]any{
		{
			"openshift_version": ocpReleaseImage.Version,
			"version":           ocpReleaseImage.Version,
			"cpu_architecture":  swag.StringValue(ocpReleaseImage.CpuArchitecture),
			"url":               ocpReleaseImage.URL,
		},
	}
	releaseImages, _ := json.Marshal(releaseImageArr)

	osImageArr := []map[string]any{
		{
			"openshift_version": ocpReleaseImage.Version,
			"cpu_architecture":  swag.StringValue(ocpReleaseImage.CpuArchitecture),
			"version":           "n/a",
			"url":               "n/a",
		},
	}
	osImages, _ := json.Marshal(osImageArr)

	data := struct {
		IsBootstrapStep              bool
		IsLiveISO                    bool
		EnableInteractiveFlow        bool
		InstallIgnitionConfig        string
		RendezvousHostEnvPlaceholder string

		ReleaseImages, ReleaseImage, OsImages                             string
		RegistryDomain, RegistryFilePath, RegistryImage string

		Partition0, Partition1, Partition2, Partition3 Partition
	}{
		IsBootstrapStep:              true,
		IsLiveISO:                    isLiveISO,
		EnableInteractiveFlow:        enableInteractiveFlow,
		InstallIgnitionConfig:        installIgnitionConfig,
		RendezvousHostEnvPlaceholder: rendezvousHostEnvPlaceholder,

		// Images
		ReleaseImages: string(releaseImages),
		ReleaseImage:  swag.StringValue(ocpReleaseImage.URL),
		OsImages:      string(osImages),

		// Registry
		RegistryDomain:   registry.RegistryDomain,
		RegistryFilePath: consts.RegistryFilePath,
		RegistryImage:    consts.RegistryImage,
	}

	// Fetch base image partitions (Disk image mode)
	if coreosImagePath != "" {
		partitions, err := NewPartitions().GetCoreOSPartitions(coreosImagePath)
		if err != nil {
			logrus.Fatal(err)
		}

		// CoreOS Partitions
		data.Partition0 = partitions[0]
		data.Partition1 = partitions[1]
		data.Partition2 = partitions[2]
		data.Partition3 = partitions[3]
	}

	return data
}

func GetInstallIgnitionTemplateData(isLiveISO bool, corePassHash string) interface{} {
	return struct {
		IsBootstrapStep bool
		IsLiveISO       bool

		RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage string
		CorePassHash, GrubCfgFilePath, UserCfgFilePath                    string
	}{
		IsBootstrapStep: false,
		IsLiveISO:       isLiveISO,

		// Registry
		RegistryDomain:   registry.RegistryDomain,
		RegistryFilePath: consts.RegistryFilePath,
		RegistryImage:    consts.RegistryImage,
		GrubCfgFilePath:  consts.GrubCfgFilePath,
		UserCfgFilePath:  consts.UserCfgFilePath,
		CorePassHash:     corePassHash,
	}
}

func GetDeployIgnitionTemplateData(targetDevice, postScript string, sparseClone, dryRun bool) interface{} {
	return struct {
		ApplianceFileName, ApplianceImageName, ApplianceImageTar string
		TargetDevice, PostScript                                 string
		SparseClone, DryRun                                      bool
	}{
		ApplianceFileName:  consts.ApplianceFileName,
		ApplianceImageName: consts.ApplianceImageName,
		ApplianceImageTar:  consts.ApplianceImageTar,
		TargetDevice:       targetDevice,
		PostScript:         postScript,
		SparseClone:        sparseClone,
		DryRun:             dryRun,
	}
}

func GetRegistryEnv(registryImage, registryData, registryUpgrade string) string {
	return fmt.Sprintf(`REGISTRY_IMAGE=%s
REGISTRY_DATA=%s
REGISTRY_UPGRADE=%s
`, registryImage, registryData, registryUpgrade)
}

func GetUpgradeISOEnv(releaseImage, releaseVersion string) string {
	return fmt.Sprintf(`RELEASE_IMAGE=%s
RELEASE_VERSION=%s
`, releaseImage, releaseVersion)
}
