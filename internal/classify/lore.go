package classify

// LoreProvider supplies project-specific context from Lore (re-cinq/lore)
// that enriches task classification. Lore is an MCP server providing shared
// context infrastructure — org conventions, agent memories, knowledge graph,
// and task history.
//
// The interface mirrors Lore's two key MCP tools:
//   - assemble_context: retrieves structured context from all sources
//   - search_memory: semantic search over org memories and facts
//
// Implementations may call the Lore MCP server directly, use the HTTP API,
// or pull from cached context. Wave works identically without Lore — the
// NoOpLoreProvider ensures classification is never blocked by Lore
// availability.
type LoreProvider interface {
	// GetTaskContext retrieves assembled context relevant to the given input.
	// Maps to Lore's assemble_context MCP tool with template="default".
	// Returns a TaskContext with domain hints, conventions, and memories
	// that the classifier can use to enrich its decisions.
	GetTaskContext(input string) TaskContext
}

// TaskContext holds assembled context from Lore for a classification decision.
type TaskContext struct {
	// Hints are advisory classification signals from historical data.
	Hints []LoreHint

	// Conventions are org-wide rules that may influence pipeline selection
	// (e.g. "all auth changes require security review").
	Conventions []string

	// Memories are relevant past decisions or outcomes from search_memory.
	Memories []MemoryResult
}

// LoreHint is a single advisory signal from Lore's historical data.
type LoreHint struct {
	Domain     Domain     // suggested domain (empty = no opinion)
	Complexity Complexity // suggested complexity (empty = no opinion)
	Confidence float64    // 0.0–1.0 confidence in this hint
	Source     string     // e.g. "orchestration_history", "memory", "graph"
}

// MemoryResult is a single result from Lore's search_memory tool.
type MemoryResult struct {
	Key    string  // memory key
	Value  string  // memory content
	Score  float64 // relevance score
	Source string  // "memory", "fact", "episode", "graph"
}

// NoOpLoreProvider returns empty context. Used as the default when Lore
// is not configured. Classification works identically without Lore.
type NoOpLoreProvider struct{}

// GetTaskContext returns empty context for the no-op provider.
func (NoOpLoreProvider) GetTaskContext(string) TaskContext { return TaskContext{} }

var activeLoreProvider LoreProvider = NoOpLoreProvider{}

// RegisterLoreProvider sets the active lore provider. Not concurrency-safe;
// call during init or startup before any classification runs.
func RegisterLoreProvider(p LoreProvider) {
	if p == nil {
		p = NoOpLoreProvider{}
	}
	activeLoreProvider = p
}

// loreProvider returns the currently registered provider.
func loreProvider() LoreProvider {
	return activeLoreProvider
}
