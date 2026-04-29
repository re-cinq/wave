//go:build webui_preview

package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newPreviewMux constructs an http.ServeMux with /preview/* routes
// registered via the same FeatureRegistry seam used in production. No
// *Server is constructed — preview handlers are stateless.
func newPreviewMux(t *testing.T) *http.ServeMux {
	t.Helper()
	r := &FeatureRegistry{}
	registerPreview(r)
	mux := http.NewServeMux()
	for _, fn := range r.routeFns {
		fn(nil, mux)
	}
	return mux
}

func TestPreviewRoutesRespond(t *testing.T) {
	mux := newPreviewMux(t)

	cases := []struct {
		name string
		path string
	}{
		{"index", "/preview/"},
		{"onboard", "/preview/onboard"},
		{"work", "/preview/work"},
		{"work-item", "/preview/work-item"},
		{"proposal", "/preview/proposal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
				t.Errorf("Content-Type = %q, want text/html...", ct)
			}
			body := w.Body.String()
			if !strings.Contains(body, "PREVIEW") {
				t.Errorf("body missing PREVIEW banner string")
			}
			if !strings.Contains(body, "/preview/static/style.css") {
				t.Errorf("body missing preview stylesheet link")
			}
		})
	}
}

func TestPreviewCSSRoute(t *testing.T) {
	mux := newPreviewMux(t)
	req := httptest.NewRequest(http.MethodGet, "/preview/static/style.css", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css...", ct)
	}
	if w.Body.Len() == 0 {
		t.Errorf("css body empty")
	}
}

func TestPreviewIndexUnknownTemplate(t *testing.T) {
	// Guards against regression where renderPreview dispatches by name —
	// an unknown name must surface a clean 500, not panic.
	w := httptest.NewRecorder()
	renderPreview(w, "does-not-exist", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
