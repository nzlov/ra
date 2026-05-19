package pluginbundle

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	manifestSection     = "ra.manifest"
	capabilitiesSection = "ra.capabilities"
	assetsSection       = "ra.assets"
)

type Manifest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Permissions  []string `json:"permissions,omitempty"`
	MinRAVersion string   `json:"minRaVersion,omitempty"`
}

type Capability struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Icon     string   `json:"icon,omitempty"`
	UI       string   `json:"ui"`
	Keywords []string `json:"keywords,omitempty"`
}

type Bundle struct {
	Manifest     Manifest
	Capabilities []Capability
	Assets       map[string][]byte
	RawManifest  []byte
}

var validID = regexp.MustCompile(`^[a-z0-9][a-z0-9-_.]*$`)

func Build(manifest Manifest, capabilities []Capability, assets map[string][]byte) ([]byte, error) {
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	if err := validateCapabilities(capabilities); err != nil {
		return nil, err
	}
	assetEntries, err := encodeAssets(assets)
	if err != nil {
		return nil, err
	}
	if err := validateCapabilityAssets(capabilities, assets); err != nil {
		return nil, err
	}
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	rawCapabilities, err := json.Marshal(capabilities)
	if err != nil {
		return nil, err
	}
	rawAssets, err := json.Marshal(assetEntries)
	if err != nil {
		return nil, err
	}
	return BuildRaw(map[string][]byte{
		manifestSection:     rawManifest,
		capabilitiesSection: rawCapabilities,
		assetsSection:       rawAssets,
	}, nil)
}

func BuildRaw(customSections map[string][]byte, extraSections [][]byte) ([]byte, error) {
	var out bytes.Buffer
	out.Write([]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00})
	names := make([]string, 0, len(customSections))
	for name := range customSections {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		out.Write(encodeCustomSection(name, customSections[name]))
	}
	for _, section := range extraSections {
		out.Write(section)
	}
	return out.Bytes(), nil
}

func Read(raw []byte) (Bundle, error) {
	sections, err := readCustomSections(raw)
	if err != nil {
		return Bundle{}, err
	}
	rawManifest, ok := sections[manifestSection]
	if !ok {
		return Bundle{}, errors.New("missing ra.manifest section")
	}
	var manifest Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return Bundle{}, fmt.Errorf("read manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return Bundle{}, err
	}

	var capabilities []Capability
	if rawCapabilities, ok := sections[capabilitiesSection]; ok && len(rawCapabilities) > 0 {
		if err := json.Unmarshal(rawCapabilities, &capabilities); err != nil {
			return Bundle{}, fmt.Errorf("read capabilities: %w", err)
		}
	}
	if err := validateCapabilities(capabilities); err != nil {
		return Bundle{}, err
	}

	assets := map[string][]byte{}
	if rawAssets, ok := sections[assetsSection]; ok && len(rawAssets) > 0 {
		var entries map[string]string
		if err := json.Unmarshal(rawAssets, &entries); err != nil {
			return Bundle{}, fmt.Errorf("read assets: %w", err)
		}
		for assetPath, encoded := range entries {
			if err := validateAssetPath(assetPath); err != nil {
				return Bundle{}, err
			}
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return Bundle{}, fmt.Errorf("read asset %q: %w", assetPath, err)
			}
			assets[assetPath] = data
		}
	}
	if err := validateCapabilityAssets(capabilities, assets); err != nil {
		return Bundle{}, err
	}

	return Bundle{
		Manifest:     manifest,
		Capabilities: capabilities,
		Assets:       assets,
		RawManifest:  append([]byte(nil), rawManifest...),
	}, nil
}

func encodeCustomSection(name string, payload []byte) []byte {
	var body bytes.Buffer
	writeU32(&body, uint32(len(name)))
	body.WriteString(name)
	body.Write(payload)

	var out bytes.Buffer
	out.WriteByte(0)
	writeU32(&out, uint32(body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func readCustomSections(raw []byte) (map[string][]byte, error) {
	if len(raw) < 8 || !bytes.Equal(raw[:4], []byte{0x00, 0x61, 0x73, 0x6d}) || !bytes.Equal(raw[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
		return nil, errors.New("invalid wasm header")
	}
	sections := map[string][]byte{}
	offset := 8
	for offset < len(raw) {
		sectionID := raw[offset]
		offset++
		size, next, err := readU32(raw, offset)
		if err != nil {
			return nil, err
		}
		offset = next
		end := offset + int(size)
		if end > len(raw) {
			return nil, errors.New("truncated wasm section")
		}
		if sectionID == 0 {
			nameLen, nameOffset, err := readU32(raw, offset)
			if err != nil {
				return nil, err
			}
			nameEnd := nameOffset + int(nameLen)
			if nameEnd > end {
				return nil, errors.New("truncated wasm custom section name")
			}
			name := string(raw[nameOffset:nameEnd])
			if _, exists := sections[name]; exists {
				return nil, fmt.Errorf("duplicate wasm custom section %q", name)
			}
			sections[name] = append([]byte(nil), raw[nameEnd:end]...)
		}
		offset = end
	}
	return sections, nil
}

func writeU32(buf *bytes.Buffer, value uint32) {
	var tmp [5]byte
	n := binary.PutUvarint(tmp[:], uint64(value))
	buf.Write(tmp[:n])
}

func readU32(raw []byte, offset int) (uint32, int, error) {
	value, n := binary.Uvarint(raw[offset:])
	if n <= 0 {
		return 0, offset, errors.New("invalid wasm varint")
	}
	return uint32(value), offset + n, nil
}

func validateManifest(manifest Manifest) error {
	if !validID.MatchString(manifest.ID) {
		return fmt.Errorf("invalid plugin id %q", manifest.ID)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("plugin %q has empty name", manifest.ID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("plugin %q has empty version", manifest.ID)
	}
	return nil
}

func validateCapabilities(capabilities []Capability) error {
	seen := map[string]struct{}{}
	seenUIDirs := map[string]string{}
	for _, capability := range capabilities {
		if !validID.MatchString(capability.ID) {
			return fmt.Errorf("invalid capability id %q", capability.ID)
		}
		if _, ok := seen[capability.ID]; ok {
			return fmt.Errorf("duplicate capability id %q", capability.ID)
		}
		seen[capability.ID] = struct{}{}
		if strings.TrimSpace(capability.Title) == "" {
			return fmt.Errorf("capability %q has empty title", capability.ID)
		}
		if err := validateAssetPath(capability.UI); err != nil {
			return fmt.Errorf("capability %q has invalid ui: %w", capability.ID, err)
		}
		if path.Ext(capability.UI) != ".html" {
			return fmt.Errorf("capability %q ui asset must be .html", capability.ID)
		}
		if path.Dir(capability.UI) == "/" {
			return fmt.Errorf("capability %q ui asset must live under a capability directory", capability.ID)
		}
		uiDir := path.Dir(capability.UI)
		if other, ok := seenUIDirs[uiDir]; ok {
			return fmt.Errorf("duplicate capability ui directory %q for %q and %q", uiDir, other, capability.ID)
		}
		seenUIDirs[uiDir] = capability.ID
		if capability.Icon != "" {
			if err := validateAssetPath(capability.Icon); err != nil {
				return fmt.Errorf("capability %q has invalid icon: %w", capability.ID, err)
			}
			if path.Ext(capability.Icon) == ".html" {
				return fmt.Errorf("capability %q icon asset must not be .html", capability.ID)
			}
		}
	}
	return nil
}

func encodeAssets(assets map[string][]byte) (map[string]string, error) {
	paths := make([]string, 0, len(assets))
	for assetPath := range assets {
		paths = append(paths, assetPath)
	}
	sort.Strings(paths)
	entries := make(map[string]string, len(paths))
	for _, assetPath := range paths {
		if err := validateAssetPath(assetPath); err != nil {
			return nil, err
		}
		entries[assetPath] = base64.StdEncoding.EncodeToString(assets[assetPath])
	}
	return entries, nil
}

func validateCapabilityAssets(capabilities []Capability, assets map[string][]byte) error {
	for _, capability := range capabilities {
		if _, ok := assets[capability.UI]; !ok {
			return fmt.Errorf("capability %q missing ui asset %q", capability.ID, capability.UI)
		}
		if capability.Icon != "" {
			if _, ok := assets[capability.Icon]; !ok {
				return fmt.Errorf("capability %q missing icon asset %q", capability.ID, capability.Icon)
			}
		}
	}
	return nil
}

func validateAssetPath(assetPath string) error {
	if !strings.HasPrefix(assetPath, "/") {
		return fmt.Errorf("asset path %q must start with /", assetPath)
	}
	clean := path.Clean(assetPath)
	if clean != assetPath || clean == "/" || strings.Contains(clean, "..") {
		return fmt.Errorf("invalid asset path %q", assetPath)
	}
	return nil
}
