//go:build webui_preview

// Package webui — /preview/* route group (build-tag-gated).
//
// Compiled only when -tags=webui_preview is set. Default builds ship zero
// preview footprint: templates, css, fixtures, handlers, and the route
// registrar are all behind this tag.

package webui

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/preview/*.html
var previewTemplatesFS embed.FS

//go:embed static/preview/style.css
var previewStaticFS embed.FS

// previewTemplates holds parsed standalone HTML templates keyed by page name
// (index, onboard, work, work_item, proposal). Each template includes the
// shared "banner" partial parsed from _banner.html.
var (
	previewTemplates map[string]*template.Template
	previewLoadErr   error
)

func init() {
	previewTemplates, previewLoadErr = parsePreviewTemplates()
}

// previewPages enumerates the (page-name, template-file) pairs registered
// under /preview/*. Listed once so registration, parsing, and tests share a
// single source of truth.
var previewPages = []struct {
	name string
	file string
}{
	{"index", "index.html"},
	{"onboard", "onboard.html"},
	{"work", "work.html"},
	{"work_item", "work_item.html"},
	{"proposal", "proposal.html"},
}

func parsePreviewTemplates() (map[string]*template.Template, error) {
	bannerData, err := previewTemplatesFS.ReadFile("templates/preview/_banner.html")
	if err != nil {
		return nil, fmt.Errorf("preview: read banner: %w", err)
	}
	out := make(map[string]*template.Template, len(previewPages))
	for _, p := range previewPages {
		t := template.New(p.name)
		if _, err := t.Parse(string(bannerData)); err != nil {
			return nil, fmt.Errorf("preview: parse banner for %s: %w", p.name, err)
		}
		pageData, err := previewTemplatesFS.ReadFile("templates/preview/" + p.file)
		if err != nil {
			return nil, fmt.Errorf("preview: read %s: %w", p.file, err)
		}
		if _, err := t.Parse(string(pageData)); err != nil {
			return nil, fmt.Errorf("preview: parse %s: %w", p.file, err)
		}
		out[p.name] = t
	}
	return out, nil
}

func registerPreview(r *FeatureRegistry) {
	r.addRoutes(func(_ *Server, mux *http.ServeMux) {
		mux.HandleFunc("GET /preview/{$}", handlePreviewIndex)
		mux.HandleFunc("GET /preview/onboard", handlePreviewOnboard)
		mux.HandleFunc("GET /preview/work", handlePreviewWork)
		mux.HandleFunc("GET /preview/work-item", handlePreviewWorkItem)
		mux.HandleFunc("GET /preview/proposal", handlePreviewProposal)
		mux.HandleFunc("GET /preview/static/style.css", handlePreviewCSS)
	})
}

func renderPreview(w http.ResponseWriter, name string, data any) {
	if previewLoadErr != nil {
		http.Error(w, "preview templates failed to load: "+previewLoadErr.Error(), http.StatusInternalServerError)
		return
	}
	t, ok := previewTemplates[name]
	if !ok {
		http.Error(w, "preview template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "preview render error: "+err.Error(), http.StatusInternalServerError)
	}
}

func handlePreviewIndex(w http.ResponseWriter, _ *http.Request) {
	renderPreview(w, "index", landingFixture)
}

func handlePreviewOnboard(w http.ResponseWriter, _ *http.Request) {
	renderPreview(w, "onboard", onboardFixture)
}

func handlePreviewWork(w http.ResponseWriter, _ *http.Request) {
	renderPreview(w, "work", workFixture)
}

func handlePreviewWorkItem(w http.ResponseWriter, _ *http.Request) {
	renderPreview(w, "work_item", workItemFixture)
}

func handlePreviewProposal(w http.ResponseWriter, _ *http.Request) {
	renderPreview(w, "proposal", proposalFixture)
}

func handlePreviewCSS(w http.ResponseWriter, _ *http.Request) {
	data, err := previewStaticFS.ReadFile("static/preview/style.css")
	if err != nil {
		http.Error(w, "preview css missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(data)
}
