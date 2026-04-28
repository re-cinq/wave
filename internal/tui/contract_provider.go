package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/pipeline"
)

// ContractInfo is the TUI data projection for a contract.
type ContractInfo struct {
	Label         string
	Type          string
	SchemaPath    string
	Source        string
	SchemaPreview string
	PipelineUsage []PipelineStepRef
}

// ContractDataProvider fetches contract data for the Contracts view.
type ContractDataProvider interface {
	FetchContracts() ([]ContractInfo, error)
}

// DefaultContractDataProvider implements ContractDataProvider by scanning pipeline YAML files.
type DefaultContractDataProvider struct {
	pipelinesDir string
}

// NewDefaultContractDataProvider creates a new contract data provider.
func NewDefaultContractDataProvider(pipelinesDir string) *DefaultContractDataProvider {
	return &DefaultContractDataProvider{pipelinesDir: pipelinesDir}
}

// FetchContracts scans all pipeline step definitions and returns distinct contracts.
func (p *DefaultContractDataProvider) FetchContracts() ([]ContractInfo, error) {
	if p.pipelinesDir == "" {
		return nil, nil
	}

	// Map by schema path for deduplication; inline contracts use "pipeline:step" key
	contractMap := make(map[string]*ContractInfo)

	for _, pl := range pipeline.ScanPipelinesDir(p.pipelinesDir) {
		for _, step := range pl.Steps {
			contract := step.Handover.Contract
			if contract.Type == "" {
				continue
			}

			ref := PipelineStepRef{
				PipelineName: pl.Metadata.Name,
				StepID:       step.ID,
			}

			if contract.SchemaPath != "" {
				// File-backed contract — deduplicate by schema path
				key := contract.SchemaPath
				if existing, ok := contractMap[key]; ok {
					existing.PipelineUsage = append(existing.PipelineUsage, ref)
				} else {
					label := filepath.Base(contract.SchemaPath)
					preview := loadSchemaPreview(contract.SchemaPath)
					contractMap[key] = &ContractInfo{
						Label:         label,
						Type:          contract.Type,
						SchemaPath:    contract.SchemaPath,
						SchemaPreview: preview,
						PipelineUsage: []PipelineStepRef{ref},
					}
				}
			} else {
				// Inline contract — unique per pipeline:step
				key := pl.Metadata.Name + ":" + step.ID
				source := contract.Source
				if source == "" {
					source = contract.Command
				}
				contractMap[key] = &ContractInfo{
					Label:         key,
					Type:          contract.Type,
					Source:        source,
					SchemaPreview: source,
					PipelineUsage: []PipelineStepRef{ref},
				}
			}
		}
	}

	// Convert map to sorted slice
	var contracts []ContractInfo
	for _, c := range contractMap {
		contracts = append(contracts, *c)
	}

	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Label < contracts[j].Label
	})

	return contracts, nil
}

// loadSchemaPreview reads the first ~30 lines of a schema file.
func loadSchemaPreview(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > 30 {
		lines = lines[:30]
		lines = append(lines, "... (truncated)")
	}

	return strings.Join(lines, "\n")
}
