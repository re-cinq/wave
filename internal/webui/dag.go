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
	ID          string
	Label       string
	Persona     string
	Status      string
	Duration    string
	Tokens      int
	Contract    string
	Artifacts   string
	StepType    string // "conditional", "command", "gate", "pipeline", or "" (persona)
	Script      string // Shell script for command steps
	SubPipeline string // Referenced pipeline for pipeline steps
	GatePrompt  string // Gate prompt/message
	GateChoices string // Comma-separated gate choice labels
	EdgeInfo    string // Serialized edge conditions for conditional steps
	Thread      string // Thread group ID for conversation continuity
	X           int
	Y           int
}

// DAGLayoutEdge is an edge with computed bezier curve points for SVG.
type DAGLayoutEdge struct {
	From       string
	To         string
	FromX      int
	FromY      int
	ToX        int
	ToY        int
	CX1        int
	CY1        int
	CX2        int
	CY2        int
	IsBackward bool   // True for edges pointing to earlier layers (loop edges)
	Condition  string // Edge condition label (e.g. "outcome=success")
}

const (
	nodeWidth  = 140
	nodeHeight = 80
	layerGapX  = 180 // horizontal gap between layers (left→right columns)
	nodeGapY   = 80  // vertical gap between nodes in the same layer (rows)
	paddingX   = 20
	paddingY   = 20
)

// ComputeDAGLayout takes pipeline step definitions and step progress,
// then computes a left-to-right layered layout for SVG rendering.
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

	// Compute positions — left-to-right: layers on X axis (columns), nodes within layer on Y axis (rows)
	layout := &DAGLayout{}
	maxNodesInLayer := 0
	for _, layer := range layers {
		if len(layer) > maxNodesInLayer {
			maxNodesInLayer = len(layer)
		}
	}

	for layerIdx, layer := range layers {
		// Center nodes within the layer vertically
		layerHeight := len(layer)*nodeGapY - (nodeGapY - nodeHeight)
		totalHeight := maxNodesInLayer*nodeGapY - (nodeGapY - nodeHeight)
		offsetY := (totalHeight - layerHeight) / 2

		for nodeIdx, id := range layer {
			s := stepMap[id]
			x := paddingX + layerIdx*layerGapX
			y := paddingY + offsetY + nodeIdx*nodeGapY

			layout.Nodes = append(layout.Nodes, DAGLayoutNode{
				ID:          s.ID,
				Label:       s.ID,
				Persona:     s.Persona,
				Status:      s.Status,
				Duration:    s.Duration,
				Tokens:      s.Tokens,
				Contract:    s.Contract,
				Artifacts:   s.Artifacts,
				StepType:    s.StepType,
				Script:      s.Script,
				SubPipeline: s.SubPipeline,
				GatePrompt:  s.GatePrompt,
				GateChoices: s.GateChoices,
				EdgeInfo:    s.EdgeInfo,
				Thread:      s.Thread,
				X:           x,
				Y:           y,
			})
		}
	}

	layout.Width = paddingX*2 + len(layers)*layerGapX - (layerGapX - nodeWidth)
	if layout.Width < nodeWidth+paddingX*2 {
		layout.Width = nodeWidth + paddingX*2
	}
	layout.Height = paddingY*2 + maxNodesInLayer*nodeGapY - (nodeGapY - nodeHeight)
	if layout.Height < nodeHeight+paddingY*2 {
		layout.Height = nodeHeight + paddingY*2
	}

	// Build node position map and layer index for edge computation
	nodePos := make(map[string][2]int)
	nodeLayerIdx := make(map[string]int)
	for _, n := range layout.Nodes {
		nodePos[n.ID] = [2]int{n.X, n.Y}
	}
	for layerIdx, layer := range layers {
		for _, id := range layer {
			nodeLayerIdx[id] = layerIdx
		}
	}

	// Build edge condition lookup from step edges
	edgeConditionMap := make(map[[2]string]string) // [from, to] -> condition
	for _, s := range steps {
		for _, e := range s.Edges {
			edgeConditionMap[[2]string{s.ID, e.Target}] = e.Condition
		}
	}

	// Compute edges — from right-center of source node to left-center of target node
	// Also include graph-mode edges (step.Edges) in addition to dependency edges
	type edgePair struct {
		from, to string
	}
	seen := make(map[edgePair]bool)

	addEdge := func(from, to, condition string) {
		ep := edgePair{from, to}
		if seen[ep] {
			return
		}
		seen[ep] = true

		fromPos, ok1 := nodePos[from]
		toPos, ok2 := nodePos[to]
		if !ok1 || !ok2 {
			return
		}

		fromLayer := nodeLayerIdx[from]
		toLayer := nodeLayerIdx[to]
		isBackward := toLayer <= fromLayer

		fromX := fromPos[0] + nodeWidth
		fromY := fromPos[1] + nodeHeight/2
		toX := toPos[0]
		toY := toPos[1] + nodeHeight/2
		midX := (fromX + toX) / 2

		if isBackward {
			// For backward edges, curve below the nodes
			belowY := layout.Height + 20
			midX = (fromPos[0] + toPos[0] + nodeWidth) / 2
			layout.Edges = append(layout.Edges, DAGLayoutEdge{
				From:       from,
				To:         to,
				FromX:      fromX,
				FromY:      fromY,
				ToX:        toX,
				ToY:        toY,
				CX1:        midX,
				CY1:        belowY,
				CX2:        midX,
				CY2:        belowY,
				IsBackward: true,
				Condition:  condition,
			})
		} else {
			layout.Edges = append(layout.Edges, DAGLayoutEdge{
				From:       from,
				To:         to,
				FromX:      fromX,
				FromY:      fromY,
				ToX:        toX,
				ToY:        toY,
				CX1:        midX,
				CY1:        fromY,
				CX2:        midX,
				CY2:        toY,
				IsBackward: false,
				Condition:  condition,
			})
		}
	}

	for _, s := range steps {
		// Standard dependency edges (drawn as dep -> s.ID)
		for _, dep := range s.Dependencies {
			cond := edgeConditionMap[[2]string{dep, s.ID}]
			addEdge(dep, s.ID, cond)
		}
		// Graph-mode edges (drawn as s.ID -> target)
		for _, e := range s.Edges {
			addEdge(s.ID, e.Target, e.Condition)
		}
	}

	// Expand height if backward edges exist (they curve below)
	for _, e := range layout.Edges {
		if e.IsBackward {
			needed := layout.Height + 50
			if needed > layout.Height {
				layout.Height = needed
			}
			break
		}
	}

	return layout
}

// DAGStepInput is the input for DAG layout computation.
type DAGStepInput struct {
	ID           string
	Persona      string
	Status       string
	Duration     string
	Tokens       int
	Contract     string
	Artifacts    string
	Dependencies []string
	StepType     string // "conditional", "command", "gate", "pipeline", or "" (persona)
	Script       string
	SubPipeline  string
	GatePrompt   string
	GateChoices  string
	EdgeInfo     string
	Thread       string         // Thread group ID for conversation continuity
	Edges        []DAGEdgeInput // Outgoing edges for graph-mode routing
}

// DAGEdgeInput carries edge metadata from the pipeline definition.
type DAGEdgeInput struct {
	Target    string
	Condition string
}

// stripExcludedDeps removes dependency references to excluded step IDs from
// the given DAGStepInput slice. This prevents broken edges after filtering
// out rework-only steps.
func stripExcludedDeps(steps []DAGStepInput, excluded map[string]bool) {
	if len(excluded) == 0 {
		return
	}
	for i := range steps {
		var filtered []string
		for _, dep := range steps[i].Dependencies {
			if !excluded[dep] {
				filtered = append(filtered, dep)
			}
		}
		steps[i].Dependencies = filtered
	}
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
