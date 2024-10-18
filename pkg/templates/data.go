package templates

import (
	"encoding/json"
	"fmt"

	"github.com/go-openapi/swag"
	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/registry"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/types"
	"github.com/sirupsen/logrus"
)

func GetUserCfgTemplateData(grubMenuEntryName string) interface{} {
	return struct {
		GrubTimeout                              int
		GrubMenuEntryName, RecoveryPartitionName string
	}{
		GrubTimeout:           consts.GrubTimeout,
		GrubMenuEntryName:     grubMenuEntryName,
		RecoveryPartitionName: consts.RecoveryPartitionName,
	}
}

func GetGuestfishScriptTemplateData(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize int64, baseImageFile, applianceImageFile, recoveryIsoFile, dataIsoFile, cfgFile string) interface{} {
	partitionsInfo := NewPartitions().GetAgentPartitions(diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize)

	return struct {
		ApplianceFile, RecoveryIsoFile, DataIsoFile, CoreOSImage, RecoveryPartitionName, DataPartitionName, ReservedPartitionGUID, CfgFile string
		DiskSize, RecoveryStartSector, RecoveryEndSector, DataStartSector, DataEndSector, RootStartSector, RootEndSector                   int64
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
		CfgFile:               cfgFile,
	}
}

func GetImageSetTemplateData(applianceConfig *config.ApplianceConfig, blockedImages, additionalImages, operators string) interface{} {
	version := applianceConfig.Config.OcpRelease.Version
	channel := *applianceConfig.Config.OcpRelease.Channel
	return struct {
		Architectures    string
		ChannelName      string
		MinVersion       string
		MaxVersion       string
		BlockedImages    string
		AdditionalImages string
		Operators        string
	}{
		Architectures:    config.GetReleaseArchitectureByCPU(applianceConfig.GetCpuArchitecture()),
		ChannelName:      fmt.Sprintf("%s-%s", channel, toMajorMinor(version)),
		MinVersion:       version,
		MaxVersion:       version,
		BlockedImages:    blockedImages,
		AdditionalImages: additionalImages,
		Operators:        operators,
	}
}

func GetBootstrapIgnitionTemplateData(ocpReleaseImage types.ReleaseImage, registryDataPath, installIgnitionConfig, coreosImagePath string) interface{} {
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

	// Fetch base image partitions
	partitions, err := NewPartitions().GetCoreOSPartitions(coreosImagePath)
	if err != nil {
		logrus.Fatal(err)
	}

	return struct {
		IsBootstrapStep       bool
		InstallIgnitionConfig string

		ReleaseImages, ReleaseImage, OsImages                             string
		RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage string

		Partition0, Partition1, Partition2, Partition3 Partition
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
		RegistryFilePath: consts.RegistryFilePath,
		RegistryImage:    consts.RegistryImage,

		// CoreOS Partitions
		Partition0: partitions[0],
		Partition1: partitions[1],
		Partition2: partitions[2],
		Partition3: partitions[3],
	}
}

func GetInstallIgnitionTemplateData(registryDataPath, corePassHash string) interface{} {
	return struct {
		IsBootstrapStep bool

		RegistryDataPath, RegistryDomain, RegistryFilePath, RegistryImage string
		CorePassHash, GrubCfgFilePath, UserCfgFilePath                    string
	}{
		IsBootstrapStep: false,

		// Registry
		RegistryDataPath: registryDataPath,
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

func GetRegistryEnv(registryData string) string {
	return fmt.Sprintf(`REGISTRY_IMAGE=%s
REGISTRY_DATA=%s
`, consts.RegistryImage, registryData)
}

// Returns version in major.minor format
func toMajorMinor(openshiftVersion string) string {
	v, _ := version.NewVersion(openshiftVersion)
	return fmt.Sprintf("%d.%d", v.Segments()[0], v.Segments()[1])
}
