package continuous

import (
	"context"
	"fmt"
)

// WorkItemSource produces work items for the continuous loop.
type WorkItemSource interface {
	// Next returns the next work item, or nil when exhausted.
	Next(ctx context.Context) (*WorkItem, error)

	// Name returns a human-readable description of the source.
	Name() string
}

// NewSourceFromConfig creates a WorkItemSource from a parsed SourceConfig.
func NewSourceFromConfig(cfg *SourceConfig) (WorkItemSource, error) {
	switch cfg.Provider {
	case "github":
		return NewGitHubSource(cfg.Params)
	case "file":
		path := cfg.Params["path"]
		if path == "" {
			return nil, fmt.Errorf("file source requires a path parameter")
		}
		return NewFileSource(path)
	default:
		return nil, fmt.Errorf("unknown source provider: %q (supported: github, file)", cfg.Provider)
	}
}
