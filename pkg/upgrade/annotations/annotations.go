package annotations

// This file contains constants for frequently used annotations.

// BundleFile is the annotation that contains the name of the bundle file.
const BundleFile = prefix + "/bundle-file"

// FundleMetadata contains the metadata of the bundle, except the list of images.
const BundleMetadata = prefix + "/bundle-metadata"

// Progress contains information about the progress of the upgrade.
const Progress = prefix + "/progress"

// prefix is the prefix for all the annotations.
const prefix = "upgrade-tool"
