package continuous

import (
	"net/url"
	"strings"

	"github.com/recinq/wave/internal/state"
)

// BuildItemKey normalizes an issue URL into a stable key for deduplication.
// For GitHub URLs like "https://github.com/owner/repo/issues/42",
// it returns "owner/repo#42".
func BuildItemKey(issueURL string) string {
	u, err := url.Parse(issueURL)
	if err != nil || u.Host == "" {
		return issueURL
	}

	// Only normalize GitHub URLs
	if u.Host != "github.com" {
		return issueURL
	}

	// Parse path: /owner/repo/issues/42
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 4 && parts[2] == "issues" {
		return parts[0] + "/" + parts[1] + "#" + parts[3]
	}

	return issueURL
}

// ClearProcessedItems resets all processed-item tracking for a pipeline,
// allowing previously processed items to be re-processed.
// This accepts the full StateStore since ClearProcessedItems is not on the
// ProcessedItemTracker interface (it's an admin operation, not a runtime one).
func ClearProcessedItems(store state.StateStore, pipelineName string) error {
	return store.ClearProcessedItems(pipelineName)
}
