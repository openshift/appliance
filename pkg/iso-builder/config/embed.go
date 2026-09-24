package isobuilder

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

const (
	// ConfigStartMarker is the sentinel that marks the beginning of the embed area.
	ConfigStartMarker = "\x00_ISO_BUILDER_CONFIG_START_\x00"
	// ConfigEndMarker is the sentinel that marks the end of the embed area.
	ConfigEndMarker = "\x00_ISO_BUILDER_CONFIG_END_\x00"
	// ConfigEmbedSize is the size of the payload area between the markers.
	ConfigEmbedSize = 1 << 20 // 1 MiB
)

// Encode serialises a Config to a base64-encoded JSON payload.
func Encode(cfg *Config) ([]byte, error) {
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(jsonData)))
	base64.StdEncoding.Encode(encoded, jsonData)
	return encoded, nil
}

// Decode deserialises a base64-encoded JSON payload into a Config.
func Decode(data []byte) (*Config, error) {
	jsonData := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(jsonData, data)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(jsonData[:n], &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	return &cfg, nil
}

// ReadFromData extracts and decodes the config from raw binary data containing markers.
func ReadFromData(data []byte) (*Config, error) {
	payload, err := extractPayload(data)
	if err != nil {
		return nil, err
	}
	return Decode(payload)
}

// ReadFromBinary reads the embedded config from an iso-builder binary file.
func ReadFromBinary(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading binary: %w", err)
	}
	return ReadFromData(data)
}

// WriteToBinary writes a config into the embed area of an iso-builder binary file.
func WriteToBinary(path string, cfg *Config) error {
	encoded, err := Encode(cfg)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading binary: %w", err)
	}

	areaStart, areaEnd, err := findEmbedArea(data)
	if err != nil {
		return err
	}
	areaSize := areaEnd - areaStart
	if len(encoded) > areaSize {
		return fmt.Errorf("encoded config (%d bytes) exceeds embed area (%d bytes)", len(encoded), areaSize)
	}

	for i := areaStart; i < areaEnd; i++ {
		data[i] = 0
	}
	copy(data[areaStart:], encoded)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stating binary: %w", err)
	}
	return os.WriteFile(path, data, info.Mode())
}

// extractPayload locates the embed area and returns the non-NUL content.
func extractPayload(data []byte) ([]byte, error) {
	start, end, err := findEmbedArea(data)
	if err != nil {
		return nil, err
	}
	payload := bytes.TrimRight(data[start:end], "\x00")
	if len(payload) == 0 {
		return nil, fmt.Errorf("no embedded configuration found")
	}
	return payload, nil
}

// findEmbedArea returns the byte offsets of the payload area (between markers).
func findEmbedArea(data []byte) (start, end int, err error) {
	startIdx := bytes.Index(data, []byte(ConfigStartMarker))
	if startIdx == -1 {
		return 0, 0, fmt.Errorf("start marker not found in binary")
	}
	endIdx := bytes.Index(data, []byte(ConfigEndMarker))
	if endIdx == -1 {
		return 0, 0, fmt.Errorf("end marker not found in binary")
	}
	return startIdx + len(ConfigStartMarker), endIdx, nil
}
