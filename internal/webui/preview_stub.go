//go:build webui_preview

package webui

// registerPreview is a no-op stub when the webui_preview build tag is active.
// Real implementations are added in Phase B/C.
func registerPreview(r *FeatureRegistry) {}