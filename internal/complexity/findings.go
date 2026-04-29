package complexity

import (
	"fmt"
	"time"
)

// Finding maps a per-function complexity breach to the shared-findings
// JSON schema used by audit pipelines (see contracts/shared-findings.schema.json).
type Finding struct {
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Package        string `json:"package,omitempty"`
	File           string `json:"file,omitempty"`
	Line           int    `json:"line,omitempty"`
	Item           string `json:"item,omitempty"`
	Description    string `json:"description,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// FindingsDocument is the shared-findings root object.
type FindingsDocument struct {
	Findings  []Finding `json:"findings"`
	Summary   string    `json:"summary,omitempty"`
	ScanType  string    `json:"scan_type,omitempty"`
	ScannedAt string    `json:"scanned_at,omitempty"`
}

// ToSharedFindings maps over-threshold function scores to shared-findings
// objects. Functions exceeding the fail threshold get severity "high"; those
// in the warn band get severity "medium". Functions below the warn threshold
// produce no finding.
func ToSharedFindings(report Report, opts Options) FindingsDocument {
	opts = opts.withDefaults()
	out := FindingsDocument{
		ScanType:  "complexity",
		ScannedAt: report.ScannedAt.Format(time.RFC3339),
	}
	if out.ScannedAt == "0001-01-01T00:00:00Z" {
		out.ScannedAt = time.Now().UTC().Format(time.RFC3339)
	}
	var failCount, warnCount int
	for _, s := range report.Scores {
		if f, ok := cyclomaticFinding(s, opts); ok {
			if f.Severity == "high" {
				failCount++
			} else {
				warnCount++
			}
			out.Findings = append(out.Findings, f)
		}
		if f, ok := cognitiveFinding(s, opts); ok {
			if f.Severity == "high" {
				failCount++
			} else {
				warnCount++
			}
			out.Findings = append(out.Findings, f)
		}
	}
	out.Summary = fmt.Sprintf(
		"%d function(s) scanned, %d breach (high), %d warn (medium)",
		len(report.Scores), failCount, warnCount,
	)
	return out
}

func cyclomaticFinding(s FunctionScore, opts Options) (Finding, bool) {
	if s.Cyclomatic <= opts.WarnCyclomatic {
		return Finding{}, false
	}
	severity := "medium"
	threshold := opts.WarnCyclomatic
	if s.Cyclomatic > opts.MaxCyclomatic {
		severity = "high"
		threshold = opts.MaxCyclomatic
	}
	return Finding{
		Type:           "complexity",
		Severity:       severity,
		Package:        s.Package,
		File:           s.File,
		Line:           s.Line,
		Item:           s.Function,
		Description:    fmt.Sprintf("cyclomatic complexity %d exceeds threshold %d", s.Cyclomatic, threshold),
		Evidence:       fmt.Sprintf("cyclomatic=%d cognitive=%d", s.Cyclomatic, s.Cognitive),
		Recommendation: "refactor",
	}, true
}

func cognitiveFinding(s FunctionScore, opts Options) (Finding, bool) {
	if s.Cognitive <= opts.WarnCognitive {
		return Finding{}, false
	}
	severity := "medium"
	threshold := opts.WarnCognitive
	if s.Cognitive > opts.MaxCognitive {
		severity = "high"
		threshold = opts.MaxCognitive
	}
	return Finding{
		Type:           "complexity",
		Severity:       severity,
		Package:        s.Package,
		File:           s.File,
		Line:           s.Line,
		Item:           s.Function,
		Description:    fmt.Sprintf("cognitive complexity %d exceeds threshold %d", s.Cognitive, threshold),
		Evidence:       fmt.Sprintf("cyclomatic=%d cognitive=%d", s.Cyclomatic, s.Cognitive),
		Recommendation: "refactor",
	}, true
}

// HasBreach returns true when any finding has severity "high" — i.e., any
// function exceeded a fail threshold.
func (d FindingsDocument) HasBreach() bool {
	for _, f := range d.Findings {
		if f.Severity == "high" {
			return true
		}
	}
	return false
}
