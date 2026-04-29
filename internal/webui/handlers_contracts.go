package webui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContractSummary holds summary information about a contract schema file.
type ContractSummary struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Filename    string `json:"filename"`
}

// ContractDetailResponse holds the full contract schema content.
type ContractDetailResponse struct {
	ContractSummary
	Schema string `json:"schema"`
}

// contractsDir returns the path to the .agents/contracts directory.
func contractsDir() string {
	return filepath.Join(".agents", "contracts")
}

// listContractSummaries reads all JSON schema files from .agents/contracts/ and
// returns a sorted slice of ContractSummary values.
func listContractSummaries() []ContractSummary {
	dir := contractsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var summaries []ContractSummary
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		sum := ContractSummary{
			Filename: name,
			Name:     strings.TrimSuffix(name, ".schema.json"),
		}
		if !strings.HasSuffix(name, ".schema.json") {
			sum.Name = strings.TrimSuffix(name, ".json")
		}
		// Attempt to extract title/description from the schema JSON.
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr == nil {
			var meta struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			}
			if jsonErr := json.Unmarshal(data, &meta); jsonErr == nil {
				sum.Title = meta.Title
				sum.Description = meta.Description
			}
		}
		summaries = append(summaries, sum)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}

// handleContractsPage handles GET /contracts — serves the HTML contracts page.
func (s *Server) handleContractsPage(w http.ResponseWriter, r *http.Request) {
	contracts := listContractSummaries()

	data := struct {
		ActivePage string
		Contracts  []ContractSummary
	}{
		ActivePage: "contracts",
		Contracts:  contracts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/contracts.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIContracts handles GET /api/contracts — returns contract list as JSON.
func (s *Server) handleAPIContracts(w http.ResponseWriter, r *http.Request) {
	contracts := listContractSummaries()
	writeJSON(w, http.StatusOK, map[string]interface{}{"contracts": contracts})
}

// findContract looks up a contract by name, returning the detail response.
// Returns nil if not found.
func findContract(name string) *ContractDetailResponse {
	// Prevent path traversal.
	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, string(os.PathSeparator)) {
		return nil
	}

	dir := contractsDir()
	var filePath string
	for _, candidate := range []string{
		filepath.Join(dir, name+".schema.json"),
		filepath.Join(dir, name+".json"),
		filepath.Join(dir, name),
	} {
		if _, statErr := os.Stat(candidate); statErr == nil {
			filePath = candidate
			break
		}
	}

	if filePath == "" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	filename := filepath.Base(filePath)
	contractName := strings.TrimSuffix(filename, ".schema.json")
	if !strings.HasSuffix(filename, ".schema.json") {
		contractName = strings.TrimSuffix(filename, ".json")
	}

	var meta struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(data, &meta)

	return &ContractDetailResponse{
		ContractSummary: ContractSummary{
			Name:        contractName,
			Title:       meta.Title,
			Description: meta.Description,
			Filename:    filename,
		},
		Schema: string(data),
	}
}

// handleAPIContractDetail handles GET /api/contracts/{name} — returns contract
// schema content as JSON.
func (s *Server) handleAPIContractDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing contract name")
		return
	}

	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, string(os.PathSeparator)) {
		writeJSONError(w, http.StatusBadRequest, "invalid contract name")
		return
	}

	resp := findContract(name)
	if resp == nil {
		writeJSONError(w, http.StatusNotFound, "contract not found")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleContractDetailPage handles GET /contracts/{name} — serves the HTML contract detail page.
func (s *Server) handleContractDetailPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing contract name", http.StatusBadRequest)
		return
	}

	contract := findContract(name)
	if contract == nil {
		http.Error(w, "contract not found", http.StatusNotFound)
		return
	}

	// Find pipeline usage
	var usedBy []ContractUsageRef
	pipelineNames := listPipelineNames()
	for _, pName := range pipelineNames {
		pl, err := loadPipelineYAML(pName)
		if err != nil {
			continue
		}
		for _, step := range pl.Steps {
			if step.Handover.Contract.SchemaPath == "" {
				continue
			}
			// Extract schema base name from path like ".agents/contracts/foo.schema.json"
			schemaBase := filepath.Base(step.Handover.Contract.SchemaPath)
			schemaName := strings.TrimSuffix(schemaBase, ".schema.json")
			if !strings.HasSuffix(schemaBase, ".schema.json") {
				schemaName = strings.TrimSuffix(schemaBase, ".json")
			}
			if schemaName == name {
				usedBy = append(usedBy, ContractUsageRef{
					Pipeline:     pName,
					StepID:       step.ID,
					ContractType: step.Handover.Contract.Type,
				})
			}
		}
	}

	data := ContractDetailPage{
		ActivePage: "contracts",
		Contract:   *contract,
		UsedBy:     usedBy,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/contract_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
