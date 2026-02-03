package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/github"
)

// GitHubAdapter wraps the GitHub API client as a Wave adapter
type GitHubAdapter struct {
	client *github.Client
}

// NewGitHubAdapter creates a new GitHub adapter
func NewGitHubAdapter(token string) *GitHubAdapter {
	if token == "" {
		// Try to get token from environment
		token = os.Getenv("GITHUB_TOKEN")
	}

	client := github.NewClient(github.ClientConfig{
		Token: token,
	})

	return &GitHubAdapter{
		client: client,
	}
}

// Run executes a GitHub operation as a Wave adapter
func (a *GitHubAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	// Parse the prompt to determine what GitHub operation to perform
	operation, err := a.parseOperation(cfg.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse operation: %w", err)
	}

	// Execute the operation
	result, err := a.executeOperation(ctx, operation)
	if err != nil {
		return nil, fmt.Errorf("operation failed: %w", err)
	}

	// Format result as AdapterResult
	return a.formatResult(result)
}

// GitHubOperation represents a GitHub operation to perform
type GitHubOperation struct {
	Type   string                 `json:"type"` // list_issues, get_issue, update_issue, create_pr, etc.
	Owner  string                 `json:"owner"`
	Repo   string                 `json:"repo"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// parseOperation parses the prompt to extract the GitHub operation
func (a *GitHubAdapter) parseOperation(prompt string) (*GitHubOperation, error) {
	// Try to parse as JSON first
	var op GitHubOperation
	if err := json.Unmarshal([]byte(prompt), &op); err == nil {
		return &op, nil
	}

	// If not JSON, parse as natural language
	// This is a simplified parser - in production, you'd want more sophisticated parsing
	promptLower := strings.ToLower(prompt)

	// Extract owner/repo from common patterns
	owner, repo := extractRepoInfo(prompt)

	if strings.Contains(promptLower, "list issues") || strings.Contains(promptLower, "scan issues") {
		return &GitHubOperation{
			Type:   "list_issues",
			Owner:  owner,
			Repo:   repo,
			Params: make(map[string]interface{}),
		}, nil
	}

	if strings.Contains(promptLower, "analyze issues") || strings.Contains(promptLower, "find poor") {
		return &GitHubOperation{
			Type:   "analyze_issues",
			Owner:  owner,
			Repo:   repo,
			Params: map[string]interface{}{"threshold": 70},
		}, nil
	}

	if strings.Contains(promptLower, "update issue") || strings.Contains(promptLower, "enhance issue") {
		return &GitHubOperation{
			Type:   "update_issue",
			Owner:  owner,
			Repo:   repo,
			Params: make(map[string]interface{}),
		}, nil
	}

	if strings.Contains(promptLower, "create pr") || strings.Contains(promptLower, "create pull request") {
		return &GitHubOperation{
			Type:   "create_pr",
			Owner:  owner,
			Repo:   repo,
			Params: make(map[string]interface{}),
		}, nil
	}

	return nil, fmt.Errorf("unable to determine operation from prompt: %s", prompt)
}

// extractRepoInfo extracts owner and repo from various formats
func extractRepoInfo(text string) (owner, repo string) {
	// Look for owner/repo pattern
	if strings.Contains(text, "/") {
		parts := strings.Split(text, "/")
		for i := 0; i < len(parts)-1; i++ {
			// Check if this looks like owner/repo
			if isValidRepoName(parts[i]) && isValidRepoName(parts[i+1]) {
				return strings.TrimSpace(parts[i]), strings.TrimSpace(parts[i+1])
			}
		}
	}

	// Try to extract from current git repo
	// This would require git CLI integration
	return "unknown", "unknown"
}

// isValidRepoName checks if a string looks like a valid repo name
func isValidRepoName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	// Basic validation - alphanumeric, dash, underscore
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

// executeOperation performs the GitHub operation
func (a *GitHubAdapter) executeOperation(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	switch op.Type {
	case "list_issues":
		return a.listIssues(ctx, op)
	case "analyze_issues":
		return a.analyzeIssues(ctx, op)
	case "get_issue":
		return a.getIssue(ctx, op)
	case "update_issue":
		return a.updateIssue(ctx, op)
	case "create_pr":
		return a.createPR(ctx, op)
	case "get_repo":
		return a.getRepo(ctx, op)
	case "create_branch":
		return a.createBranch(ctx, op)
	default:
		return nil, fmt.Errorf("unsupported operation type: %s", op.Type)
	}
}

// listIssues lists issues for a repository
func (a *GitHubAdapter) listIssues(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	opts := github.ListIssuesOptions{
		State:   "open",
		PerPage: 100,
	}

	if state, ok := op.Params["state"].(string); ok {
		opts.State = state
	}

	issues, err := a.client.ListIssues(ctx, op.Owner, op.Repo, opts)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
		"owner":  op.Owner,
		"repo":   op.Repo,
	}, nil
}

// analyzeIssues analyzes issues for quality
func (a *GitHubAdapter) analyzeIssues(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	threshold := 70
	if t, ok := op.Params["threshold"].(float64); ok {
		threshold = int(t)
	} else if t, ok := op.Params["threshold"].(int); ok {
		threshold = t
	}

	analyzer := github.NewAnalyzer(a.client)
	analyses, err := analyzer.FindPoorQualityIssues(ctx, op.Owner, op.Repo, threshold)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"analyses": analyses,
		"count":    len(analyses),
		"threshold": threshold,
		"owner":    op.Owner,
		"repo":     op.Repo,
	}, nil
}

// getIssue retrieves a single issue
func (a *GitHubAdapter) getIssue(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	number, ok := op.Params["number"].(int)
	if !ok {
		if numFloat, ok := op.Params["number"].(float64); ok {
			number = int(numFloat)
		} else {
			return nil, fmt.Errorf("issue number required")
		}
	}

	issue, err := a.client.GetIssue(ctx, op.Owner, op.Repo, number)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// updateIssue updates an issue
func (a *GitHubAdapter) updateIssue(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	number, ok := op.Params["number"].(int)
	if !ok {
		if numFloat, ok := op.Params["number"].(float64); ok {
			number = int(numFloat)
		} else {
			return nil, fmt.Errorf("issue number required")
		}
	}

	var update github.IssueUpdate

	if title, ok := op.Params["title"].(string); ok {
		update.Title = &title
	}
	if body, ok := op.Params["body"].(string); ok {
		update.Body = &body
	}
	if state, ok := op.Params["state"].(string); ok {
		update.State = &state
	}
	if labels, ok := op.Params["labels"].([]string); ok {
		update.Labels = &labels
	}

	issue, err := a.client.UpdateIssue(ctx, op.Owner, op.Repo, number, update)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// createPR creates a pull request
func (a *GitHubAdapter) createPR(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	title, ok := op.Params["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title required")
	}

	head, ok := op.Params["head"].(string)
	if !ok {
		return nil, fmt.Errorf("head branch required")
	}

	base, ok := op.Params["base"].(string)
	if !ok {
		base = "main" // Default to main
	}

	req := github.CreatePullRequestRequest{
		Title: title,
		Head:  head,
		Base:  base,
	}

	if body, ok := op.Params["body"].(string); ok {
		req.Body = body
	}

	pr, err := a.client.CreatePullRequest(ctx, op.Owner, op.Repo, req)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

// getRepo retrieves repository information
func (a *GitHubAdapter) getRepo(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	repo, err := a.client.GetRepository(ctx, op.Owner, op.Repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// createBranch creates a new branch
func (a *GitHubAdapter) createBranch(ctx context.Context, op *GitHubOperation) (interface{}, error) {
	branchName, ok := op.Params["branch"].(string)
	if !ok {
		return nil, fmt.Errorf("branch name required")
	}

	fromRef, ok := op.Params["from"].(string)
	if !ok {
		fromRef = "main" // Default to main
	}

	ref, err := a.client.CreateBranch(ctx, op.Owner, op.Repo, branchName, fromRef)
	if err != nil {
		return nil, err
	}

	return ref, nil
}

// formatResult formats the operation result as an AdapterResult
func (a *GitHubAdapter) formatResult(data interface{}) (*AdapterResult, error) {
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Create result
	result := &AdapterResult{
		ExitCode:      0,
		Stdout:        strings.NewReader(string(jsonData)),
		ResultContent: string(jsonData),
		TokensUsed:    estimateTokens(string(jsonData)),
		Artifacts:     []string{},
	}

	return result, nil
}

// GitHubAdapterWithContext provides a context-aware wrapper
type GitHubAdapterWithContext struct {
	adapter *GitHubAdapter
	ctx     context.Context
}

// NewGitHubAdapterWithContext creates a GitHub adapter with context
func NewGitHubAdapterWithContext(ctx context.Context, token string) *GitHubAdapterWithContext {
	return &GitHubAdapterWithContext{
		adapter: NewGitHubAdapter(token),
		ctx:     ctx,
	}
}

// Execute runs a GitHub operation with the stored context
func (a *GitHubAdapterWithContext) Execute(operation string, params map[string]interface{}) (interface{}, error) {
	op := &GitHubOperation{
		Type:   operation,
		Params: params,
	}

	// Extract owner/repo from params
	if owner, ok := params["owner"].(string); ok {
		op.Owner = owner
	}
	if repo, ok := params["repo"].(string); ok {
		op.Repo = repo
	}

	return a.adapter.executeOperation(a.ctx, op)
}

// GitHubWorkflowRunner orchestrates complex GitHub workflows
type GitHubWorkflowRunner struct {
	adapter  *GitHubAdapter
	analyzer *github.Analyzer
}

// NewGitHubWorkflowRunner creates a workflow runner
func NewGitHubWorkflowRunner(token string) *GitHubWorkflowRunner {
	adapter := NewGitHubAdapter(token)
	return &GitHubWorkflowRunner{
		adapter:  adapter,
		analyzer: github.NewAnalyzer(adapter.client),
	}
}

// EnhanceIssuesWorkflow runs the complete issue enhancement workflow
func (r *GitHubWorkflowRunner) EnhanceIssuesWorkflow(ctx context.Context, owner, repo string, threshold int) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	// Step 1: Find poor quality issues
	analyses, err := r.analyzer.FindPoorQualityIssues(ctx, owner, repo, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to find poor quality issues: %w", err)
	}

	results["analyzed_count"] = len(analyses)
	results["threshold"] = threshold

	// Step 2: Generate enhancement suggestions
	enhanced := make([]*github.IssueAnalysis, 0, len(analyses))
	for _, analysis := range analyses {
		r.analyzer.GenerateEnhancementSuggestions(analysis.Issue, analysis)
		enhanced = append(enhanced, analysis)
	}

	results["enhanced_issues"] = enhanced
	results["timestamp"] = time.Now()

	return results, nil
}
