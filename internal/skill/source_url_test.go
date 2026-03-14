package skill

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createTestTarGz(t *testing.T, skillName, description string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add skill directory
	if err := tw.WriteHeader(&tar.Header{
		Name:     skillName + "/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}); err != nil {
		t.Fatal(err)
	}

	// Add SKILL.md
	content := []byte("---\nname: " + skillName + "\ndescription: " + description + "\n---\n# " + skillName + "\n")
	if err := tw.WriteHeader(&tar.Header{
		Name:     skillName + "/SKILL.md",
		Size:     int64(len(content)),
		Mode:     0644,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func createTestZip(t *testing.T, skillName, description string) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	content := "---\nname: " + skillName + "\ndescription: " + description + "\n---\n# " + skillName + "\n"
	f, err := w.Create(skillName + "/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestURLAdapterPrefix(t *testing.T) {
	a := NewURLAdapter()
	if a.Prefix() != "https://" {
		t.Errorf("Prefix() = %q, want %q", a.Prefix(), "https://")
	}
}

func TestURLAdapterTarGz(t *testing.T) {
	archive := createTestTarGz(t, "tar-skill", "Skill from tar.gz")

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		if _, err := w.Write(archive); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	result, err := a.Install(context.Background(), server.URL+"/skill.tar.gz", store)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "tar-skill" {
		t.Errorf("Name = %q, want %q", result.Skills[0].Name, "tar-skill")
	}
	if store.writes != 1 {
		t.Errorf("expected 1 write, got %d", store.writes)
	}
}

func TestURLAdapterZip(t *testing.T) {
	archive := createTestZip(t, "zip-skill", "Skill from zip")

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		if _, err := w.Write(archive); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	result, err := a.Install(context.Background(), server.URL+"/skill.zip", store)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "zip-skill" {
		t.Errorf("Name = %q, want %q", result.Skills[0].Name, "zip-skill")
	}
}

func TestURLAdapterNonArchive(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte("<html>Not an archive</html>")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	_, err := a.Install(context.Background(), server.URL+"/page.html", store)
	if err == nil {
		t.Fatal("expected error for non-archive URL")
	}
	if !strings.Contains(err.Error(), "unrecognized archive format") {
		t.Errorf("error should mention unrecognized format: %v", err)
	}
}

func TestURLAdapterHTTPError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	_, err := a.Install(context.Background(), server.URL+"/skill.tar.gz", store)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain 404: %v", err)
	}
}

func TestURLAdapterUnreachable(t *testing.T) {
	// Use a closed server to get connection refused
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := server.Client()
	serverURL := server.URL
	server.Close()

	a := &URLAdapter{client: client}
	store := newMemoryStore()

	_, err := a.Install(context.Background(), serverURL+"/skill.tar.gz", store)
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestURLAdapterZipSlip(t *testing.T) {
	// Create a zip with path traversal
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, err := w.Create("../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("malicious content")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(buf.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	_, err = a.Install(context.Background(), server.URL+"/skill.zip", store)
	if err == nil {
		t.Fatal("expected error for zip-slip attack")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("error should mention path traversal: %v", err)
	}
}

func TestURLAdapterTgzExtension(t *testing.T) {
	archive := createTestTarGz(t, "tgz-skill", "Skill from tgz")

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(archive); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	result, err := a.Install(context.Background(), server.URL+"/skill.tgz", store)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("malicious")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "../../../etc/passwd",
		Size:     int64(len(content)),
		Mode:     0644,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	err := extractTarGz(bytes.NewReader(buf.Bytes()), destDir)
	if err == nil {
		t.Fatal("expected error for path traversal in tar.gz")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("error should mention path traversal: %v", err)
	}
}

func TestURLAdapterHTTPServer500(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	a := &URLAdapter{client: server.Client()}
	store := newMemoryStore()

	_, err := a.Install(context.Background(), server.URL+"/skill.tar.gz", store)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain 500: %v", err)
	}
}
