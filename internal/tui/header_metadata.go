package tui

// HealthStatus represents the aggregate health of pipeline runs.
type HealthStatus int

const (
	HealthOK   HealthStatus = iota // No failures
	HealthWarn                     // Soft failures
	HealthErr                      // Hard failures
)

// String returns a display representation of the health status.
func (h HealthStatus) String() string {
	switch h {
	case HealthWarn:
		return "▲ WARN"
	case HealthErr:
		return "✗ ERR"
	default:
		return "● OK"
	}
}

// GitHubAuthState represents the GitHub CLI authentication state.
type GitHubAuthState int

const (
	GitHubNotConfigured GitHubAuthState = iota // No gh CLI or auth
	GitHubOffline                              // Auth exists, API unreachable
	GitHubConnected                            // Working connection
)

// GitState holds the result of a git state fetch.
type GitState struct {
	Branch     string
	CommitHash string
	IsDirty    bool
	RemoteName string
}

// ManifestInfo holds the result of a manifest info fetch.
type ManifestInfo struct {
	ProjectName string
	RepoName    string // owner/repo format
}

// GitHubInfo holds the result of a GitHub info fetch.
type GitHubInfo struct {
	AuthState   GitHubAuthState
	IssuesCount int
}

// HeaderMetadata holds all displayable project metadata fields.
type HeaderMetadata struct {
	// Git state
	Branch     string
	CommitHash string
	IsDirty    bool
	RemoteName string

	// Manifest state
	ProjectName string
	RepoName    string

	// Pipeline state
	RunningCount int
	StepCount    int
	Health       HealthStatus

	// GitHub state
	IssuesCount int
	GitHubState GitHubAuthState

	// Override state
	OverrideBranch string
}

// MetadataProvider is an interface for fetching project metadata.
// It decouples data fetching from rendering for testability.
type MetadataProvider interface {
	FetchGitState() (GitState, error)
	FetchManifestInfo() (ManifestInfo, error)
	FetchGitHubInfo(repo string) (GitHubInfo, error)
	FetchPipelineHealth() (HealthStatus, error)
}
