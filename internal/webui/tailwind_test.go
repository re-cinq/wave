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

	// `bg-surface` is a custom color token defined in tailwind.config.js and
	// referenced by consolidated templates. Its presence confirms the config
	// customizations are picked up by the content scan.
	if !strings.Contains(css, "bg-surface") {
		t.Error("static/tailwind.css missing `bg-surface` custom color utility — content scan likely broken")
	}
}

// TestStandaloneTemplatesUseDesignSystem asserts that standalone pages link
// the production stylesheet and do not regress to the Tailwind CDN.
func TestStandaloneTemplatesUseDesignSystem(t *testing.T) {
	for _, path := range standalonePageTemplates {
		data, err := templatesFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		body := string(data)
		if strings.Contains(body, "cdn.tailwindcss.com") {
			t.Errorf("%s still references cdn.tailwindcss.com — use /static/style.css design system", path)
		}
		if !strings.Contains(body, "/static/style.css") {
			t.Errorf("%s does not link /static/style.css — standalone pages must use the design system", path)
		}
	}
}
