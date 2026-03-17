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

// contractsDir returns the path to the .wave/contracts directory.
func contractsDir() string {
	return filepath.Join(".wave", "contracts")
}

// listContractSummaries reads all JSON schema files from .wave/contracts/ and
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
		Contracts  []ContractSummary
		ActivePage string
	}{
		Contracts:  contracts,
		ActivePage: "contracts",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/contracts.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIContracts handles GET /api/contracts — returns contract list as JSON.
func (s *Server) handleAPIContracts(w http.ResponseWriter, r *http.Request) {
	contracts := listContractSummaries()
	writeJSON(w, http.StatusOK, map[string]interface{}{"contracts": contracts})
}

// handleAPIContractDetail handles GET /api/contracts/{name} — returns contract
// schema content as JSON.
func (s *Server) handleAPIContractDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing contract name")
		return
	}

	// Prevent path traversal.
	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, string(os.PathSeparator)) {
		writeJSONError(w, http.StatusBadRequest, "invalid contract name")
		return
	}

	dir := contractsDir()
	// Try exact filename first, then with .schema.json suffix.
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
		writeJSONError(w, http.StatusNotFound, "contract not found")
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to read contract")
		return
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

	resp := ContractDetailResponse{
		ContractSummary: ContractSummary{
			Name:        contractName,
			Title:       meta.Title,
			Description: meta.Description,
			Filename:    filename,
		},
		Schema: string(data),
	}

	writeJSON(w, http.StatusOK, resp)
}
