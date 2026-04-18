package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

type ClaudeAdapter struct {
	claudePath string
}

func NewClaudeAdapter() *ClaudeAdapter {
	path := "/usr/local/bin/claude"
	if p, err := exec.LookPath("claude"); err == nil {
		path = p
	}
	return &ClaudeAdapter{claudePath: path}
}

// SandboxOnlySettings is a minimal settings.json structure written only when
// sandbox is enabled. It replaces the full ClaudeSettings struct — model,
// temperature, permissions, and output format now live in agent frontmatter.
type SandboxOnlySettings struct {
	Sandbox *SandboxSettings `json:"sandbox,omitempty"`
}

type SandboxSettings struct {
	Enabled                  bool             `json:"enabled"`
	AllowUnsandboxedCommands bool             `json:"allowUnsandboxedCommands"`
	AutoAllowBashIfSandboxed bool             `json:"autoAllowBashIfSandboxed"`
	Network                  *NetworkSettings `json:"network,omitempty"`
}

type NetworkSettings struct {
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

func (a *ClaudeAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		return nil, fmt.Errorf("WorkspacePath is required — refusing to use project root as workspace")
	}

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.claudePath, args...)
	cmd.Dir = workspacePath

	if cfg.Debug {
		fmt.Printf("[DEBUG] Claude command: %s %s\n", a.claudePath, shelljoinArgs(args))
		fmt.Printf("[DEBUG] Working directory: %s\n", workspacePath)
	}

	cmd.Env = a.buildEnvironment(cfg)

	// Set up process group for clean timeout kill
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	var stderrBuf bytes.Buffer
	var stdoutBuf bytes.Buffer
	stdoutDone := make(chan error, 1)
	stderrDone := make(chan error, 1)

	go func() {
		_, err := io.Copy(&stderrBuf, stderrPipe)
		stderrDone <- err
	}()

	// Stream stdout line-by-line, parsing NDJSON events in real-time
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
		for scanner.Scan() {
			line := scanner.Bytes()
			stdoutBuf.Write(line)
			stdoutBuf.WriteByte('\n')

			// Parse and emit stream events to the callback
			if cfg.OnStreamEvent != nil {
				if evt, ok := parseStreamLine(line); ok {
					cfg.OnStreamEvent(evt)
				}
			}
		}
		stdoutDone <- scanner.Err()
	}()

	// Wait for both streams to finish or context cancellation
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			killProcessGroup(cmd.Process, cfg.ProcessGrace)
		}
		// Wait briefly for stdout to drain so we can capture diagnostic data
		drainTimeout := cfg.StdoutDrain
		if drainTimeout <= 0 {
			drainTimeout = timeouts.StdoutDrain
		}
		select {
		case <-stdoutDone:
		case <-time.After(drainTimeout):
		}
		_ = cmd.Wait()

		// Parse buffered output for token usage and subtype even on timeout
		parsed := a.parseOutput(stdoutBuf.Bytes())
		reason := ClassifyFailure(parsed.Subtype, parsed.ResultContent, ctx.Err())
		return nil, NewStepError(reason, ctx.Err(), parsed.Tokens, parsed.Subtype)
	case err := <-stdoutDone:
		if err != nil {
			return nil, fmt.Errorf("failed to read stdout: %w", err)
		}
	}

	// Wait for stderr to finish too
	<-stderrDone

	cmdErr := cmd.Wait()
	result := &AdapterResult{
		ExitCode: 0,
		Stdout:   bytes.NewReader(stdoutBuf.Bytes()),
	}

	if cmdErr != nil {
		result.ExitCode = exitCodeFromError(cmdErr)
	}

	parsed := a.parseOutput(stdoutBuf.Bytes())
	result.TokensUsed = parsed.Tokens
	result.TokensIn = parsed.TokensIn
	result.TokensOut = parsed.TokensOut
	result.Artifacts = parsed.Artifacts
	result.Subtype = parsed.Subtype

	// Classify failure for non-zero exit codes with context exhaustion indicators
	if result.ExitCode != 0 || parsed.Subtype == "error_max_turns" || parsed.Subtype == "error_during_execution" {
		result.FailureReason = ClassifyFailure(parsed.Subtype, parsed.ResultContent, nil)
	}

	// NOTE: Do NOT re-scan ResultContent for rate limit strings on exit code 0.
	// The CLI always exits non-zero on rate limits, and scanning the full output
	// produces false positives when personas write about "rate limiting" in their
	// analysis (e.g. security reviews mentioning "No rate limiting on endpoints").

	// The persona's text response (ResultContent) is always natural language,
	// not the JSON artifact. Artifact validation is handled by the contract
	// validator which reads the actual file. Skip format validation here.
	result.ResultContent = parsed.ResultContent

	if cfg.Debug {
		fmt.Printf("[DEBUG] Claude exit code: %d\n", result.ExitCode)
		fmt.Printf("[DEBUG] Claude tokens used: %d\n", parsed.Tokens)
		if stderrBuf.Len() > 0 {
			fmt.Printf("[DEBUG] Claude stderr:\n%s\n", stderrBuf.String())
		}
		fmt.Printf("[DEBUG] Claude raw output (%d bytes):\n%s\n", stdoutBuf.Len(), stdoutBuf.String())
		fmt.Printf("[DEBUG] Extracted result content (%d chars):\n%s\n", len(parsed.ResultContent), parsed.ResultContent)
	}

	return result, nil
}

// agentFilePath is the workspace-relative path for the self-contained agent .md
// file. It lives inside .claude/ so it stays out of the project root and is
// excluded by the standard .gitignore rule.
const agentFilePath = ".claude/wave-agent.md"

func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	settingsDir := filepath.Join(workspacePath, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Write minimal settings.json only when sandbox is enabled.
	// All other configuration (model, permissions, tools) lives in the agent frontmatter.
	if cfg.SandboxEnabled {
		sandboxSettings := SandboxOnlySettings{
			Sandbox: &SandboxSettings{
				Enabled:                  true,
				AllowUnsandboxedCommands: false,
				AutoAllowBashIfSandboxed: true,
			},
		}
		if len(cfg.AllowedDomains) > 0 {
			sandboxSettings.Sandbox.Network = &NetworkSettings{
				AllowedDomains: cfg.AllowedDomains,
			}
		}
		settingsPath := filepath.Join(settingsDir, "settings.json")
		settingsData, err := json.MarshalIndent(sandboxSettings, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal sandbox settings: %w", err)
		}
		if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
			return fmt.Errorf("failed to write settings.json: %w", err)
		}
	}

	// 0. Base protocol preamble (shared across all personas)
	baseProtocolPath := filepath.Join(".agents", "personas", "base-protocol.md")
	baseProtocol, err := os.ReadFile(baseProtocolPath)
	if err != nil {
		return fmt.Errorf("failed to read base protocol %s: %w", baseProtocolPath, err)
	}

	// 1. Persona system prompt
	var systemPrompt string
	if cfg.SystemPrompt != "" {
		systemPrompt = cfg.SystemPrompt
	} else {
		personaPath := filepath.Join(".agents", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			systemPrompt = string(data)
		} else {
			systemPrompt = fmt.Sprintf("# %s\n\nYou are operating as the %s persona.\n", cfg.Persona, cfg.Persona)
		}
	}

	// 1.5. Available skills section (resolved from hierarchical config)
	if skillSection := buildSkillSection(cfg.ResolvedSkills); skillSection != "" {
		systemPrompt += skillSection
	}

	// 2. Concurrency hint (when MaxConcurrentAgents > 1); contract schema is
	// appended to the user prompt by the executor, not the agent file.
	contractSection := ""
	// 2.5. Concurrency hint (when MaxConcurrentAgents > 1)
	if hint := buildConcurrencyHint(cfg.MaxConcurrentAgents); hint != "" {
		contractSection += hint
	}

	// Inject TodoWrite into deny list if not already present.
	// This must happen BEFORE buildRestrictionSection so that both the
	// frontmatter (disallowedTools) and the body (## Restrictions) are consistent.
	hasTodoWrite := false
	for _, tool := range cfg.DenyTools {
		if tool == "TodoWrite" {
			hasTodoWrite = true
			break
		}
	}
	if !hasTodoWrite {
		cfg.DenyTools = append(cfg.DenyTools, "TodoWrite")
	}

	// 3. Restriction section from manifest
	restrictions := buildRestrictionSection(cfg)

	// Compile persona into a self-contained agent .md file with YAML frontmatter.
	spec := PersonaSpec{
		Model:        cfg.Model,
		AllowedTools: cfg.AllowedTools,
		DenyTools:    cfg.DenyTools,
	}
	agentMd := PersonaToAgentMarkdown(
		spec,
		string(baseProtocol),
		cfg.OntologySection,
		systemPrompt,
		contractSection,
		restrictions,
	)
	agentMdPath := filepath.Join(workspacePath, agentFilePath)
	if err := os.WriteFile(agentMdPath, []byte(agentMd), 0644); err != nil {
		return fmt.Errorf("failed to write agent .md: %w", err)
	}

	// Provision .claude/skills/ via shared helper. The helper is workspace-scope
	// safe (panics on traversal) and only removes Wave-managed dirs (sentinel-tagged),
	// preserving any user-committed skills inherited from a worktree checkout.
	if err := ProvisionSkills(workspacePath, ".claude/skills", cfg.ResolvedSkills); err != nil {
		return fmt.Errorf("provision claude skills: %w", err)
	}

	// Copy skill command files into workspace .claude/commands/
	if cfg.SkillCommandsDir != "" {
		if err := a.copySkillCommands(settingsDir, cfg.SkillCommandsDir); err != nil {
			return fmt.Errorf("failed to copy skill commands: %w", err)
		}
	}

	return nil
}

// copySkillDir copies a skill source directory into the destination path,
// preserving all subdirectories (references/, scripts/, assets/).
func copySkillDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// copySkillCommands copies skill command files from the source directory
// into the workspace's .claude/commands/ directory.
func (a *ClaudeAdapter) copySkillCommands(settingsDir, sourceDir string) error {
	commandsDir := filepath.Join(settingsDir, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Source doesn't exist, nothing to copy
		}
		return fmt.Errorf("failed to read skill commands directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		src := filepath.Join(sourceDir, entry.Name())
		dst := filepath.Join(commandsDir, entry.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read skill command %q: %w", entry.Name(), err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("failed to write skill command %q: %w", entry.Name(), err)
		}
	}

	return nil
}

// buildEnvironment constructs a curated environment for the Claude Code subprocess.
// It uses the shared BuildCuratedEnvironment for base vars, passthrough, and step env,
// then appends Claude-specific telemetry suppression variables.
func (a *ClaudeAdapter) buildEnvironment(cfg AdapterRunConfig) []string {
	env := BuildCuratedEnvironment(cfg)

	// Claude-specific telemetry suppression
	env = append(env,
		"DISABLE_TELEMETRY=1",
		"DISABLE_ERROR_REPORTING=1",
		"CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1",
		"DISABLE_BUG_COMMAND=1",
	)

	return env
}

func getenvDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func (a *ClaudeAdapter) buildArgs(cfg AdapterRunConfig) []string {
	args := []string{"-p"}

	// Agent mode: pass --agent pointing to the compiled persona .md file.
	// Model, permissions, and tools are embedded in the agent frontmatter.
	args = append(args, "--agent", agentFilePath)

	args = append(args, "--output-format", "stream-json")
	args = append(args, "--verbose")
	// Skip permission prompts — Wave enforces permissions via agent frontmatter.
	// The CLI flag is still required for the top-level process; the frontmatter
	// permissionMode only applies to subagents.
	args = append(args, "--dangerously-skip-permissions")
	args = append(args, "--no-session-persistence")

	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	return args
}

// parseOutputResult holds the parsed output data from NDJSON stream.
type parseOutputResult struct {
	Tokens        int
	TokensIn      int // Input tokens (prompt + cache creation)
	TokensOut     int // Output tokens (completion)
	Artifacts     []string
	ResultContent string
	Subtype       string // Result event subtype: "success", "error_max_turns", "error_during_execution"
}

func (a *ClaudeAdapter) parseOutput(data []byte) parseOutputResult {
	var resultTokens int
	var resultTokensIn int
	var resultTokensOut int
	var assistantTokens int
	var artifacts []string
	var resultContent string
	var subtype string

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Parse stream-json NDJSON format
		// Note: Claude API usage includes cache_read_input_tokens and
		// cache_creation_input_tokens which represent cached prompt tokens.
		// For result events (cumulative across all turns), we exclude
		// cache_read_input_tokens because it represents the same cached
		// context being re-read on each turn — that content is already
		// counted once via cache_creation_input_tokens. Including it
		// inflates totals enormously for multi-turn conversations.
		var obj struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
			Result  string `json:"result"`
			Usage   struct {
				InputTokens              int `json:"input_tokens"`
				OutputTokens             int `json:"output_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
			Message struct {
				Usage struct {
					InputTokens              int `json:"input_tokens"`
					OutputTokens             int `json:"output_tokens"`
					CacheReadInputTokens     int `json:"cache_read_input_tokens"`
					CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}

		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		switch obj.Type {
		case "result":
			// "result" type carries cumulative usage across all conversation turns.
			// Exclude cache_read_input_tokens: it's the same cached context re-read
			// on each turn (already counted once in cache_creation_input_tokens).
			resultTokens = obj.Usage.InputTokens + obj.Usage.OutputTokens +
				obj.Usage.CacheCreationInputTokens
			resultTokensIn = obj.Usage.InputTokens + obj.Usage.CacheCreationInputTokens
			resultTokensOut = obj.Usage.OutputTokens
			resultContent = obj.Result
			subtype = obj.Subtype
		case "assistant":
			// Take the last assistant event's usage (not sum), since each turn's
			// input_tokens already includes the full conversation history.
			u := obj.Message.Usage
			assistantTokens = u.InputTokens + u.OutputTokens +
				u.CacheReadInputTokens + u.CacheCreationInputTokens
		}
	}

	// Prefer result-level usage (cumulative total from Claude Code),
	// fall back to accumulated assistant-event tokens, then byte estimate
	tokens := resultTokens
	if tokens == 0 {
		tokens = assistantTokens
	}
	if tokens == 0 {
		tokens = len(data) / 4
	}

	// Try to extract JSON from markdown code blocks if result looks like markdown
	if strings.Contains(resultContent, "```json") {
		if extracted := ExtractJSONFromMarkdown(resultContent); extracted != "" {
			resultContent = extracted
		}
	}

	return parseOutputResult{
		Tokens:        tokens,
		TokensIn:      resultTokensIn,
		TokensOut:     resultTokensOut,
		Artifacts:     artifacts,
		ResultContent: resultContent,
		Subtype:       subtype,
	}
}

// parseStreamLine parses a single NDJSON line from Claude Code's stream-json output
// and converts it to a StreamEvent. Returns (event, true) if the line produced a
// meaningful event, or (zero, false) if it should be skipped.
func parseStreamLine(line []byte) (StreamEvent, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		return StreamEvent{}, false
	}

	var eventType string
	if raw, ok := obj["type"]; ok {
		_ = json.Unmarshal(raw, &eventType)
	}

	switch eventType {
	case "system":
		return StreamEvent{Type: "system"}, true

	case "assistant":
		return parseAssistantEvent(obj)

	case "tool_result":
		return StreamEvent{Type: "tool_result"}, false // skip, tool_use already reported

	case "result":
		var usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		}
		if raw, ok := obj["usage"]; ok {
			_ = json.Unmarshal(raw, &usage)
		}
		var subtype string
		if raw, ok := obj["subtype"]; ok {
			_ = json.Unmarshal(raw, &subtype)
		}
		return StreamEvent{
			Type:      "result",
			TokensIn:  usage.InputTokens + usage.CacheCreationInputTokens,
			TokensOut: usage.OutputTokens,
			Subtype:   subtype,
		}, true

	default:
		return StreamEvent{}, false
	}
}

// parseAssistantEvent extracts tool_use and text events from assistant messages.
func parseAssistantEvent(obj map[string]json.RawMessage) (StreamEvent, bool) {
	var msg struct {
		Message struct {
			Content []struct {
				Type  string          `json:"type"`
				Name  string          `json:"name,omitempty"`
				Text  string          `json:"text,omitempty"`
				Input json.RawMessage `json:"input,omitempty"`
			} `json:"content"`
			Usage struct {
				InputTokens              int `json:"input_tokens"`
				OutputTokens             int `json:"output_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}

	if err := json.Unmarshal(flattenToMessage(obj), &msg); err != nil {
		return StreamEvent{}, false
	}

	u := msg.Message.Usage
	totalIn := u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
	for _, block := range msg.Message.Content {
		switch block.Type {
		case "tool_use":
			target := extractToolTarget(block.Name, block.Input)
			return StreamEvent{
				Type:      "tool_use",
				ToolName:  block.Name,
				ToolInput: target,
				TokensIn:  totalIn,
				TokensOut: u.OutputTokens,
			}, true
		case "text":
			if block.Text == "" {
				continue
			}
			// Preserve enough text for display layer to truncate to terminal width
			text := block.Text
			if len(text) > 200 {
				text = text[:200]
			}
			return StreamEvent{
				Type:    "text",
				Content: text,
			}, true
		}
	}

	return StreamEvent{}, false
}

// flattenToMessage wraps the raw JSON map back into bytes for structured parsing.
func flattenToMessage(obj map[string]json.RawMessage) []byte {
	data, _ := json.Marshal(obj)
	return data
}

// extractToolTarget pulls the most relevant input field for display.
func extractToolTarget(toolName string, input json.RawMessage) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return ""
	}

	// Extract the most relevant field per tool
	switch toolName {
	case "Read":
		return jsonString(fields["file_path"])
	case "Write":
		return jsonString(fields["file_path"])
	case "Edit":
		return jsonString(fields["file_path"])
	case "Glob":
		return jsonString(fields["pattern"])
	case "Grep":
		return jsonString(fields["pattern"])
	case "Bash":
		cmd := jsonString(fields["command"])
		if len(cmd) > 200 {
			cmd = cmd[:200] + "..."
		}
		return cmd
	case "Task":
		return jsonString(fields["description"])
	case "WebFetch":
		return jsonString(fields["url"])
	case "WebSearch":
		return jsonString(fields["query"])
	case "NotebookEdit":
		return jsonString(fields["notebook_path"])
	case "TodoWrite":
		return extractTodoSummary(fields["todos"])
	default:
		// Generic heuristic: check common field names in priority order
		for _, field := range []string{"file_path", "url", "pattern", "command", "query", "notebook_path"} {
			if val := jsonString(fields[field]); val != "" {
				return val
			}
		}
		return ""
	}
}

// jsonString extracts a string from a json.RawMessage, stripping quotes.
func jsonString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return strings.Trim(string(raw), "\"")
	}
	return s
}

// extractTodoSummary returns a display string for TodoWrite showing the in-progress task
// or a count summary. Input is the raw "todos" JSON array field.
func extractTodoSummary(raw json.RawMessage) string {
	var todos []struct {
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(raw, &todos); err != nil || len(todos) == 0 {
		return ""
	}

	// Show the in-progress task content if there is one
	for _, t := range todos {
		if t.Status == "in_progress" {
			return t.Content
		}
	}

	// Otherwise show counts
	var done, total int
	for _, t := range todos {
		total++
		if t.Status == "completed" {
			done++
		}
	}
	return fmt.Sprintf("%d/%d tasks", done, total)
}

// ExtractJSONFromMarkdown extracts JSON content from markdown code blocks.
// Returns the extracted JSON or empty string if not found.
// Exported for testing without Claude dependency.
func ExtractJSONFromMarkdown(content string) string {
	// Look for ```json ... ``` blocks
	start := strings.Index(content, "```json")
	if start == -1 {
		return ""
	}
	start += len("```json")

	// Skip any whitespace/newline after ```json
	for start < len(content) && (content[start] == '\n' || content[start] == '\r' || content[start] == ' ') {
		start++
	}

	end := strings.Index(content[start:], "```")
	if end == -1 {
		return ""
	}

	jsonStr := strings.TrimSpace(content[start : start+end])

	// Validate it's actually JSON
	var js json.RawMessage
	if json.Unmarshal([]byte(jsonStr), &js) != nil {
		return ""
	}

	return jsonStr
}

// shelljoinArgs formats command arguments for debug logging, quoting any
// argument that contains shell metacharacters or whitespace so the logged
// command line is copy-pasteable and not misleading.
func shelljoinArgs(args []string) string {
	var parts []string
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n|&;$`\\!(){}[]<>*?~#'\"") {
			// Single-quote the argument, escaping interior single quotes
			escaped := strings.ReplaceAll(arg, "'", `'\''`)
			parts = append(parts, "'"+escaped+"'")
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}

// maxConcurrentAgentsCap is the hard upper limit for concurrent sub-agents.
// Claude Code has a practical limit of ~10 subagents.
const maxConcurrentAgentsCap = 10

// buildConcurrencyHint returns a CLAUDE.md section telling the persona how many
// concurrent sub-agents it may spawn. Returns "" when n <= 1 (default behavior).
func buildConcurrencyHint(n int) string {
	if n <= 1 {
		return ""
	}
	if n > maxConcurrentAgentsCap {
		n = maxConcurrentAgentsCap
	}
	return fmt.Sprintf("\n\n## Agent Concurrency\n\nYou may spawn up to %d concurrent sub-agents or workers for this step.\n", n)
}

// buildSkillSection generates a CLAUDE.md section listing available skills.
// Returns "" when no skills are provided.
func buildSkillSection(skills []SkillRef) string {
	if len(skills) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n---\n\n## Available Skills\n\n")
	b.WriteString("The following skills are available in this workspace:\n\n")
	for _, s := range skills {
		fmt.Fprintf(&b, "- **%s**: %s (see `.agents/skills/%s/SKILL.md`)\n", s.Name, s.Description, s.Name)
	}
	return b.String()
}

// buildRestrictionSection generates the restriction directives for CLAUDE.md
// derived from the manifest permissions in AdapterRunConfig.
func buildRestrictionSection(cfg AdapterRunConfig) string {
	hasDeny := len(cfg.DenyTools) > 0
	hasAllow := len(cfg.AllowedTools) > 0
	hasDomains := len(cfg.AllowedDomains) > 0

	if !hasDeny && !hasAllow && !hasDomains {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n---\n\n## Restrictions\n\n")
	b.WriteString("The following restrictions are enforced by the pipeline orchestrator.\n\n")

	if hasDeny {
		b.WriteString("### Denied Tools\n\n")
		for _, deny := range cfg.DenyTools {
			fmt.Fprintf(&b, "- `%s`\n", deny)
		}
		b.WriteString("\n")
	}

	if hasAllow {
		b.WriteString("### Allowed Tools\n\n")
		b.WriteString("You may ONLY use the following tools:\n\n")
		for _, tool := range cfg.AllowedTools {
			fmt.Fprintf(&b, "- `%s`\n", tool)
		}
		b.WriteString("\n")
	}

	if hasDomains {
		b.WriteString("### Network Access\n\n")
		b.WriteString("Network requests are restricted to:\n\n")
		for _, domain := range cfg.AllowedDomains {
			fmt.Fprintf(&b, "- `%s`\n", domain)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// PersonaSpec holds the subset of persona configuration needed by the agent
// compiler. It is intentionally decoupled from manifest.Persona to avoid an
// import cycle (adapter → manifest → adapter).
//
// Callers that hold a manifest.Persona should map it with PersonaSpecFromManifest
// (defined in the commands package where the manifest import is already present)
// or build the struct directly.
type PersonaSpec struct {
	// Model is the Claude model identifier (e.g. "claude-opus-4") or tier alias
	// (cheapest, balanced, strongest, resolved before this point by the executor).
	// Leave empty to omit the frontmatter field and inherit the CLI default.
	Model string

	// AllowedTools is the list of tool names the agent may use.
	AllowedTools []string

	// DenyTools is the list of tool patterns the agent must not use.
	DenyTools []string
}

// PersonaToAgentMarkdown compiles a PersonaSpec into a Claude Code agent .md
// file with YAML frontmatter. The generated file can be passed directly to
// `claude --agent <path>` to run the persona in agent mode.
//
// The frontmatter sets model, tools, disallowedTools, and permissionMode so
// the agent is fully self-contained — no separate settings.json needed.
//
// The body is assembled from five layers (matching the runtime CLAUDE.md):
//  1. baseProtocol — the shared Wave agent protocol preamble
//  2. ontologySection — the project domain context (telos, invariants, conventions)
//  3. systemPrompt — the persona's role/responsibilities/constraints text
//  4. contractSection — the auto-generated contract compliance section
//  5. restrictions — the denied/allowed tools and network domain section
func PersonaToAgentMarkdown(persona PersonaSpec, baseProtocol, ontologySection, systemPrompt, contractSection, restrictions string) string {
	var b strings.Builder

	// --- YAML frontmatter ---
	b.WriteString("---\n")

	if persona.Model != "" {
		b.WriteString("model: ")
		b.WriteString(persona.Model)
		b.WriteString("\n")
	}

	if len(persona.AllowedTools) > 0 {
		b.WriteString("tools:\n")
		for _, tool := range persona.AllowedTools {
			b.WriteString("  - ")
			b.WriteString(tool)
			b.WriteString("\n")
		}
	}

	if len(persona.DenyTools) > 0 {
		b.WriteString("disallowedTools:\n")
		for _, tool := range persona.DenyTools {
			b.WriteString("  - ")
			b.WriteString(tool)
			b.WriteString("\n")
		}
	}

	b.WriteString("permissionMode: bypassPermissions\n")
	b.WriteString("---\n")

	// --- Body sections ---
	if baseProtocol != "" {
		b.WriteString(baseProtocol)
		b.WriteString("\n\n---\n\n")
	}

	if ontologySection != "" {
		b.WriteString(ontologySection)
		b.WriteString("\n\n---\n\n")
	}

	if systemPrompt != "" {
		b.WriteString(systemPrompt)
	}

	if contractSection != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString(contractSection)
	}

	if restrictions != "" {
		b.WriteString(restrictions)
	}

	return b.String()
}
