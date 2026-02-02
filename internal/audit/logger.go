package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AuditLogger interface {
	LogToolCall(pipelineID, stepID, tool, args string) error
	LogFileOp(pipelineID, stepID, op, path string) error
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
	traceDir := ".wave/traces"
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

func (l *TraceLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
