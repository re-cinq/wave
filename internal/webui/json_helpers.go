package webui

import (
	"encoding/json"
	"net/http"
	"os"
)

// writeJSON encodes data as JSON with the given status code. Used by every
// handler for the success path.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeJSONError writes a single-field error JSON body with the given status.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// listPipelineNames returns pipeline names by scanning .agents/pipelines/.
// Used by views that need a static list of pipelines (filters, dropdowns).
func listPipelineNames() []string {
	entries, err := os.ReadDir(".agents/pipelines")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 5 && name[len(name)-5:] == ".yaml" {
			names = append(names, name[:len(name)-5])
		}
	}
	return names
}
