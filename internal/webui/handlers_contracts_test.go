package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContractsPageRendersEmptyHTML verifies that the contracts page renders HTML
// successfully when no contract files exist.
func TestContractsPageRendersEmptyHTML(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleContractsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: expected %q, got %q", "text/html; charset=utf-8", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("expected HTML content in response")
	}
}

// TestContractsPageRendersContractNames verifies that the contracts page renders
// contract names when contract files exist.
func TestContractsPageRendersContractNames(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	schema := `{"title": "Test Contract", "description": "A test schema", "type": "object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "page-render-contract.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write contract: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleContractsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "page-render-contract") {
		t.Errorf("expected body to contain contract name 'page-render-contract', got: %s", body)
	}
}

// TestContractsAPIReturnsEmptyList verifies the API returns an empty contract list
// when no contract files exist.
func TestContractsAPIReturnsEmptyList(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIContracts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	contracts, ok := resp["contracts"]
	if !ok {
		t.Fatal("expected 'contracts' key in response")
	}
	// When nil slice is marshalled, it becomes JSON null.
	if contracts != nil {
		arr, ok := contracts.([]interface{})
		if ok && len(arr) != 0 {
			t.Errorf("expected empty contracts list, got %d items", len(arr))
		}
	}
}

// TestContractsAPIExtractsTitleAndDescription verifies the API returns contract
// summaries with title and description extracted from the JSON schema.
func TestContractsAPIExtractsTitleAndDescription(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	schema := `{"title": "Assessment Schema", "description": "Issue assessment output", "type": "object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "assessment-extract.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write contract: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIContracts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected %q, got %q", "application/json", contentType)
	}

	var resp map[string][]ContractSummary
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	contracts := resp["contracts"]
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}

	c := contracts[0]
	if c.Name != "assessment-extract" {
		t.Errorf("expected name 'assessment-extract', got %q", c.Name)
	}
	if c.Title != "Assessment Schema" {
		t.Errorf("expected title 'Assessment Schema', got %q", c.Title)
	}
	if c.Description != "Issue assessment output" {
		t.Errorf("expected description 'Issue assessment output', got %q", c.Description)
	}
	if c.Filename != "assessment-extract.schema.json" {
		t.Errorf("expected filename 'assessment-extract.schema.json', got %q", c.Filename)
	}
}

// TestContractDetailReturnsSchemaContent verifies that a valid contract name
// returns the full schema content.
func TestContractDetailReturnsSchemaContent(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	schema := `{"title": "Detail Content Test", "type": "object", "properties": {"name": {"type": "string"}}}`
	if err := os.WriteFile(filepath.Join(contractDir, "content-detail.schema.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write contract: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts/content-detail", nil)
	req.SetPathValue("name", "content-detail")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ContractDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "content-detail" {
		t.Errorf("expected name 'content-detail', got %q", resp.Name)
	}
	if resp.Title != "Detail Content Test" {
		t.Errorf("expected title 'Detail Content Test', got %q", resp.Title)
	}
	if resp.Schema == "" {
		t.Error("expected non-empty schema content")
	}
	if !strings.Contains(resp.Schema, `"properties"`) {
		t.Errorf("expected schema to contain 'properties', got: %s", resp.Schema)
	}
}

// TestContractDetailMissingNameReturns400 verifies that an empty name returns 400.
func TestContractDetailMissingNameReturns400(t *testing.T) {
	srv, _ := testServer(t)

	req := httptest.NewRequest("GET", "/api/contracts/", nil)
	// Do not set path value to simulate missing name.
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", rec.Code)
	}
}

// TestContractDetailNotFoundReturns404 verifies that a non-existent contract returns 404.
func TestContractDetailNotFoundReturns404(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts/nonexistent-xyz", nil)
	req.SetPathValue("name", "nonexistent-xyz")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent contract, got %d", rec.Code)
	}
}

// TestContractDetailPathTraversalRejects verifies that names containing
// path traversal characters are rejected with 400.
func TestContractDetailPathTraversalRejects(t *testing.T) {
	srv, _ := testServer(t)

	tests := []struct {
		name  string
		input string
	}{
		{"slash_traversal", "../../etc/passwd"},
		{"dotdot_only", ".."},
		{"embedded_slash", "foo/bar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/contracts/"+tc.input, nil)
			req.SetPathValue("name", tc.input)
			rec := httptest.NewRecorder()
			srv.handleAPIContractDetail(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for path traversal input %q, got %d", tc.input, rec.Code)
			}
		})
	}
}

// TestContractsAPIIgnoresNonJsonFiles verifies that non-JSON files in
// the contracts directory are not included in the response.
func TestContractsAPIIgnoresNonJsonFiles(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	// Write a non-JSON file that should be ignored.
	if err := os.WriteFile(filepath.Join(contractDir, "readme.txt"), []byte("not a contract"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	// Write a valid JSON contract.
	schema := `{"title": "Valid Only", "type": "object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "valid-only.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write contract: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts", nil)
	rec := httptest.NewRecorder()
	srv.handleAPIContracts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string][]ContractSummary
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	contracts := resp["contracts"]
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract (non-JSON ignored), got %d", len(contracts))
	}
	if contracts[0].Name != "valid-only" {
		t.Errorf("expected contract name 'valid-only', got %q", contracts[0].Name)
	}
}

// TestContractDetailLooksUpPlainJsonSuffix verifies that the detail endpoint
// can look up contracts with a plain .json suffix (not .schema.json).
func TestContractDetailLooksUpPlainJsonSuffix(t *testing.T) {
	srv, _ := testServer(t)

	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, ".agents", "contracts")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	schema := `{"title": "Plain JSON", "type": "object"}`
	if err := os.WriteFile(filepath.Join(contractDir, "plain-lookup.json"), []byte(schema), 0o644); err != nil {
		t.Fatalf("failed to write contract: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore dir: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/contracts/plain-lookup", nil)
	req.SetPathValue("name", "plain-lookup")
	rec := httptest.NewRecorder()
	srv.handleAPIContractDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ContractDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "plain-lookup" {
		t.Errorf("expected name 'plain-lookup', got %q", resp.Name)
	}
	if resp.Title != "Plain JSON" {
		t.Errorf("expected title 'Plain JSON', got %q", resp.Title)
	}
}
