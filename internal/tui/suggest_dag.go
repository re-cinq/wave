package tui

import (
	"fmt"
	"strings"
)

// RenderDAG produces a text-based DAG visualization for the detail pane.
// For sequences: vertical flow with arrows between pipeline names.
// For parallels: grouped layout indicating concurrent execution.
// For single proposals: returns empty string (no DAG needed).
func RenderDAG(proposal SuggestProposedPipeline) string {
	if len(proposal.Sequence) <= 1 {
		return ""
	}

	var sb strings.Builder

	switch proposal.Type {
	case "sequence":
		for i, name := range proposal.Sequence {
			fmt.Fprintf(&sb, "  [%s]", name)
			if i < len(proposal.Sequence)-1 {
				sb.WriteString("\n    │\n    ▼\n")
			}
		}
		sb.WriteString("\n")

	case "parallel":
		for i, name := range proposal.Sequence {
			switch {
			case i == 0:
				fmt.Fprintf(&sb, "  ┌ [%s]\n", name)
			case i == len(proposal.Sequence)-1:
				fmt.Fprintf(&sb, "  └ [%s]\n", name)
			default:
				fmt.Fprintf(&sb, "  ├ [%s]\n", name)
			}
		}
		sb.WriteString("  (concurrent)\n")
	}

	return sb.String()
}
