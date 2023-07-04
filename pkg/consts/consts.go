package consts

const (
	// user.cfg template
	UserCfgTemplateFile = "scripts/grub/user.cfg.template"
	GrubTimeout         = 10
	GrubDefault         = 0
	GrubMenuEntryName   = "Agent-Based Installer"
	// For installation ignition
	GrubMenuEntryNameRecovery = "Recovery: Agent-Based Installer"
	GrubDefaultRecovery       = 1
	UserCfgFilePath           = "/boot/grub2/user.cfg"

	// guestfish.sh template
	GuestfishScriptTemplateFile = "scripts/guestfish/guestfish.sh.template"
	ApplianceFileName           = "appliance.raw"
	RecoveryIsoFileName         = "recovery.iso"
	DataIsoFileName             = "data.iso"

	// ImageSetTemplateFile imageset.yaml.template
	ImageSetTemplateFile = "scripts/mirror/imageset.yaml.template"

	// Recovery/Data partitions
	RecoveryPartitionName = "agentboot"
	DataPartitionName     = "agentdata"

	// ReservedPartitionGUID Set partition as Linux reserved partition: https://en.wikipedia.org/wiki/GUID_Partition_Table
	ReservedPartitionGUID = "8DA63339-0007-60C0-C436-083AC8230908"

	// Local registry
	RegistryImage     = "docker.io/library/registry:2"
	RegistryImageName = "registry:2"
	RegistryFilePath  = "registry/registry.tar"
	RegistryPort      = 5005

	// AI images
	// TODO: remove when official images are updates
	AssistedInstallerAgentImage = "quay.io/edge-infrastructure/assisted-installer-agent:latest"
)
