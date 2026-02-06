# Implementation Plan: Issue Research Pipeline

## Overview

This plan details the implementation of the issue-research pipeline, including new personas, contracts, and the pipeline definition.

## Implementation Phases

### Phase 1: Personas

#### 1.1 Create researcher persona
**File**: `.wave/personas/researcher.md`

Web research specialist with:
- WebSearch and WebFetch for web research
- Read for artifacts
- Write(output/*) for results
- No Bash (read-only)
- No Edit (read-only)

#### 1.2 Create github-commenter persona
**File**: `.wave/personas/github-commenter.md`

GitHub comment specialist with:
- Bash(gh issue comment*) for posting
- Bash(gh --version) for verification
- Read for artifacts
- Write(output/*) for results
- No issue editing or PR operations

#### 1.3 Register personas in wave.yaml
Add persona definitions with permissions to main manifest.

### Phase 2: Contracts

All contracts in `.wave/contracts/`:

#### 2.1 issue-content.schema.json
- issue_number, title, body, author, labels, url, repository, state

#### 2.2 research-topics.schema.json
- issue_reference, topics array with id, title, questions, keywords, priority, category

#### 2.3 research-findings.schema.json
- findings_by_topic with findings, sources, confidence_level, gaps

#### 2.4 research-report.schema.json
- executive_summary, detailed_findings, recommendations, sources, markdown_content

#### 2.5 comment-result.schema.json
- success, issue_reference, comment (url, id), error, timestamp

### Phase 3: Pipeline

**File**: `.wave/pipelines/issue-research.yaml`

5-step pipeline:

1. **fetch-issue** (github-analyst)
   - Input: CLI `owner/repo issue_number`
   - Output: issue-content.json
   - Contract: issue-content.schema.json

2. **analyze-topics** (researcher)
   - Input: issue-content artifact
   - Output: research-topics.json
   - Contract: research-topics.schema.json

3. **research-topics** (researcher)
   - Input: issue-content + research-topics artifacts
   - Output: research-findings.json
   - Contract: research-findings.schema.json

4. **synthesize-report** (summarizer)
   - Input: issue-content + research-findings artifacts
   - Output: research-report.json
   - Contract: research-report.schema.json

5. **post-comment** (github-commenter)
   - Input: issue-content + research-report artifacts
   - Output: comment-result.json
   - Contract: comment-result.schema.json

## File Manifest

### New Files
| File | Type | Purpose |
|------|------|---------|
| `.wave/personas/researcher.md` | Persona | Web research |
| `.wave/personas/github-commenter.md` | Persona | Post comments |
| `.wave/contracts/issue-content.schema.json` | Contract | Issue data |
| `.wave/contracts/research-topics.schema.json` | Contract | Topics |
| `.wave/contracts/research-findings.schema.json` | Contract | Findings |
| `.wave/contracts/research-report.schema.json` | Contract | Report |
| `.wave/contracts/comment-result.schema.json` | Contract | Result |
| `.wave/pipelines/issue-research.yaml` | Pipeline | Main pipeline |

### Modified Files
| File | Change |
|------|--------|
| `wave.yaml` | Add researcher and github-commenter personas |

## Testing Strategy

1. **Unit Tests**: Verify contract schemas with valid/invalid data
2. **Integration Test**: Run pipeline against test issue
3. **Manual Test**: Run against re-cinq/CFOAgent#112

## Usage

```bash
# Run the pipeline
wave run issue-research "re-cinq/CFOAgent 112"

# With debug output
wave run issue-research "re-cinq/CFOAgent 112" --debug
```

## Rollback Plan

If issues arise:
1. Remove personas from wave.yaml
2. Delete new persona files
3. Delete new contract files
4. Delete pipeline file
