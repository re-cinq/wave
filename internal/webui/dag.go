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
	layerGapX  = 200
	nodeGapY   = 80
	paddingX   = 40
	paddingY   = 40
)

// ComputeDAGLayout takes pipeline step definitions and step progress,
// then computes a Sugiyama-style layered layout for SVG rendering.
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

	// Compute positions
	layout := &DAGLayout{}
	maxNodesInLayer := 0
	for _, layer := range layers {
		if len(layer) > maxNodesInLayer {
			maxNodesInLayer = len(layer)
		}
	}

	for layerIdx, layer := range layers {
		for nodeIdx, id := range layer {
			s := stepMap[id]
			x := paddingX + layerIdx*layerGapX
			y := paddingY + nodeIdx*nodeGapY

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

	layout.Width = paddingX*2 + len(layers)*layerGapX
	if layout.Width < nodeWidth+paddingX*2 {
		layout.Width = nodeWidth + paddingX*2
	}
	layout.Height = paddingY*2 + maxNodesInLayer*nodeGapY
	if layout.Height < nodeHeight+paddingY*2 {
		layout.Height = nodeHeight + paddingY*2
	}

	// Build node position map for edge computation
	nodePos := make(map[string][2]int)
	for _, n := range layout.Nodes {
		nodePos[n.ID] = [2]int{n.X, n.Y}
	}

	// Compute edges with bezier control points
	for _, s := range steps {
		for _, dep := range s.Dependencies {
			fromPos, ok1 := nodePos[dep]
			toPos, ok2 := nodePos[s.ID]
			if !ok1 || !ok2 {
				continue
			}

			fromX := fromPos[0] + nodeWidth
			fromY := fromPos[1] + nodeHeight/2
			toX := toPos[0]
			toY := toPos[1] + nodeHeight/2
			midX := (fromX + toX) / 2

			layout.Edges = append(layout.Edges, DAGLayoutEdge{
				From:  dep,
				To:    s.ID,
				FromX: fromX,
				FromY: fromY,
				ToX:   toX,
				ToY:   toY,
				CX1:   midX,
				CY1:   fromY,
				CX2:   midX,
				CY2:   toY,
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
