package listing

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadManifest reads and parses a wave manifest at the given path. A missing
// file is reported via the returned error so callers can decide whether to
// degrade gracefully.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
