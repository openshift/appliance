package data

import "embed"

//go:embed all:services all:scripts all:udev
var EmbeddedAssets embed.FS
