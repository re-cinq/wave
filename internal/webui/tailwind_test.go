package webui

import (
	"strings"
	"testing"
)

// TestEmbeddedTailwindCSSPresent verifies that the vendored Tailwind
// stylesheet is embedded in staticFS and contains expected utility output.
// Guards against accidental deletion of internal/webui/static/tailwind.css
// or a regression that would force the WebUI back onto the CDN.
func TestEmbeddedTailwindCSSPresent(t *testing.T) {
	data, err := staticFS.ReadFile("static/tailwind.css")
	if err != nil {
		t.Fatalf("static/tailwind.css missing from embed: %v (run `make tailwind`)", err)
	}
	if len(data) == 0 {
		t.Fatal("static/tailwind.css is empty (run `make tailwind`)")
	}

	css := string(data)

	// `--tw-` custom properties are emitted by every Tailwind v3 build,
	// regardless of which utilities are scanned in.
	if !strings.Contains(css, "--tw-") {
		t.Error("static/tailwind.css missing Tailwind `--tw-` custom properties — file may not be a real Tailwind build")
	}

	// `bg-slate-50` is referenced by templates/work/board.html and
	// templates/work/detail.html, so a successful content scan must emit it.
	if !strings.Contains(css, "bg-slate-50") {
		t.Error("static/tailwind.css missing `bg-slate-50` utility — content scan likely broken")
	}
}

// TestStandaloneTemplatesUseEmbeddedTailwind asserts that the standalone
// pages do not regress to the Tailwind CDN script tag.
func TestStandaloneTemplatesUseEmbeddedTailwind(t *testing.T) {
	for _, path := range standalonePageTemplates {
		data, err := templatesFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		body := string(data)
		if strings.Contains(body, "cdn.tailwindcss.com") {
			t.Errorf("%s still references cdn.tailwindcss.com — must use /static/tailwind.css", path)
		}
		if !strings.Contains(body, "/static/tailwind.css") {
			t.Errorf("%s does not link /static/tailwind.css", path)
		}
	}
}
