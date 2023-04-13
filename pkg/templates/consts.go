package templates

const (
	// user.cfg template
	UserCfgTemplateFile = "guestfish/user.cfg.template"
	GrubTimeout         = 10
	GrubDefault         = 0
	GrubMenuEntryName   = "Agent-Based Installer"
	LiveISO             = "rhcos-412.86.202301311551-0"

	// guestfish.sh template
	GuestfishScriptTemplateFile = "guestfish/guestfish.sh.template"
	ApplianceFileName           = "appliance.raw"
	RecoveryIsoFileName         = "recovery.iso"

	// Recovery partition
	RecoveryPartitionName = "agentrecovery"

	// ReservedPartitionGUID Set partition as Linux reserved partition: https://en.wikipedia.org/wiki/GUID_Partition_Table
	ReservedPartitionGUID = "8DA63339-0007-60C0-C436-083AC8230908"
)
