// gen_embed_area generates config_embed_area.bin, a binary blob that
// go embed bakes into the iso-builder executable. The blob contains a
// start marker, 1 MiB of NUL-padded payload space, and an end marker.
// External tools locate and overwrite the payload area to inject a
// base64-encoded config after compilation. Marker constants are
// imported from the config module to avoid duplication.
package main

import (
	"os"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder/config"
)

func main() {
	f, err := os.Create("config_embed_area.bin")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, err := f.WriteString(isobuilder.ConfigStartMarker); err != nil {
		panic(err)
	}
	if _, err := f.Write(make([]byte, isobuilder.ConfigEmbedSize)); err != nil {
		panic(err)
	}
	if _, err := f.WriteString(isobuilder.ConfigEndMarker); err != nil {
		panic(err)
	}
}
