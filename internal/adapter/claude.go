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

type ClaudeSettings struct {
	Model        string            `json:"model"`
	Temperature  float64           `json:"temperature"`
	OutputFormat string            `json:"output_format"`
	Permissions  ClaudePermissions `json:"permissions"`
	Sandbox      *SandboxSettings  `json:"sandbox,omitempty"`
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

type ClaudePermissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny,omitempty"`
}

func (a *ClaudeAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Minute
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	workspacePath := cfg.WorkspacePath
	if workspacePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		workspacePath = wd
	}

	if err := a.prepareWorkspace(workspacePath, cfg); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	args := a.buildArgs(cfg)
	cmd := exec.CommandContext(ctx, a.claudePath, args...)
	cmd.Dir = workspacePath

	if cfg.Debug {
		fmt.Printf("[DEBUG] Claude command: %s %s\n", a.claudePath, strings.Join(args, " "))
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
			killProcessGroup(cmd.Process)
		}
		cmd.Wait()
		return nil, ctx.Err()
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

	tokens, artifacts, resultContent := a.parseOutput(stdoutBuf.Bytes())
	result.TokensUsed = tokens
	result.Artifacts = artifacts

	// Apply output validation and correction
	correctedContent, err := a.validateAndCorrectOutput(resultContent, cfg.OutputFormat)
	if err != nil && cfg.Debug {
		fmt.Printf("[DEBUG] Output validation/correction failed: %v\n", err)
		fmt.Printf("[DEBUG] Using original content\n")
		result.ResultContent = resultContent
	} else {
		result.ResultContent = correctedContent
	}

	if cfg.Debug {
		fmt.Printf("[DEBUG] Claude exit code: %d\n", result.ExitCode)
		fmt.Printf("[DEBUG] Claude tokens used: %d\n", tokens)
		if stderrBuf.Len() > 0 {
			fmt.Printf("[DEBUG] Claude stderr:\n%s\n", stderrBuf.String())
		}
		fmt.Printf("[DEBUG] Claude raw output (%d bytes):\n%s\n", stdoutBuf.Len(), stdoutBuf.String())
		fmt.Printf("[DEBUG] Extracted result content (%d chars):\n%s\n", len(resultContent), resultContent)
	}

	return result, nil
}

func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
	settingsDir := filepath.Join(workspacePath, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Build allowed tools list from config
	allowedTools := cfg.AllowedTools
	if len(allowedTools) == 0 {
		allowedTools = []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"}
	}

	// Generate settings.json for this step's persona
	model := cfg.Model
	if model == "" {
		model = "opus" // Default to opus for best quality
	}
	settings := ClaudeSettings{
		Model:        model,
		Temperature:  cfg.Temperature,
		OutputFormat: "stream-json",
		Permissions: ClaudePermissions{
			Allow: normalizeAllowedTools(allowedTools),
			Deny:  cfg.DenyTools,
		},
	}

	// Add sandbox settings when sandbox is enabled (master switch)
	if cfg.SandboxEnabled {
		settings.Sandbox = &SandboxSettings{
			Enabled:                  true,
			AllowUnsandboxedCommands: false,
			AutoAllowBashIfSandboxed: true,
		}
		if len(cfg.AllowedDomains) > 0 {
			settings.Sandbox.Network = &NetworkSettings{
				AllowedDomains: cfg.AllowedDomains,
			}
		}
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	settingsData, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	// Build CLAUDE.md: persona prompt + manifest-derived restrictions
	claudeMdPath := filepath.Join(workspacePath, "CLAUDE.md")
	var claudeMd strings.Builder

	// 1. Persona system prompt
	if cfg.SystemPrompt != "" {
		claudeMd.WriteString(cfg.SystemPrompt)
	} else {
		personaPath := filepath.Join(".wave", "personas", cfg.Persona+".md")
		if data, err := os.ReadFile(personaPath); err == nil {
			claudeMd.Write(data)
		} else {
			fmt.Fprintf(&claudeMd, "# %s\n\nYou are operating as the %s persona.\n", cfg.Persona, cfg.Persona)
		}
	}

	// 2. Restriction section from manifest
	restrictions := buildRestrictionSection(cfg)
	if restrictions != "" {
		claudeMd.WriteString(restrictions)
	}

	if err := os.WriteFile(claudeMdPath, []byte(claudeMd.String()), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return nil
}

// buildEnvironment constructs a curated environment for the Claude Code subprocess.
// Instead of passing the full host environment, it provides only the base variables
// needed for operation plus explicitly allowed passthrough variables from the manifest.
func (a *ClaudeAdapter) buildEnvironment(cfg AdapterRunConfig) []string {
	// Base environment (always needed)
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"TERM=" + getenvDefault("TERM", "xterm-256color"),
		"TMPDIR=/tmp",
		"DISABLE_TELEMETRY=1",
		"DISABLE_ERROR_REPORTING=1",
		"CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1",
		"DISABLE_BUG_COMMAND=1",
	}

	// Add explicitly allowed env vars from manifest
	for _, key := range cfg.EnvPassthrough {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	// Step-specific env vars (from pipeline config)
	env = append(env, cfg.Env...)
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

	// Set model - default to opus for best quality
	model := cfg.Model
	if model == "" {
		model = "opus"
	}
	args = append(args, "--model", model)

	if len(cfg.AllowedTools) > 0 {
		normalized := normalizeAllowedTools(cfg.AllowedTools)
		args = append(args, "--allowedTools", strings.Join(normalized, ","))
	}

	args = append(args, "--output-format", "stream-json")
	args = append(args, "--verbose")
	// Skip permission prompts — Wave enforces permissions via allowedTools
	args = append(args, "--dangerously-skip-permissions")
	// Note: Claude CLI doesn't support --temperature flag
	// Temperature is set via .claude/settings.json in prepareWorkspace
	// Use --no-session-persistence to avoid saving sessions
	args = append(args, "--no-session-persistence")

	if cfg.Prompt != "" {
		args = append(args, cfg.Prompt)
	}

	return args
}

func (a *ClaudeAdapter) parseOutput(data []byte) (int, []string, string) {
	var tokens int
	var artifacts []string
	var resultContent string

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Parse stream-json NDJSON format
		var obj struct {
			Type   string `json:"type"`
			Result string `json:"result"`
			Usage  struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}

		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		// "result" type carries the final output in stream-json format
		if obj.Type == "result" {
			tokens = obj.Usage.InputTokens + obj.Usage.OutputTokens
			resultContent = obj.Result
		}
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

	return tokens, artifacts, resultContent
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
		json.Unmarshal(raw, &eventType)
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
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}
		if raw, ok := obj["usage"]; ok {
			json.Unmarshal(raw, &usage)
		}
		return StreamEvent{
			Type:      "result",
			TokensIn:  usage.InputTokens,
			TokensOut: usage.OutputTokens,
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
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}

	if err := json.Unmarshal(flattenToMessage(obj), &msg); err != nil {
		return StreamEvent{}, false
	}

	for _, block := range msg.Message.Content {
		switch block.Type {
		case "tool_use":
			target := extractToolTarget(block.Name, block.Input)
			return StreamEvent{
				Type:      "tool_use",
				ToolName:  block.Name,
				ToolInput: target,
				TokensIn:  msg.Message.Usage.InputTokens,
				TokensOut: msg.Message.Usage.OutputTokens,
			}, true
		case "text":
			if block.Text == "" {
				continue
			}
			// Only emit text events for substantial content
			text := block.Text
			if len(text) > 80 {
				text = text[:80]
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
		if len(cmd) > 60 {
			cmd = cmd[:60] + "..."
		}
		return cmd
	case "Task":
		return jsonString(fields["description"])
	default:
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

// validateAndCorrectOutput validates and attempts to fix common output format issues
func (a *ClaudeAdapter) validateAndCorrectOutput(content, outputFormat string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("empty output content")
	}

	// Apply format-specific validation and correction
	switch outputFormat {
	case "json":
		return a.validateAndCorrectJSON(content)
	default:
		// For non-JSON formats, return as-is
		return content, nil
	}
}

// validateAndCorrectJSON validates JSON output and applies automatic corrections
func (a *ClaudeAdapter) validateAndCorrectJSON(content string) (string, error) {
	// First, try to parse the content as-is
	var js json.RawMessage
	if json.Unmarshal([]byte(content), &js) == nil {
		// Already valid JSON
		return content, nil
	}

	// Try extracting JSON from markdown code blocks
	if strings.Contains(content, "```") {
		if extracted := ExtractJSONFromMarkdown(content); extracted != "" {
			if json.Unmarshal([]byte(extracted), &js) == nil {
				return extracted, nil
			}
		}
	}

	// Try basic JSON cleanup
	cleaned := a.cleanJSONContent(content)
	if cleaned != "" {
		if json.Unmarshal([]byte(cleaned), &js) == nil {
			return cleaned, nil
		}
	}

	// If all corrections failed, return original with error
	return content, fmt.Errorf("could not correct malformed JSON output")
}

// cleanJSONContent performs basic JSON cleanup operations
func (a *ClaudeAdapter) cleanJSONContent(content string) string {
	// Remove common non-JSON text patterns
	lines := strings.Split(content, "\n")
	var jsonLines []string
	inJSON := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Skip lines that look like explanatory text
		if strings.HasPrefix(trimmed, "Here") ||
		   strings.HasPrefix(trimmed, "This") ||
		   strings.HasPrefix(trimmed, "The") ||
		   strings.Contains(strings.ToLower(trimmed), "explanation") ||
		   strings.Contains(strings.ToLower(trimmed), "here is") {
			continue
		}

		// Look for JSON start
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			inJSON = true
		}

		if inJSON {
			jsonLines = append(jsonLines, line)
		}

		// Look for JSON end
		if (strings.HasSuffix(trimmed, "}") || strings.HasSuffix(trimmed, "]")) && inJSON {
			// Check if this completes the JSON
			candidate := strings.Join(jsonLines, "\n")
			var js json.RawMessage
			if json.Unmarshal([]byte(candidate), &js) == nil {
				return candidate
			}
		}
	}

	// If we collected JSON lines but validation failed, try the full content
	if len(jsonLines) > 0 {
		candidate := strings.Join(jsonLines, "\n")

		// Try some common fixes
		candidate = strings.TrimSpace(candidate)

		// Remove trailing commas before closing braces/brackets
		candidate = strings.ReplaceAll(candidate, ",}", "}")
		candidate = strings.ReplaceAll(candidate, ",]", "]")

		return candidate
	}

	return content
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

// normalizeAllowedTools converts scoped Write entries to bare Write
// since Claude Code doesn't support Write(path) specifiers.
// It also deduplicates entries.
func normalizeAllowedTools(tools []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, tool := range tools {
		if strings.HasPrefix(tool, "Write(") {
			tool = "Write"
		}
		if !seen[tool] {
			seen[tool] = true
			result = append(result, tool)
		}
	}
	return result
}
