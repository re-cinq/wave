# Prototype Pipeline Artifact Conventions

This document defines the artifact paths and naming conventions for the prototype-driven development pipeline.

## Artifact Naming Convention

### Phase Output Files

**Spec Phase:**
- `spec.md` - Feature specification document
- `requirements.md` - Extracted requirements (optional)

**Docs Phase:**
- `feature-docs.md` - Comprehensive feature documentation
- `stakeholder-summary.md` - Non-technical summary for stakeholders

**Dummy Phase:**
- `prototype/` - Directory containing working prototype code
- `interfaces.md` - Interface definitions and API documentation

**Implement Phase:**
- `implementation-plan.md` - Detailed implementation roadmap
- `implementation-checklist.md` - Progress tracking checklist

**PR Cycle Phases:**
- `pr-info.json` - Pull request metadata (URL, number, status)
- `review-responses.md` - Responses to review feedback
- `fixes-applied.md` - Summary of fixes implemented

## Workspace Structure

Each phase executes in an isolated workspace following this pattern:
```
.wave/workspaces/prototype-{timestamp}/
├── {step-id}/              # Current step workspace
│   ├── artifacts/          # Injected artifacts from previous steps
│   ├── {output-files}      # Step output files
│   └── workspace/          # Working directory
└── shared/                 # Cross-step shared resources (if needed)
```

## Artifact Injection Mapping

**Docs Phase receives:**
- `artifacts/input-spec.md` ← `spec.md` from spec phase

**Dummy Phase receives:**
- `artifacts/feature-docs.md` ← `feature-docs.md` from docs phase
- `artifacts/spec.md` ← `spec.md` from spec phase

**Implement Phase receives:**
- `artifacts/spec.md` ← `spec.md` from spec phase
- `artifacts/feature-docs.md` ← `feature-docs.md` from docs phase
- `artifacts/prototype/` ← `prototype/` from dummy phase

**PR Phases receive:**
- `artifacts/implementation-plan.md` ← `implementation-plan.md` from implement phase
- Additional artifacts as needed for specific PR operations

## File Type Classifications

- **specification** - Requirements and feature definitions
- **documentation** - Human-readable guides and explanations
- **code** - Executable prototype or implementation code
- **metadata** - Structured data about process state
- **checklist** - Progress tracking and validation lists

## Path Resolution

All artifact paths are resolved relative to the current step's workspace. Output artifacts are written to the workspace root and registered with the pipeline executor for handover to subsequent steps.

## Validation

Contract schemas validate both the presence and structure of required artifacts at each phase boundary. See `.wave/contracts/` for detailed validation rules.