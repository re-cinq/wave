package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AuditLogger interface {
	LogToolCall(pipelineID, stepID, tool, args string) error
	LogFileOp(pipelineID, stepID, op, path string) error
	LogStepStart(pipelineID, stepID, persona string, injectedArtifacts []string) error
	LogStepStartWithAdapter(pipelineID, stepID, persona, adapter, model string, injectedArtifacts []string) error
	LogStepEnd(pipelineID, stepID, status string, duration time.Duration, exitCode int, outputBytes int, tokensUsed int, errMsg string) error
	LogContractResult(pipelineID, stepID, contractType, result string) error
	LogOntologyInject(pipelineID, stepID string, contexts []string, invariantCount int) error
	LogOntologyLineage(pipelineID, stepID, contextName, stepStatus string, invariantCount int) error
	LogOntologyWarn(pipelineID, stepID string, undefinedContexts []string) error
	Close() error
}

type TraceLogger struct {
	traceDir  string
	credRegex *regexp.Regexp
	file      *os.File
}

var credentialPatterns = []string{
	`API[_-]?KEY`,
	`TOKEN`,
	`SECRET`,
	`PASSWORD`,
	`CREDENTIAL`,
	`AUTH`,
	`PRIVATE[_-]?KEY`,
	`ACCESS[_-]?KEY`,
}

func NewTraceLogger() (*TraceLogger, error) {
	return NewTraceLoggerWithDir(".agents/traces")
}

func NewTraceLoggerWithDir(traceDir string) (*TraceLogger, error) {
	pattern := `(?i)(` + strings.Join(credentialPatterns, `|`) + `)[=:]?\s*[\w\-]+`
	credRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("20060102-150405")
	tracePath := filepath.Join(traceDir, "trace-"+timestamp+".log")
	file, err := os.OpenFile(tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &TraceLogger{
		traceDir:  traceDir,
		credRegex: credRegex,
		file:      file,
	}, nil
}

func (l *TraceLogger) scrub(text string) string {
	return l.credRegex.ReplaceAllString(text, "[REDACTED]")
}

func (l *TraceLogger) LogToolCall(pipelineID, stepID, tool, args string) error {
	scrubbedArgs := l.scrub(args)
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := timestamp + " [TOOL] pipeline=" + pipelineID + " step=" + stepID + " tool=" + tool + " args=" + scrubbedArgs + "\n"
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogFileOp(pipelineID, stepID, op, path string) error {
	scrubbedPath := l.scrub(path)
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := timestamp + " [FILE] pipeline=" + pipelineID + " step=" + stepID + " op=" + op + " path=" + scrubbedPath + "\n"
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogStepStart(pipelineID, stepID, persona string, injectedArtifacts []string) error {
	return l.LogStepStartWithAdapter(pipelineID, stepID, persona, "", "", injectedArtifacts)
}

func (l *TraceLogger) LogStepStartWithAdapter(pipelineID, stepID, persona, adapter, model string, injectedArtifacts []string) error {
	scrubbedPersona := l.scrub(persona)
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := timestamp + " [STEP_START] pipeline=" + pipelineID + " step=" + stepID + " persona=" + scrubbedPersona
	if adapter != "" {
		line += " adapter=" + adapter
	}
	if model != "" {
		line += " model=" + model
	}
	if len(injectedArtifacts) > 0 {
		scrubbed := make([]string, len(injectedArtifacts))
		for i, a := range injectedArtifacts {
			scrubbed[i] = l.scrub(a)
		}
		line += " artifacts=" + strings.Join(scrubbed, ",")
	}
	line += "\n"
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogStepEnd(pipelineID, stepID, status string, duration time.Duration, exitCode int, outputBytes int, tokensUsed int, errMsg string) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s [STEP_END] pipeline=%s step=%s status=%s duration=%s exit_code=%d output_bytes=%d tokens_used=%d",
		timestamp, pipelineID, stepID, status, formatDuration(duration), exitCode, outputBytes, tokensUsed)
	if errMsg != "" {
		line += fmt.Sprintf(" error=%q", l.scrub(errMsg))
	}
	line += "\n"
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogContractResult(pipelineID, stepID, contractType, result string) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := timestamp + " [CONTRACT] pipeline=" + pipelineID + " step=" + stepID + " type=" + contractType + " result=" + result + "\n"
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogOntologyInject(pipelineID, stepID string, contexts []string, invariantCount int) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s [ONTOLOGY_INJECT] pipeline=%s step=%s contexts=[%s] invariants=%d\n",
		timestamp, pipelineID, stepID, strings.Join(contexts, ","), invariantCount)
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogOntologyLineage(pipelineID, stepID, contextName, stepStatus string, invariantCount int) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s [ONTOLOGY_LINEAGE] pipeline=%s step=%s context=%s status=%s invariants=%d\n",
		timestamp, pipelineID, stepID, contextName, stepStatus, invariantCount)
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) LogOntologyWarn(pipelineID, stepID string, undefinedContexts []string) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s [ONTOLOGY_WARN] pipeline=%s step=%s undefined_contexts=[%s]\n",
		timestamp, pipelineID, stepID, strings.Join(undefinedContexts, ","))
	_, err := l.file.WriteString(line)
	return err
}

func (l *TraceLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// formatDuration formats a duration as seconds with millisecond precision (e.g. "7.523s").
func formatDuration(d time.Duration) string {
	secs := d.Seconds()
	return fmt.Sprintf("%.3fs", secs)
}
