// Package defaults provides embedded default personas, pipelines, and contracts
// that are included in the Wave binary for use with `wave init`.
package defaults

import (
	"embed"
	"io/fs"
	"path/filepath"
)

//go:embed personas/*.md
var personasFS embed.FS

//go:embed pipelines/*.yaml
var pipelinesFS embed.FS

//go:embed contracts/*.json
var contractsFS embed.FS

// GetPersonas returns a map of filename to content for all default personas.
func GetPersonas() (map[string]string, error) {
	return readDir(personasFS, "personas")
}

// GetPipelines returns a map of filename to content for all default pipelines.
func GetPipelines() (map[string]string, error) {
	return readDir(pipelinesFS, "pipelines")
}

// GetContracts returns a map of filename to content for all default contracts.
func GetContracts() (map[string]string, error) {
	return readDir(contractsFS, "contracts")
}

func readDir(fsys embed.FS, dir string) (map[string]string, error) {
	result := make(map[string]string)

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		filename := filepath.Base(path)
		result[filename] = string(content)
		return nil
	})

	return result, err
}

// PersonaNames returns a list of all persona filenames.
func PersonaNames() []string {
	personas, _ := GetPersonas()
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	return names
}

// PipelineNames returns a list of all pipeline filenames.
func PipelineNames() []string {
	pipelines, _ := GetPipelines()
	names := make([]string, 0, len(pipelines))
	for name := range pipelines {
		names = append(names, name)
	}
	return names
}

// ContractNames returns a list of all contract filenames.
func ContractNames() []string {
	contracts, _ := GetContracts()
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	return names
}
