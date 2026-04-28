package listing

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultContractDir is where Wave stores contract schema files.
const DefaultContractDir = ".agents/contracts"

// ListContracts reads all contract schemas from DefaultContractDir and
// cross-references their use across pipelines on disk.
func ListContracts() ([]ContractInfo, error) {
	entries, err := os.ReadDir(DefaultContractDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Index contract files by filename so pipeline references can attach to them.
	contractsByFile := make(map[string]*ContractInfo)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := entry.Name()
		displayName := strings.TrimSuffix(filename, ".json")
		displayName = strings.TrimSuffix(displayName, ".schema")

		contractsByFile[filename] = &ContractInfo{
			Name:   displayName,
			Type:   "json-schema",
			UsedBy: []ContractUsage{},
		}
	}

	pipelineEntries, err := os.ReadDir(DefaultPipelineDir)
	if err == nil {
		for _, entry := range pipelineEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			pipelineName := strings.TrimSuffix(entry.Name(), ".yaml")
			pipelinePath := filepath.Join(DefaultPipelineDir, entry.Name())

			data, err := os.ReadFile(pipelinePath)
			if err != nil {
				continue
			}

			var p struct {
				Steps []struct {
					ID       string `yaml:"id"`
					Persona  string `yaml:"persona"`
					Contract struct {
						SchemaPath string `yaml:"schema_path"`
					} `yaml:"contract"`
					Handover struct {
						Contract struct {
							SchemaPath string `yaml:"schema_path"`
						} `yaml:"contract"`
					} `yaml:"handover"`
				} `yaml:"steps"`
			}
			if err := yaml.Unmarshal(data, &p); err != nil {
				continue
			}

			for _, step := range p.Steps {
				schemaPath := step.Contract.SchemaPath
				if schemaPath == "" {
					schemaPath = step.Handover.Contract.SchemaPath
				}
				if schemaPath == "" {
					continue
				}

				contractFile := filepath.Base(schemaPath)
				if contract, exists := contractsByFile[contractFile]; exists {
					contract.UsedBy = append(contract.UsedBy, ContractUsage{
						Pipeline: pipelineName,
						Step:     step.ID,
						Persona:  step.Persona,
					})
				} else {
					displayName := strings.TrimSuffix(contractFile, ".json")
					displayName = strings.TrimSuffix(displayName, ".schema")
					contractsByFile[contractFile] = &ContractInfo{
						Name:   displayName,
						Type:   "json-schema (missing)",
						UsedBy: []ContractUsage{{Pipeline: pipelineName, Step: step.ID, Persona: step.Persona}},
					}
				}
			}
		}
	}

	contracts := make([]ContractInfo, 0, len(contractsByFile))
	for _, contract := range contractsByFile {
		contracts = append(contracts, *contract)
	}
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})

	return contracts, nil
}
