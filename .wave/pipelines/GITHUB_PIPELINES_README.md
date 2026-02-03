# GitHub Integration Pipelines

This directory contains four production-ready pipelines that leverage Wave's GitHub integration to deliver concrete value to developers.

## Overview

These pipelines demonstrate Wave's capability to perform real GitHub operations through the `gh` CLI adapter, producing deterministic, verifiable outputs with proper error handling and validation.

## Available Pipelines

### 1. github-issue-enhancer

**Purpose**: Automatically analyze and enhance poorly documented GitHub issues.

**Input**:
```yaml
repo: "owner/repository"  # Required
threshold: 70             # Optional, default 70 (quality score 0-100)
```

**What it does**:
1. **Scan Issues** - Fetches all open issues and analyzes quality based on:
   - Title quality (length, specificity, capitalization)
   - Description completeness (structure, examples, reproduction steps)
   - Label presence and appropriateness

2. **Plan Enhancements** - Creates structured enhancement recommendations:
   - Improved titles that are clear and specific
   - Enhanced descriptions with proper templates
   - Appropriate labels based on content

3. **Apply Enhancements** - Updates issues via GitHub API:
   - Updates titles (if needed)
   - Adds structured templates preserving original content
   - Applies suggested labels
   - Posts explanatory comment

4. **Verify** - Confirms all enhancements were applied successfully

**Output**: Enhanced GitHub issues with better clarity, structure, and discoverability.

**Example usage**:
```bash
wave run github-issue-enhancer --input '{"repo": "myorg/myproject", "threshold": 60}'
```

**Contracts**:
- `github-issue-analysis.schema.json`
- `github-enhancement-plan.schema.json`
- `github-enhancement-results.schema.json`
- `github-verification-report.schema.json`

---

### 2. github-feature-implementation

**Purpose**: Implement a feature from a GitHub issue and create a ready-to-review pull request.

**Input**:
```yaml
repo: "owner/repository"     # Required
issue: 123                   # Required - issue number to implement
base_branch: "main"          # Optional, default "main"
```

**What it does**:
1. **Analyze Requirement** - Extracts implementation requirements from issue:
   - Requirements and acceptance criteria
   - Affected components and files
   - Technical approach and complexity
   - Related issues and PRs

2. **Create Feature Branch** - Creates and pushes a descriptive feature branch:
   - Branch naming: `feature/issue-{number}-{description}`
   - Based on latest main/base branch

3. **Implement Feature** - Writes production-quality code:
   - Follows existing code patterns
   - Adds comprehensive error handling
   - Includes tests for new functionality
   - Runs test suite to ensure no regressions
   - Commits with clear messages

4. **Create Pull Request** - Generates comprehensive PR:
   - Clear title and structured description
   - Summary of changes and implementation notes
   - Testing instructions and checklist
   - Links to related issues

5. **Verify PR** - Confirms PR is ready for review:
   - All commits included
   - Links to issue correctly
   - CI checks running (if configured)
   - Description complete

**Output**: A ready-to-review pull request with complete implementation and tests.

**Example usage**:
```bash
wave run github-feature-implementation --input '{"repo": "myorg/myproject", "issue": 42}'
```

**Contracts**:
- `navigation-analysis.schema.json`
- `implementation-results.schema.json`
- `github-pr-info.schema.json`
- `github-verification-report.schema.json`

---

### 3. github-issue-cross-linker

**Purpose**: Automatically discover and link related issues to improve discoverability and issue management.

**Input**:
```yaml
repo: "owner/repository"           # Required
state: "open"                      # Optional: open, closed, all (default: open)
similarity_threshold: 0.6          # Optional: 0-1 (default: 0.6)
```

**What it does**:
1. **Fetch Issues** - Retrieves all issues matching state filter:
   - Extracts titles, bodies, labels, existing references
   - Identifies keywords for relationship analysis

2. **Analyze Relationships** - Detects connections between issues:
   - Keyword overlap and semantic similarity
   - Shared labels and themes
   - Dependency patterns (blocks, depends-on)
   - Duplicate detection
   - Calculates similarity scores

3. **Create Cross-Links** - Adds bidirectional links between related issues:
   - Posts comments with relationship type and rationale
   - Handles duplicates appropriately
   - Avoids duplicate links
   - Clear, helpful comment format

4. **Verify Links** - Confirms cross-references were created:
   - Verifies bidirectional links
   - Checks duplicate closures (if applicable)
   - Validates comment posting

**Output**: Enhanced issue relationships with automatic cross-referencing.

**Relationship types detected**:
- `related` - Issues discussing related topics
- `duplicate` - Issues describing the same problem
- `depends-on` - Dependency relationships
- `blocks` - Blocking relationships
- `part-of` - Epic/feature groupings

**Example usage**:
```bash
wave run github-issue-cross-linker --input '{"repo": "myorg/myproject", "state": "open"}'
```

**Contracts**:
- `github-issues-data.schema.json`
- `github-issue-relationships.schema.json` (extends `cross-reference.schema.json`)
- `github-cross-link-results.schema.json`
- `github-verification-report.schema.json`

---

### 4. github-pr-review-automation

**Purpose**: Perform comprehensive automated code review on pull requests.

**Input**:
```yaml
repo: "owner/repository"     # Required
pr: 123                      # Required - PR number to review
post_review: false           # Optional - whether to post review (default: false for safety)
```

**What it does**:
1. **Fetch PR Data** - Retrieves comprehensive PR information:
   - Metadata (title, body, author, labels)
   - Changed files with diff stats
   - Commit history
   - Actual diff content
   - CI/CD status

2. **Security Review** - Identifies security issues:
   - Input validation vulnerabilities
   - Authentication/authorization gaps
   - Credential leaks and data exposure
   - Unsafe cryptography usage
   - Race conditions and resource leaks
   - Path traversal vulnerabilities
   - Unsafe deserialization

3. **Quality Review** - Assesses code quality:
   - Error handling completeness
   - Test coverage and edge cases
   - Code structure and maintainability
   - Naming conventions and clarity
   - Performance implications
   - Documentation gaps
   - Best practices adherence

4. **Synthesize Review** - Combines findings into comprehensive review:
   - Prioritizes issues by severity
   - Groups related issues
   - Extracts positive observations
   - Determines overall assessment (APPROVE/REQUEST_CHANGES/COMMENT)
   - Creates actionable summary

5. **Post Review** - Posts review to GitHub (if enabled):
   - Formatted review comment with sections
   - Overall assessment (approve/request changes/comment)
   - Must-fix vs should-consider items
   - Positive notes and detailed findings

6. **Verify Review** - Confirms review was posted correctly

**Output**: Comprehensive code review with security and quality analysis.

**Safety feature**: By default, `post_review: false` means the review is prepared but NOT posted. Set `post_review: true` to actually post to GitHub.

**Example usage (dry-run)**:
```bash
wave run github-pr-review-automation --input '{"repo": "myorg/myproject", "pr": 42}'
```

**Example usage (post review)**:
```bash
wave run github-pr-review-automation --input '{"repo": "myorg/myproject", "pr": 42, "post_review": true}'
```

**Contracts**:
- `github-pr-data.schema.json`
- `github-pr-review.schema.json`
- `github-verification-report.schema.json`

---

## Architecture

### Personas Used

All pipelines leverage specialized Wave personas:

- **github-analyst** - Analyzes GitHub data, evaluates quality, detects patterns
- **github-enhancer** - Modifies GitHub content (issues, comments, labels)
- **github-pr-creator** - Creates and manages pull requests
- **navigator** - Read-only exploration and verification
- **craftsman** - Code implementation with testing
- **auditor** - Security and quality review
- **philosopher** - Deep analysis and relationship detection

### Contract-Driven Design

Every step output is validated against JSON schemas ensuring:
- Type safety and required fields
- Data structure consistency
- Error detection at step boundaries
- Clear documentation of data flow

### Fresh Memory Architecture

Each step receives:
- Fresh context (no memory inheritance)
- Explicitly injected artifacts from dependencies
- Clear, focused instructions

This ensures deterministic execution and prevents context pollution.

### Error Handling

All pipelines include:
- Schema validation at handovers
- Retry mechanisms (configurable per step)
- Comprehensive error tracking
- Verification steps to confirm operations

## GitHub CLI Integration

These pipelines use the GitHub CLI (`gh`) for all GitHub operations:

### Authentication
Ensure `gh` is authenticated before running:
```bash
gh auth login
```

### Required Permissions
- Read access to repository and issues
- Write access for modification operations (enhancer, cross-linker, PR creator)
- PR review permissions for review automation

### Rate Limiting
GitHub API has rate limits. The pipelines handle this by:
- Batching operations where possible
- Including delays between API calls (in persona implementations)
- Tracking and reporting errors

## Testing

### Dry Run Testing
Most pipelines can be tested in read-only mode first:

1. **Issue Enhancer**: Set high threshold to analyze without enhancing
   ```bash
   wave run github-issue-enhancer --input '{"repo": "myorg/myproject", "threshold": 100}'
   ```

2. **PR Review**: Default `post_review: false` creates review without posting
   ```bash
   wave run github-pr-review-automation --input '{"repo": "myorg/myproject", "pr": 42}'
   ```

### Testing on Small Repositories
Test on smaller repos or test repositories first:
```bash
wave run github-issue-cross-linker --input '{"repo": "myorg/test-repo", "state": "open"}'
```

### Verification Steps
All pipelines include final verification steps that:
- Confirm operations completed successfully
- Check for errors or inconsistencies
- Provide detailed summary reports

## Monitoring and Debugging

### Artifacts
Each step produces artifacts in the workspace:
```
.wave/workspaces/{pipeline}/{step}/artifact.json
```

These contain the full output data for inspection.

### Audit Logs
Wave audit logs capture all tool calls:
```
.wave/traces/{timestamp}-{pipeline}.log
```

Review these to see exact `gh` commands executed.

### Verification Reports
Final verification steps produce comprehensive reports showing:
- What operations were performed
- Success/failure counts
- URLs to modified GitHub resources
- Summary of changes

## Best Practices

### Start Small
- Test on a few issues/PRs before running on entire repos
- Use read-only operations first (navigator persona steps)
- Review artifacts before re-running with write operations

### Incremental Adoption
1. **Phase 1**: Issue enhancer on 5-10 poor quality issues
2. **Phase 2**: Cross-linker on subset of issues
3. **Phase 3**: PR review automation on non-critical PRs
4. **Phase 4**: Feature implementation on simple issues

### Quality Thresholds
Adjust quality thresholds based on your needs:
- `threshold: 80` - Only enhance very poor issues
- `threshold: 60` - Enhance moderately poor issues
- `threshold: 40` - Enhance most issues (aggressive)

### Review Before Posting
Always review the prepared content before posting:
1. Run with dry-run/read-only first
2. Inspect artifacts
3. Verify changes are appropriate
4. Then run with posting enabled

## Troubleshooting

### Common Issues

**"gh: command not found"**
- Install GitHub CLI: https://cli.github.com/
- Ensure it's in PATH

**"HTTP 401: Unauthorized"**
- Run `gh auth login`
- Check token permissions

**"Rate limit exceeded"**
- Wait for rate limit reset
- Use smaller batches
- Consider GitHub Enterprise with higher limits

**"Contract validation failed"**
- Check artifact.json in workspace
- Review error message for missing/invalid fields
- May indicate issue with GitHub API response format

**"Issue/PR not found"**
- Verify repository name format: "owner/repo"
- Check issue/PR number exists
- Confirm you have read access

## Extending the Pipelines

### Adding New Steps
Add steps to existing pipelines for additional functionality:
- Email notifications
- Slack integration
- Metrics collection
- Custom validation rules

### Creating New Pipelines
Use these as templates for new GitHub workflows:
- Release automation
- Changelog generation
- Stale issue management
- Label synchronization

### Custom Personas
Create specialized personas for domain-specific needs:
- Security-focused reviewers
- Documentation specialists
- Performance analyzers

## Support and Feedback

These pipelines are production-ready but should be tested thoroughly in your environment first. Start with small batches, verify outputs, and gradually scale up usage.

For issues or improvements, refer to the Wave documentation and the individual pipeline YAML files for detailed step configurations.

---

**Last Updated**: 2026-02-03
**Wave Version**: Compatible with Wave v1+
**GitHub CLI Version**: Requires gh CLI 2.0+
