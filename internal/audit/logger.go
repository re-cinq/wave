package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/redact"
)

type AuditLogger interface {
	LogToolCall(pipelineID, stepID, tool, args string) error
	LogFileOp(pipelineID, stepID, op, path string) error
	LogStepStart(pipelineID, stepID, persona string, injectedArtifacts []string) error
	LogStepStartWithAdapter(pipelineID, stepID, persona, adapter, model string, injectedArtifacts []string) error
	LogStepEnd(pipelineID, stepID, status string, duration time.Duration, exitCode int, outputBytes int, tokensUsed int, errMsg string) error
	LogContractResult(pipelineID, stepID, contractType, result string) error
	// LogEvent writes a generic trace line in the form
	//   <timestamp> [KIND] <body>
	// Used by bounded-context services (e.g. ontology) that want to
	// participate in the audit trail without adding domain-specific methods
	// to this interface.
	LogEvent(kind, body string) error
	Close() error
}

type TraceLogger struct {
	traceDir string
	file     *os.File
}

func NewTraceLogger() (*TraceLogger, error) {
	return NewTraceLoggerWithDir(".agents/traces")
}

func NewTraceLoggerWithDir(traceDir string) (*TraceLogger, error) {
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
		traceDir: traceDir,
		file:     file,
	}, nil
}

// scrub redacts credential patterns using the canonical internal/redact
// implementation. The receiver is preserved so existing call sites that
// already hold a *TraceLogger keep working unchanged.
func (l *TraceLogger) scrub(text string) string {
	return redact.Redact(text)
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

// LogEvent writes a generic trace line used by bounded-context services
// (e.g. internal/ontology) that do not own a dedicated method on this type.
// The body is expected to already be formatted as "key=value key=value ..."
// or similar; it is emitted verbatim after scrubbing credentials.
func (l *TraceLogger) LogEvent(kind, body string) error {
	timestamp := time.Now().Format(time.RFC3339Nano)
	scrubbed := l.scrub(body)
	line := timestamp + " [" + kind + "] " + scrubbed + "\n"
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
