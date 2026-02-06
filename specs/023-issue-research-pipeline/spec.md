# Issue Research Pipeline

## Overview

A Wave pipeline that automates research for GitHub issues by analyzing issue content, identifying research topics, performing web searches, and posting synthesized findings as comments.

## Problem Statement

When working on complex GitHub issues, developers often need to research external information:
- Best practices and patterns
- Library comparisons and recommendations
- Similar problems and solutions
- Technical documentation

This research is time-consuming and manual. Wave can automate this process by orchestrating specialized AI personas to research and synthesize information.

## Use Case

**Primary Example**: [re-cinq/CFOAgent#112](https://github.com/re-cinq/CFOAgent/issues/112)

This issue needs research on:
- LLM reliability for financial applications
- Separation of concerns (Financial Analytics vs Data Retrieval vs Calculations)
- Testing strategies for non-deterministic AI systems
- Model evaluation and comparison approaches

## User Stories

### US-1: Research a GitHub Issue
**As a** developer working on a GitHub issue
**I want** Wave to research relevant topics and post findings
**So that** I have actionable information to solve the issue

**Acceptance Criteria:**
- Pipeline accepts `owner/repo issue_number` as input
- Fetches issue content from GitHub API
- Extracts 1-10 research topics from issue
- Performs web searches for each topic
- Synthesizes findings into a coherent report
- Posts report as a comment on the issue

### US-2: Traceable Research
**As a** developer reviewing research findings
**I want** all claims to have source citations
**So that** I can verify information and explore further

**Acceptance Criteria:**
- Every finding includes source URL
- Sources are rated by credibility
- Report includes a complete sources section
- Links are valid and accessible

## Functional Requirements

### FR-1: Issue Fetching
- Fetch issue via `gh` CLI
- Extract: number, title, body, labels, state, author, URL
- Handle private repos (with authenticated gh)

### FR-2: Topic Extraction
- Analyze issue to identify research questions
- Categorize topics: technical, documentation, best_practices, security, performance
- Prioritize topics: critical, high, medium, low
- Generate search keywords for each topic

### FR-3: Web Research
- Execute web searches using WebSearch tool
- Fetch and read relevant pages using WebFetch
- Evaluate source credibility
- Extract key findings with quotes
- Handle inconclusive searches gracefully

### FR-4: Report Synthesis
- Create executive summary with key findings
- Organize detailed findings by topic
- Generate actionable recommendations
- Format as GitHub-flavored markdown
- Include complete source citations

### FR-5: Comment Posting
- Post report via `gh issue comment`
- Add Wave attribution header
- Verify comment was posted successfully
- Return comment URL

## Non-Functional Requirements

### NFR-1: Reliability
- Retry transient failures (GitHub API, web search)
- Continue pipeline if individual topics fail
- Fresh memory at each step boundary

### NFR-2: Security
- No credentials stored on disk
- Input sanitization for prompt injection
- GitHub token via environment variable only

### NFR-3: Observability
- Contract validation at each step
- Structured artifacts for debugging
- Audit trail via Wave traces

## Pipeline Architecture

```
Input: "owner/repo issue_number"
    │
    ▼
┌─────────────────┐
│  fetch-issue    │  persona: github-analyst
│  (gh CLI)       │  output: issue-content.json
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ analyze-topics  │  persona: researcher
│                 │  output: research-topics.json
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ research-topics │  persona: researcher (NEW)
│ (WebSearch)     │  output: research-findings.json
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│synthesize-report│  persona: summarizer
│                 │  output: research-report.json
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  post-comment   │  persona: github-commenter (NEW)
│  (gh CLI)       │  output: comment-result.json
└─────────────────┘
```

## New Components Required

### Personas

1. **researcher** - Web research specialist
   - Tools: WebSearch, WebFetch, Read, Write(output/*)
   - Constraints: No Bash, no Edit

2. **github-commenter** - Posts comments on issues
   - Tools: Bash(gh issue comment*), Read, Write(output/*)
   - Constraints: No issue editing, no PR operations

### Contracts

1. `issue-content.schema.json` - Parsed GitHub issue
2. `research-topics.schema.json` - Extracted topics with keywords
3. `research-findings.schema.json` - Research results by topic
4. `research-report.schema.json` - Synthesized report with markdown
5. `comment-result.schema.json` - Comment posting result

## Success Metrics

- Pipeline completes successfully for valid inputs
- Research report contains actionable information
- All sources are cited and accessible
- Comment posts successfully to GitHub

## Out of Scope

- Automatic issue resolution/closing
- PR creation based on research
- Multi-issue batch processing
- Research caching/deduplication

## Dependencies

- `gh` CLI installed and authenticated
- WebSearch/WebFetch tools available in adapter
- GitHub token with repo scope

## References

- [re-cinq/CFOAgent#112](https://github.com/re-cinq/CFOAgent/issues/112) - First use case
- Wave Constitution - Pipeline design principles
- Existing pipelines: `gh-poor-issues.yaml` - Similar GitHub integration pattern
