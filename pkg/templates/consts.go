package templates

const (
	// user.cfg template
	UserCfgTemplateFile = "scripts/guestfish/user.cfg.template"
	GrubTimeout         = 10
	GrubDefault         = 0
	GrubMenuEntryName   = "Agent-Based Installer"

	// guestfish.sh template
	GuestfishScriptTemplateFile = "scripts/guestfish/guestfish.sh.template"
	ApplianceFileName           = "appliance.raw"
	RecoveryIsoFileName         = "recovery.iso"
	DataIsoFileName             = "data.iso"

	// ImageSetBootstrapTemplateFile imageset-bootstrap.yaml.template
	ImageSetBootstrapTemplateFile = "scripts/mirror/imageset-bootstrap.yaml.template"

	// ImageSetReleaseTemplateFile imageset-release.yaml.template
	ImageSetReleaseTemplateFile = "scripts/mirror/imageset-release.yaml.template"

	// Recovery/Data partitions
	RecoveryPartitionName = "agentboot"
	DataPartitionName     = "agentdata"

	// ReservedPartitionGUID Set partition as Linux reserved partition: https://en.wikipedia.org/wiki/GUID_Partition_Table
	ReservedPartitionGUID = "8DA63339-0007-60C0-C436-083AC8230908"

	// Local registry
	RegistryImage     = "docker.io/library/registry:2"
	RegistryImageName = "registry:2"
	RegistryFilePath  = "registry/registry.tar"

	// CPU architectures
	CPUArchitectureAMD64   = "amd64"
	CPUArchitectureX86     = "x86_64"
	CPUArchitectureARM64   = "arm64"
	CPUArchitectureAARCH64 = "aarch64"

	// AI images
	// TODO: remove when official images are updates
	AssistedInstallerAgentImage = "quay.io/masayag/assisted-installer-agent:billi"
)
