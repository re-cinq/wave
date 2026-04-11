package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// ValidationSeverity indicates the importance of a validation finding.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
)

// ValidationFinding represents a single issue found during dry-run validation.
type ValidationFinding struct {
	Severity ValidationSeverity
	StepID   string
	Field    string
	Message  string
}

func (f ValidationFinding) String() string {
	if f.StepID != "" {
		if f.Field != "" {
			return fmt.Sprintf("[%s] step %q (%s): %s", f.Severity, f.StepID, f.Field, f.Message)
		}
		return fmt.Sprintf("[%s] step %q: %s", f.Severity, f.StepID, f.Message)
	}
	return fmt.Sprintf("[%s] %s", f.Severity, f.Message)
}

// DryRunReport collects all findings from a dry-run validation pass.
type DryRunReport struct {
	PipelineName string
	Findings     []ValidationFinding
}

// HasErrors returns true if any findings have error severity.
func (r *DryRunReport) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of error-severity findings.
func (r *DryRunReport) ErrorCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			n++
		}
	}
	return n
}

// WarningCount returns the number of warning-severity findings.
func (r *DryRunReport) WarningCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			n++
		}
	}
	return n
}

// Format renders the report as a human-readable string.
func (r *DryRunReport) Format() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dry-run validation: pipeline %q\n", r.PipelineName))
	if len(r.Findings) == 0 {
		sb.WriteString("  No issues found.\n")
		return sb.String()
	}
	for _, f := range r.Findings {
		sb.WriteString("  ")
		sb.WriteString(f.String())
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\n  %d error(s), %d warning(s)\n", r.ErrorCount(), r.WarningCount()))
	return sb.String()
}

// DryRunValidator walks a pipeline's DAG and validates the composition without
// executing anything. It checks:
//   - DAG structure (cycles, unknown dependencies)
//   - Artifact references: step X injects output from step Y which exists
//   - Gate configurations: type is valid, required fields present
//   - Iterate/branch/loop configs: referenced pipelines exist, expressions syntactically valid
//   - Persona references: persona exists in manifest
//   - Contract schema paths: schema files exist on disk
type DryRunValidator struct {
	pipelinesDir string
}

// NewDryRunValidator creates a DryRunValidator that looks for sub-pipelines in
// the given directory (e.g. ".wave/pipelines").
func NewDryRunValidator(pipelinesDir string) *DryRunValidator {
	return &DryRunValidator{pipelinesDir: pipelinesDir}
}

// Validate runs all checks against the pipeline and returns a DryRunReport.
// The manifest is used to resolve persona and adapter references; pass nil to
// skip those checks.
func (v *DryRunValidator) Validate(p *Pipeline, m *manifest.Manifest) *DryRunReport {
	report := &DryRunReport{PipelineName: p.Metadata.Name}

	// 1. Structural validation (DAG or graph mode).
	dag := &DAGValidator{}
	if IsGraphPipeline(p) {
		if err := dag.ValidateGraph(p); err != nil {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("graph validation failed: %v", err),
			})
			return report
		}
	} else {
		if err := dag.ValidateDAG(p); err != nil {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("DAG validation failed: %v", err),
			})
			return report
		}
	}

	// Build a map of step IDs → their output artifact names for cross-step
	// artifact reference resolution.
	stepArtifacts := buildStepArtifactMap(p)

	// 2. Per-step checks.
	for i := range p.Steps {
		step := &p.Steps[i]
		v.validateStep(step, p, m, stepArtifacts, report)
	}

	// 3. Composition-level template checks (if applicable).
	if isCompositionPipeline(p) {
		for _, errMsg := range ValidateCompositionTemplates(p) {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityWarning,
				Message:  errMsg,
			})
		}
	}

	return report
}

// validateStep runs all per-step checks and appends findings to report.
func (v *DryRunValidator) validateStep(
	step *Step,
	p *Pipeline,
	m *manifest.Manifest,
	stepArtifacts map[string]map[string]bool,
	report *DryRunReport,
) {
	// Graph-mode step type validation.
	if step.Type == StepTypeConditional {
		// Conditional steps don't need persona or exec config
		v.validateEdges(step, p, report)
		v.validateInjectArtifacts(step, p, stepArtifacts, report)
		return
	}
	if step.Type == StepTypeCommand {
		// Command steps don't need persona
		v.validateCommandStep(step, report)
		v.validateEdges(step, p, report)
		v.validateInjectArtifacts(step, p, stepArtifacts, report)
		return
	}
	// Validate edges on regular steps too (they may have forward edges)
	if len(step.Edges) > 0 {
		v.validateEdges(step, p, report)
	}

	// Persona reference (only for non-composition steps).
	if !step.IsCompositionStep() {
		v.validatePersonaRef(step, m, report)
		v.validateExecConfig(step, report)
	}

	// Step-level adapter override reference.
	if step.Adapter != "" && m != nil {
		if m.GetAdapter(step.Adapter) == nil {
			var adapterNames []string
			for name := range m.Adapters {
				adapterNames = append(adapterNames, name)
			}
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    "adapter",
				Message:  fmt.Sprintf("adapter %q is not defined in manifest adapters (available: %s)", step.Adapter, strings.Join(adapterNames, ", ")),
			})
		}
	}

	// Inject artifact references.
	v.validateInjectArtifacts(step, p, stepArtifacts, report)

	// Handover contract.
	v.validateContract(step, report)

	// Composition primitives.
	if step.Iterate != nil {
		v.validateIterate(step, report)
	}
	if step.Branch != nil {
		v.validateBranch(step, report)
	}
	if step.Gate != nil {
		v.validateGate(step, report)
	}
	if step.Loop != nil {
		v.validateLoop(step, report)
	}
	if step.Aggregate != nil {
		v.validateAggregate(step, report)
	}
	if step.SubPipeline != "" && step.Iterate == nil && step.Branch == nil && step.Loop == nil {
		v.validateSubPipeline(step, step.SubPipeline, "pipeline", report)
	}

	// Sub-pipeline config validation.
	if step.Config != nil {
		v.validateSubPipelineConfig(step, report)
	}
}

// --- persona ---

func (v *DryRunValidator) validatePersonaRef(step *Step, m *manifest.Manifest, report *DryRunReport) {
	if step.Persona == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "persona",
			Message:  "persona is required for non-composition steps",
		})
		return
	}
	if m == nil {
		return
	}
	persona := m.GetPersona(step.Persona)
	if persona == nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "persona",
			Message:  fmt.Sprintf("persona %q not found in manifest", step.Persona),
		})
		return
	}
	if m.GetAdapter(persona.Adapter) == nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "persona",
			Message:  fmt.Sprintf("persona %q references unknown adapter %q", step.Persona, persona.Adapter),
		})
	}
}

// --- exec config ---

func (v *DryRunValidator) validateExecConfig(step *Step, report *DryRunReport) {
	if step.Exec.Type == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "exec.type",
			Message:  "exec.type is required for non-composition steps",
		})
		return
	}
	switch step.Exec.Type {
	case "prompt", "command", "slash_command":
		// valid
	default:
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "exec.type",
			Message:  fmt.Sprintf("unknown exec.type %q (valid: prompt, command, slash_command)", step.Exec.Type),
		})
	}

	if step.Exec.Type == "prompt" && step.Exec.Source == "" && step.Exec.SourcePath == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "exec",
			Message:  "exec.type=prompt requires either exec.source or exec.source_path",
		})
	}
	if step.Exec.Type == "slash_command" && step.Exec.Command == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "exec.command",
			Message:  "exec.type=slash_command requires exec.command",
		})
	}
	if step.Exec.Type == "command" && step.Exec.Command == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "exec.command",
			Message:  "exec.type=command requires exec.command",
		})
	}
}

// --- inject artifacts ---

func (v *DryRunValidator) validateInjectArtifacts(
	step *Step,
	_ *Pipeline,
	stepArtifacts map[string]map[string]bool,
	report *DryRunReport,
) {
	for i, ref := range step.Memory.InjectArtifacts {
		field := fmt.Sprintf("memory.inject_artifacts[%d]", i)

		// Step vs pipeline mutual exclusion is already checked by DAGValidator.
		// Here we validate the semantic references.

		if ref.Pipeline != "" {
			// Cross-pipeline reference — we cannot validate at static analysis time
			// because the other pipeline's outputs are runtime-determined.
			continue
		}

		if ref.Step == "" {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    field,
				Message:  "inject_artifacts entry must have either step or pipeline set",
			})
			continue
		}

		// Verify the referenced step exists.
		srcArtifacts, srcStepExists := stepArtifacts[ref.Step]
		if !srcStepExists {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    field,
				Message:  fmt.Sprintf("references step %q which does not exist", ref.Step),
			})
			continue
		}

		// Verify the artifact name exists on the source step (if the source step
		// declares output_artifacts — skip if it has none since it may produce
		// outputs dynamically via stdout capture).
		if len(srcArtifacts) > 0 && ref.Artifact != "" && !srcArtifacts[ref.Artifact] {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityWarning,
				StepID:   step.ID,
				Field:    field,
				Message:  fmt.Sprintf("step %q does not declare output artifact %q (it may be produced at runtime)", ref.Step, ref.Artifact),
			})
		}

		// Check that the dependency is listed so the executor orders steps correctly.
		if !hasDependency(step, ref.Step) {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityWarning,
				StepID:   step.ID,
				Field:    field,
				Message:  fmt.Sprintf("injects artifact from step %q but %q is not listed in dependencies — execution order may be wrong", ref.Step, ref.Step),
			})
		}

		// Validate schema path if provided.
		if ref.SchemaPath != "" {
			if _, err := os.Stat(ref.SchemaPath); os.IsNotExist(err) {
				report.Findings = append(report.Findings, ValidationFinding{
					Severity: SeverityError,
					StepID:   step.ID,
					Field:    field + ".schema_path",
					Message:  fmt.Sprintf("schema file %q does not exist", ref.SchemaPath),
				})
			}
		}
	}
}

// --- contract ---

// validContractTypes mirrors the switch in contract.NewValidator.
var validContractTypes = map[string]bool{
	"json_schema":          true,
	"typescript_interface": true,
	"test_suite":           true,
	"markdown_spec":        true,
	"format":               true,
	"non_empty_file":       true,
	"llm_judge":            true,
	"agent_review":         true,
}

func (v *DryRunValidator) validateContract(step *Step, report *DryRunReport) {
	c := step.Handover.Contract
	if c.Type == "" {
		return
	}

	if !validContractTypes[c.Type] {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "handover.contract.type",
			Message:  fmt.Sprintf("unknown contract type %q (valid: %s)", c.Type, validContractTypeList()),
		})
		return
	}

	// json_schema: need either schema or schema_path.
	if c.Type == "json_schema" {
		if c.Schema == "" && c.SchemaPath == "" {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    "handover.contract",
				Message:  "json_schema contract requires either schema (inline) or schema_path",
			})
		} else if c.SchemaPath != "" {
			if _, err := os.Stat(c.SchemaPath); os.IsNotExist(err) {
				report.Findings = append(report.Findings, ValidationFinding{
					Severity: SeverityError,
					StepID:   step.ID,
					Field:    "handover.contract.schema_path",
					Message:  fmt.Sprintf("schema file %q does not exist", c.SchemaPath),
				})
			}
		}
	}

	// test_suite: need a command.
	if c.Type == "test_suite" && c.Command == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "handover.contract.command",
			Message:  "test_suite contract requires command",
		})
	}

	// on_failure validation.
	if c.OnFailure != "" {
		switch c.OnFailure {
		case "fail", "warn", "skip", "continue", "retry", "rework":
			// valid
		default:
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    "handover.contract.on_failure",
				Message:  fmt.Sprintf("unknown on_failure value %q (valid: fail, warn, skip, continue, retry, rework)", c.OnFailure),
			})
		}
	}
}

func validContractTypeList() string {
	types := make([]string, 0, len(validContractTypes))
	for t := range validContractTypes {
		types = append(types, t)
	}
	return strings.Join(types, ", ")
}

// --- gate ---

var validGateTypes = map[string]bool{
	"approval": true,
	"pr_merge": true,
	"ci_pass":  true,
	"timer":    true,
}

func (v *DryRunValidator) validateGate(step *Step, report *DryRunReport) {
	gate := step.Gate
	if gate.Type == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "gate.type",
			Message:  "gate.type is required",
		})
		return
	}
	if !validGateTypes[gate.Type] {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "gate.type",
			Message:  fmt.Sprintf("unknown gate type %q (valid: approval, pr_merge, ci_pass, timer)", gate.Type),
		})
	}
	// Timer gate requires a timeout.
	if gate.Type == "timer" && gate.Timeout == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "gate.timeout",
			Message:  "gate type=timer requires timeout",
		})
	}
}

// --- iterate ---

func (v *DryRunValidator) validateIterate(step *Step, report *DryRunReport) {
	iter := step.Iterate
	if iter.Over == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "iterate.over",
			Message:  "iterate.over is required",
		})
	} else {
		// Light syntactic check: must look like a template expression or simple value.
		v.checkTemplateExpression(step.ID, "iterate.over", iter.Over, report)
	}
	if iter.Mode != "" && iter.Mode != "sequential" && iter.Mode != "parallel" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "iterate.mode",
			Message:  fmt.Sprintf("unknown iterate mode %q (valid: sequential, parallel)", iter.Mode),
		})
	}
	if step.SubPipeline == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "pipeline",
			Message:  "iterate step must specify a pipeline to run per item",
		})
	} else {
		v.validateSubPipeline(step, step.SubPipeline, "pipeline (iterate)", report)
	}
}

// --- branch ---

func (v *DryRunValidator) validateBranch(step *Step, report *DryRunReport) {
	branch := step.Branch
	if branch.On == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "branch.on",
			Message:  "branch.on expression is required",
		})
	} else {
		v.checkTemplateExpression(step.ID, "branch.on", branch.On, report)
	}
	if len(branch.Cases) == 0 {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "branch.cases",
			Message:  "branch must have at least one case",
		})
		return
	}
	for caseVal, pipelineName := range branch.Cases {
		if pipelineName == "skip" {
			continue // "skip" is a reserved no-op value.
		}
		v.validateSubPipeline(step, pipelineName, fmt.Sprintf("branch.cases[%q]", caseVal), report)
	}
}

// --- loop ---

func (v *DryRunValidator) validateLoop(step *Step, report *DryRunReport) {
	loop := step.Loop
	if loop.MaxIterations <= 0 {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "loop.max_iterations",
			Message:  "loop.max_iterations must be > 0",
		})
	}
	if loop.Until != "" {
		v.checkTemplateExpression(step.ID, "loop.until", loop.Until, report)
	}
	if step.SubPipeline != "" {
		v.validateSubPipeline(step, step.SubPipeline, "pipeline (loop)", report)
	}
	for j := range loop.Steps {
		subStep := &loop.Steps[j]
		if subStep.SubPipeline != "" {
			v.validateSubPipeline(subStep, subStep.SubPipeline, "loop.steps[*].pipeline", report)
		}
	}
}

// --- aggregate ---

func (v *DryRunValidator) validateAggregate(step *Step, report *DryRunReport) {
	agg := step.Aggregate
	if agg.From == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "aggregate.from",
			Message:  "aggregate.from is required",
		})
	} else {
		v.checkTemplateExpression(step.ID, "aggregate.from", agg.From, report)
	}
	if agg.Into == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "aggregate.into",
			Message:  "aggregate.into is required",
		})
	}
	if agg.Strategy != "" {
		switch agg.Strategy {
		case "merge_arrays", "concat", "reduce":
			// valid
		default:
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    "aggregate.strategy",
				Message:  fmt.Sprintf("unknown aggregate strategy %q (valid: merge_arrays, concat, reduce)", agg.Strategy),
			})
		}
	}
}

// --- sub-pipeline config ---

func (v *DryRunValidator) validateSubPipelineConfig(step *Step, report *DryRunReport) {
	cfg := step.Config
	if cfg == nil {
		return
	}

	// Config only makes sense on sub-pipeline steps
	if step.SubPipeline == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityWarning,
			StepID:   step.ID,
			Field:    "config",
			Message:  "config is set but step has no pipeline reference — config will be ignored",
		})
	}

	if err := cfg.Validate(); err != nil {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "config",
			Message:  err.Error(),
		})
	}

	// Check stop_condition template syntax
	if cfg.StopCondition != "" {
		v.checkTemplateExpression(step.ID, "config.stop_condition", cfg.StopCondition, report)
	}
}

// --- sub-pipeline existence check ---

func (v *DryRunValidator) validateSubPipeline(step *Step, name, field string, report *DryRunReport) {
	if name == "" {
		return
	}
	candidates := []string{
		v.pipelinesDir + "/" + name + ".yaml",
		v.pipelinesDir + "/" + name,
		name,
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return // Found.
		}
	}
	report.Findings = append(report.Findings, ValidationFinding{
		Severity: SeverityError,
		StepID:   step.ID,
		Field:    field,
		Message:  fmt.Sprintf("sub-pipeline %q not found in %s", name, v.pipelinesDir),
	})
}

// --- template expression syntax check ---

// checkTemplateExpression does a lightweight syntactic check: balanced {{ }}.
func (v *DryRunValidator) checkTemplateExpression(stepID, field, expr string, report *DryRunReport) {
	opens := strings.Count(expr, "{{")
	closes := strings.Count(expr, "}}")
	if opens != closes {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   stepID,
			Field:    field,
			Message:  fmt.Sprintf("template expression has unbalanced braces ({{ count=%d, }} count=%d): %q", opens, closes, expr),
		})
	}
}

// --- edges ---

func (v *DryRunValidator) validateEdges(step *Step, p *Pipeline, report *DryRunReport) {
	stepMap := make(map[string]bool, len(p.Steps))
	for _, s := range p.Steps {
		stepMap[s.ID] = true
	}

	for i, edge := range step.Edges {
		field := fmt.Sprintf("edges[%d]", i)
		if edge.Target == "" {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    field,
				Message:  "edge target is required",
			})
			continue
		}
		if !stepMap[edge.Target] && edge.Target != EdgeTargetComplete && edge.Target != "_fail" {
			report.Findings = append(report.Findings, ValidationFinding{
				Severity: SeverityError,
				StepID:   step.ID,
				Field:    field,
				Message:  fmt.Sprintf("edge target %q does not exist", edge.Target),
			})
		}
		if edge.Condition != "" {
			if _, err := ParseCondition(edge.Condition); err != nil {
				report.Findings = append(report.Findings, ValidationFinding{
					Severity: SeverityError,
					StepID:   step.ID,
					Field:    field + ".condition",
					Message:  fmt.Sprintf("invalid condition: %v", err),
				})
			}
		}
	}

	// Validate max_visits range
	if step.MaxVisits < 0 {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "max_visits",
			Message:  fmt.Sprintf("max_visits must be non-negative (got %d)", step.MaxVisits),
		})
	}
}

// --- command step ---

func (v *DryRunValidator) validateCommandStep(step *Step, report *DryRunReport) {
	if step.Script == "" {
		report.Findings = append(report.Findings, ValidationFinding{
			Severity: SeverityError,
			StepID:   step.ID,
			Field:    "script",
			Message:  "type=command step requires a script field",
		})
	}
}

// --- helpers ---

// buildStepArtifactMap returns a map of stepID → set of declared output artifact names.
func buildStepArtifactMap(p *Pipeline) map[string]map[string]bool {
	m := make(map[string]map[string]bool, len(p.Steps))
	for _, step := range p.Steps {
		arts := make(map[string]bool, len(step.OutputArtifacts))
		for _, a := range step.OutputArtifacts {
			arts[a.Name] = true
		}
		m[step.ID] = arts
	}
	return m
}

// hasDependency checks whether targetID is listed in step.Dependencies.
func hasDependency(step *Step, targetID string) bool {
	for _, d := range step.Dependencies {
		if d == targetID {
			return true
		}
	}
	return false
}

// isCompositionPipeline returns true if any step uses a composition primitive.
func isCompositionPipeline(p *Pipeline) bool {
	for _, step := range p.Steps {
		if step.IsCompositionStep() {
			return true
		}
	}
	return false
}
