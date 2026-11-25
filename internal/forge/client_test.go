package forge

import (
	"testing"
)

func TestGitHubClient_ImplementsClient(t *testing.T) {
	// Compile-time interface check
	var _ Client = (*GitHubClient)(nil)
}
