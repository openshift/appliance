package consts

const (
	MaxOcpVersion = "4.19" // Latest supported version (update on each release)
	MinOcpVersion = "4.14"

	// user.cfg template
	UserCfgTemplateFile = "scripts/grub/user.cfg.template"
	GrubTimeout         = 10
	GrubMenuEntryName   = "Agent-Based Installer"
	// For installation ignition
	GrubMenuEntryNameRecovery = "Recovery: Agent-Based Installer (Reinstall Cluster)"
	GrubCfgFilePath           = "/boot/grub2/grub.cfg"
	UserCfgFilePath           = "/etc/assisted/user.cfg"

	// guestfish.sh template
	GuestfishScriptTemplateFile = "scripts/guestfish/guestfish.sh.template"
	ApplianceFileName           = "appliance.raw"
	RecoveryIsoFileName         = "recovery.iso"
	DataIsoFileName             = "data.iso"
	CoreosImagePattern          = "rhcos-*%s.raw"

	// Appliance Live ISO
	ApplianceLiveIsoFileName = "appliance.iso"

	// ImageSetTemplateFile imageset.yaml.template
	ImageSetTemplateFile = "scripts/mirror/imageset.yaml.template"

	// PinnedImageSetTemplateFile template
	PinnedImageSetTemplateFile = "scripts/mirror/pinned-image-set.yaml.template"
	// PinnedImageSetPattern - for installation ignition
	PinnedImageSetPattern = "/etc/assisted/%s-pinned-image-set.yaml"
	// OcMirrorMappingFileName - name of the mapping file created by oc mirror
	OcMirrorMappingFileName = "mapping.txt"
	// OcMirrorResourcesDir - cluster resources directory created by oc mirror
	OcMirrorResourcesDir = "cluster-resources"
	// MinOcpVersionForPinnedImageSet - minimum version that supports PinnedImageSet
	MinOcpVersionForPinnedImageSet = "4.16"

	// Recovery/Data partitions
	RecoveryPartitionName = "agentboot"
	DataPartitionName     = "agentdata"

	// ReservedPartitionGUID Set partition as Linux reserved partition: https://en.wikipedia.org/wiki/GUID_Partition_Table
	ReservedPartitionGUID = "8DA63339-0007-60C0-C436-083AC8230908"

	// Local registry
	RegistryImage    = "localhost/registry:latest"
	RegistryFilePath = "registry/registry.tar"
	RegistryPort     = 5005

	// Local registry env file
	RegistryEnvPath       = "/etc/assisted/registry.env"
	RegistryDataBootstrap = "/tmp/registry"
	RegistryDataInstall   = "/mnt/agentdata/oc-mirror/install"
	RegistryDataUpgrade   = "/media/upgrade/oc-mirror/install"

	// Deployment ISO
	CoreosIsoName      = "coreos-%s.iso"
	DeployIsoName      = "appliance.iso"
	DeployDir          = "deploy"
	ApplianceImageName = "appliance"
	ApplianceImageTar  = "appliance.tar"
	ApplianceImage     = "quay.io/edge-infrastructure/openshift-appliance:latest"

	// Upgrade ISO
	UpgradeISONamePattern = "upgrade-%s.iso"

	// Appliance config flags (default values)
	EnableDefaultSources  = false
	StopLocalRegistry     = false
	UseRegistryBinary     = false
	CreatePinnedImageSets = false
	EnableFips            = false
	EnableInteractiveFlow = false
	UseDefaultSourceNames = false
)
