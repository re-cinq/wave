package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const minTermWidth = 20

// RenderDAGPreview produces a text-based DAG preview of pipeline proposals
// suitable for terminal display. It uses Kahn's algorithm for layer assignment
// and renders layers top-to-bottom with box-drawing connectors.
func RenderDAGPreview(proposals []Proposal, termWidth int) string {
	if len(proposals) == 0 {
		return ""
	}
	if termWidth < minTermWidth {
		termWidth = minTermWidth
	}

	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Build lookup and adjacency structures.
	proposalMap := make(map[string]Proposal, len(proposals))
	for _, p := range proposals {
		proposalMap[p.Pipeline] = p
	}

	adj := make(map[string][]string) // parent -> children
	inDeg := make(map[string]int, len(proposals))
	for _, p := range proposals {
		if _, ok := inDeg[p.Pipeline]; !ok {
			inDeg[p.Pipeline] = 0
		}
		for _, dep := range p.Dependencies {
			if _, exists := proposalMap[dep]; exists {
				adj[dep] = append(adj[dep], p.Pipeline)
				inDeg[p.Pipeline]++
			}
		}
	}

	// Kahn's algorithm: assign proposals to layers.
	layers := assignProposalLayers(proposals, adj, inDeg)
	if len(layers) == 0 {
		return ""
	}

	// Single proposal shortcut.
	if len(proposals) == 1 {
		label := formatNode(proposals[0].Pipeline, nameStyle, termWidth)
		return centerLine(label, termWidth)
	}

	var sb strings.Builder

	for layerIdx, layer := range layers {
		// Detect parallel groups within this layer.
		groups := groupByParallel(layer, proposalMap)
		hasParallelGroup := len(groups) == 1 && groups[0].group != "" && len(groups[0].names) > 1

		// Render parallel group header if applicable.
		if hasParallelGroup {
			header := dimStyle.Render("── parallel ──")
			sb.WriteString(centerLine(header, termWidth))
			sb.WriteByte('\n')
		}

		// Render nodes on this layer.
		nodeLabels := make([]string, len(layer))
		for i, name := range layer {
			nodeLabels[i] = formatNode(name, nameStyle, termWidth)
		}

		nodesLine := buildNodesLine(nodeLabels, termWidth)
		sb.WriteString(nodesLine)
		sb.WriteByte('\n')

		// Render parallel group footer if applicable.
		if hasParallelGroup {
			footer := dimStyle.Render("── parallel ──")
			sb.WriteString(centerLine(footer, termWidth))
			sb.WriteByte('\n')
		}

		// Draw connectors between this layer and the next.
		if layerIdx < len(layers)-1 {
			nextLayer := layers[layerIdx+1]
			connectorLines := renderConnectors(layer, nextLayer, adj, dimStyle, termWidth)
			sb.WriteString(connectorLines)
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// assignProposalLayers uses Kahn's algorithm to assign proposals to layers.
// Layer 0 contains proposals with no dependencies, layer 1 contains proposals
// that depend only on layer 0, and so on.
func assignProposalLayers(proposals []Proposal, adj map[string][]string, inDeg map[string]int) [][]string {
	deg := make(map[string]int, len(inDeg))
	for k, v := range inDeg {
		deg[k] = v
	}

	// Collect names in insertion order for deterministic seeding.
	ordered := make([]string, 0, len(proposals))
	for _, p := range proposals {
		ordered = append(ordered, p.Pipeline)
	}

	var layers [][]string
	var queue []string

	for _, name := range ordered {
		if deg[name] == 0 {
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		sort.Strings(queue)
		layers = append(layers, queue)
		var nextQueue []string
		for _, id := range queue {
			for _, next := range adj[id] {
				deg[next]--
				if deg[next] == 0 {
					nextQueue = append(nextQueue, next)
				}
			}
		}
		queue = nextQueue
	}

	return layers
}

// parallelGroupInfo holds a set of proposals sharing the same ParallelGroup.
type parallelGroupInfo struct {
	group string
	names []string
}

// groupByParallel partitions a layer's nodes by their ParallelGroup.
// Nodes without a ParallelGroup each form their own singleton entry.
func groupByParallel(layer []string, proposalMap map[string]Proposal) []parallelGroupInfo {
	// Collect groups preserving first-seen order.
	var order []string
	groups := make(map[string][]string)

	for _, name := range layer {
		p := proposalMap[name]
		key := p.ParallelGroup
		if key == "" {
			// Each ungrouped node is its own "group" keyed by a unique sentinel.
			order = append(order, "\x00"+name)
			groups["\x00"+name] = []string{name}
			continue
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], name)
	}

	result := make([]parallelGroupInfo, 0, len(order))
	for _, key := range order {
		g := ""
		if key[0] != '\x00' {
			g = key
		}
		result = append(result, parallelGroupInfo{group: g, names: groups[key]})
	}
	return result
}

// formatNode renders a single node label as [ name ], truncating if it
// would exceed the available terminal width.
func formatNode(name string, nameStyle lipgloss.Style, termWidth int) string {
	// "[ " + name + " ]" = name length + 4 framing characters.
	maxLabelLen := termWidth - 4
	if maxLabelLen < 1 {
		maxLabelLen = 1
	}

	display := name
	if len(display) > maxLabelLen {
		display = display[:maxLabelLen-1] + "…"
	}

	return "[ " + nameStyle.Render(display) + " ]"
}

// buildNodesLine arranges node labels horizontally, centered within termWidth.
func buildNodesLine(labels []string, termWidth int) string {
	if len(labels) == 1 {
		return centerLine(labels[0], termWidth)
	}

	gap := "     "
	joined := strings.Join(labels, gap)
	return centerLine(joined, termWidth)
}

// renderConnectors draws connector lines between a parent layer and child layer.
func renderConnectors(parentLayer, childLayer []string, adj map[string][]string, dimStyle lipgloss.Style, termWidth int) string {
	// Verify that at least one connection exists between these layers.
	childSet := make(map[string]bool, len(childLayer))
	for _, c := range childLayer {
		childSet[c] = true
	}

	hasConnection := false
	for _, parent := range parentLayer {
		for _, child := range adj[parent] {
			if childSet[child] {
				hasConnection = true
				break
			}
		}
		if hasConnection {
			break
		}
	}

	if !hasConnection {
		return ""
	}

	// Linear chain: single parent to single child.
	if len(parentLayer) == 1 && len(childLayer) == 1 {
		return centerLine(dimStyle.Render("│"), termWidth) + "\n"
	}

	// Fan-out: single parent to multiple children.
	if len(parentLayer) == 1 && len(childLayer) > 1 {
		return centerLine(dimStyle.Render("╱")+"     "+dimStyle.Render("╲"), termWidth) + "\n"
	}

	// Fan-in: multiple parents to single child.
	if len(parentLayer) > 1 && len(childLayer) == 1 {
		return centerLine(dimStyle.Render("╲")+"     "+dimStyle.Render("╱"), termWidth) + "\n"
	}

	// General case: multiple parents to multiple children.
	connectors := make([]string, len(childLayer))
	for i := range connectors {
		connectors[i] = dimStyle.Render("│")
	}
	return centerLine(strings.Join(connectors, "     "), termWidth) + "\n"
}

// centerLine centers a string within the given terminal width.
// It uses lipgloss.Width to measure printable width, correctly handling
// ANSI escape sequences. If the text is already wider than termWidth,
// it is returned unmodified.
func centerLine(s string, termWidth int) string {
	textWidth := lipgloss.Width(s)
	if textWidth >= termWidth {
		return s
	}
	pad := (termWidth - textWidth) / 2
	return strings.Repeat(" ", pad) + s
}
