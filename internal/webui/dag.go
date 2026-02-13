//go:build webui

package webui

// DAGLayout holds the computed layout for SVG rendering.
type DAGLayout struct {
	Nodes  []DAGLayoutNode
	Edges  []DAGLayoutEdge
	Width  int
	Height int
}

// DAGLayoutNode is a node with computed position for SVG rendering.
type DAGLayoutNode struct {
	ID      string
	Label   string
	Persona string
	Status  string
	X       int
	Y       int
}

// DAGLayoutEdge is an edge with computed bezier curve points for SVG.
type DAGLayoutEdge struct {
	From  string
	To    string
	FromX int
	FromY int
	ToX   int
	ToY   int
	CX1   int
	CY1   int
	CX2   int
	CY2   int
}

const (
	nodeWidth  = 140
	nodeHeight = 50
	layerGapY  = 80  // vertical gap between layers (top→bottom)
	nodeGapX   = 170 // horizontal gap between nodes in the same layer
	paddingX   = 20
	paddingY   = 20
)

// ComputeDAGLayout takes pipeline step definitions and step progress,
// then computes a top-to-bottom layered layout for SVG rendering.
func ComputeDAGLayout(steps []DAGStepInput) *DAGLayout {
	if len(steps) == 0 {
		return nil
	}

	// Build adjacency and in-degree maps
	adj := make(map[string][]string)
	inDeg := make(map[string]int)
	stepMap := make(map[string]DAGStepInput)

	for _, s := range steps {
		stepMap[s.ID] = s
		if _, ok := inDeg[s.ID]; !ok {
			inDeg[s.ID] = 0
		}
		for _, dep := range s.Dependencies {
			adj[dep] = append(adj[dep], s.ID)
			inDeg[s.ID]++
		}
	}

	// Topological sort with layer assignment (Kahn's algorithm)
	layers := assignLayers(steps, adj, inDeg)

	// Compute positions — top-to-bottom: layers on Y axis, nodes within layer on X axis
	layout := &DAGLayout{}
	maxNodesInLayer := 0
	for _, layer := range layers {
		if len(layer) > maxNodesInLayer {
			maxNodesInLayer = len(layer)
		}
	}

	for layerIdx, layer := range layers {
		// Center nodes within the layer horizontally
		layerWidth := len(layer)*nodeGapX - (nodeGapX - nodeWidth)
		totalWidth := maxNodesInLayer*nodeGapX - (nodeGapX - nodeWidth)
		offsetX := (totalWidth - layerWidth) / 2

		for nodeIdx, id := range layer {
			s := stepMap[id]
			x := paddingX + offsetX + nodeIdx*nodeGapX
			y := paddingY + layerIdx*layerGapY

			layout.Nodes = append(layout.Nodes, DAGLayoutNode{
				ID:      s.ID,
				Label:   s.ID,
				Persona: s.Persona,
				Status:  s.Status,
				X:       x,
				Y:       y,
			})
		}
	}

	layout.Width = paddingX*2 + maxNodesInLayer*nodeGapX - (nodeGapX - nodeWidth)
	if layout.Width < nodeWidth+paddingX*2 {
		layout.Width = nodeWidth + paddingX*2
	}
	layout.Height = paddingY*2 + len(layers)*layerGapY - (layerGapY - nodeHeight)
	if layout.Height < nodeHeight+paddingY*2 {
		layout.Height = nodeHeight + paddingY*2
	}

	// Build node position map for edge computation
	nodePos := make(map[string][2]int)
	for _, n := range layout.Nodes {
		nodePos[n.ID] = [2]int{n.X, n.Y}
	}

	// Compute edges — from bottom of source node to top of target node
	for _, s := range steps {
		for _, dep := range s.Dependencies {
			fromPos, ok1 := nodePos[dep]
			toPos, ok2 := nodePos[s.ID]
			if !ok1 || !ok2 {
				continue
			}

			fromX := fromPos[0] + nodeWidth/2
			fromY := fromPos[1] + nodeHeight
			toX := toPos[0] + nodeWidth/2
			toY := toPos[1]
			midY := (fromY + toY) / 2

			layout.Edges = append(layout.Edges, DAGLayoutEdge{
				From:  dep,
				To:    s.ID,
				FromX: fromX,
				FromY: fromY,
				ToX:   toX,
				ToY:   toY,
				CX1:   fromX,
				CY1:   midY,
				CX2:   toX,
				CY2:   midY,
			})
		}
	}

	return layout
}

// DAGStepInput is the input for DAG layout computation.
type DAGStepInput struct {
	ID           string
	Persona      string
	Status       string
	Dependencies []string
}

// assignLayers uses Kahn's algorithm to assign nodes to layers.
func assignLayers(steps []DAGStepInput, adj map[string][]string, inDeg map[string]int) [][]string {
	// Copy in-degree map
	deg := make(map[string]int)
	for k, v := range inDeg {
		deg[k] = v
	}

	var layers [][]string
	var queue []string

	// Start with nodes that have no dependencies
	for _, s := range steps {
		if deg[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	for len(queue) > 0 {
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
