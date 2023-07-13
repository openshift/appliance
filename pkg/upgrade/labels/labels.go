package labels

// This file contains constants for frequently used labels.

// BundleExtracted is indicates that a node has the bundle files extracted into the a directory.
const BundleExtracted = prefix + "/bundle-extracted"

// BundleLoaded is indicates that a node has the images loaded into the CRI-O storage.
const BundleLoaded = prefix + "/bundle-loaded"

// BundleCleaned is indicates that a node has been cleaned after the upgrade.
const BundleCleaned = prefix + "/bundle-cleaned"

// Job contains the name the job.
const Job = prefix + "/job"

// App contains the name of the application.
const App = prefix + "/app"

// prefix is the prefix for all the annotations.
const prefix = "upgrade-tool"
