# GitHub Integration Pipelines - Complete Index

This directory contains production-ready Wave pipelines for GitHub automation, delivering concrete value through real GitHub operations.

## Quick Navigation

### 📋 Documentation
- **[GITHUB_PIPELINES_README.md](GITHUB_PIPELINES_README.md)** - Comprehensive overview and reference
- **[GITHUB_PIPELINES_QUICK_START.md](GITHUB_PIPELINES_QUICK_START.md)** - One-page quick reference
- **[GITHUB_PIPELINES_TESTING.md](GITHUB_PIPELINES_TESTING.md)** - Testing strategies and guides
- **[EXAMPLES.md](EXAMPLES.md)** - Real-world usage examples
- **[INDEX.md](INDEX.md)** - This file

### 🔧 Pipelines

#### 1. [github-issue-enhancer.yaml](github-issue-enhancer.yaml)
**Enhance poorly documented GitHub issues**
```bash
wave run github-issue-enhancer --input '{"repo": "owner/repo", "threshold": 70}'
```
- Analyzes issue quality (title, description, labels)
- Plans enhancements preserving original content
- Applies structured templates and better titles
- Verifies all updates successful

**Use cases**: Issue cleanup, standardization, improving discoverability

---

#### 2. [github-feature-implementation.yaml](github-feature-implementation.yaml)
**Implement features from issues to ready PRs**
```bash
wave run github-feature-implementation --input '{"repo": "owner/repo", "issue": 42}'
```
- Extracts requirements from issue
- Creates feature branch
- Implements code with tests
- Creates comprehensive PR
- Verifies PR ready for review

**Use cases**: Automated feature development, rapid prototyping, issue resolution

---

#### 3. [github-issue-cross-linker.yaml](github-issue-cross-linker.yaml)
**Discover and link related issues**
```bash
wave run github-issue-cross-linker --input '{"repo": "owner/repo"}'
```
- Fetches all issues with metadata
- Detects relationships (similarity, duplicates, dependencies)
- Creates bidirectional cross-reference links
- Verifies links posted correctly

**Use cases**: Issue organization, duplicate detection, relationship discovery

---

#### 4. [github-pr-review-automation.yaml](github-pr-review-automation.yaml)
**Automated comprehensive code review**
```bash
wave run github-pr-review-automation --input '{"repo": "owner/repo", "pr": 123}'
```
- Fetches PR data and diffs
- Security review (vulnerabilities, credential leaks)
- Quality review (errors, tests, patterns)
- Synthesizes comprehensive review
- Posts review to GitHub (optional)
- Verifies review posted

**Use cases**: Pre-review automation, security scanning, quality checks

---

## Architecture

### Design Principles
- **Fresh Memory**: No context inheritance between steps
- **Contract-Driven**: JSON schema validation at handovers
- **Verifiable**: Every pipeline includes verification steps
- **Safe by Default**: Read-only modes, dry-run capabilities
- **Deterministic**: Reproducible outputs with clear artifacts

### Personas
- **github-analyst** - Analysis and quality assessment
- **github-enhancer** - Issue/PR modifications
- **github-pr-creator** - PR creation and management
- **navigator** - Read-only exploration and verification
- **craftsman** - Code implementation with testing
- **auditor** - Security and quality review
- **philosopher** - Deep analysis and relationships

### Contracts
All pipelines use validated JSON schemas in `.wave/contracts/`:
- `github-issue-analysis.schema.json`
- `github-enhancement-plan.schema.json`
- `github-enhancement-results.schema.json`
- `github-issues-data.schema.json`
- `github-issue-relationships.schema.json`
- `github-cross-link-results.schema.json`
- `github-pr-data.schema.json`
- `github-pr-review.schema.json`
- `github-pr-info.schema.json`
- `github-verification-report.schema.json`

---

## Quick Start

### Prerequisites
```bash
# Install and authenticate GitHub CLI
gh auth login
gh auth status

# Verify repository access
gh repo view owner/repository
```

### Run Your First Pipeline
```bash
# 1. Analyze issues (read-only)
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 100
}'

# 2. Review results
cat .wave/workspaces/github-issue-enhancer/scan-issues/artifact.json | jq .

# 3. Actually enhance issues
wave run github-issue-enhancer --input '{
  "repo": "yourname/test-repo",
  "threshold": 70
}'
```

---

## Common Workflows

### Workflow 1: Issue Management
```bash
# Clean up issue backlog
wave run github-issue-enhancer --input '{"repo": "myorg/project", "threshold": 60}'

# Link related issues
wave run github-issue-cross-linker --input '{"repo": "myorg/project"}'
```

### Workflow 2: Feature Development
```bash
# Implement feature from issue
wave run github-feature-implementation --input '{"repo": "myorg/project", "issue": 42}'

# Review the created PR
wave run github-pr-review-automation --input '{"repo": "myorg/project", "pr": 123}'
```

### Workflow 3: Code Quality
```bash
# Review all open PRs
for pr in $(gh pr list --repo myorg/project --json number -q '.[].number'); do
  wave run github-pr-review-automation --input "{\"repo\": \"myorg/project\", \"pr\": $pr}"
done
```

---

## Safety and Testing

### Safety Features
- ✓ Dry-run modes (high thresholds, post_review: false)
- ✓ Verification steps in every pipeline
- ✓ Artifact preservation for inspection
- ✓ Comprehensive audit logs
- ✓ Contract validation at boundaries

### Testing Strategy
1. **Syntax validation** - YAML well-formed
2. **Read-only testing** - No modifications
3. **Small batch** - Test on 5-10 items
4. **Review artifacts** - Inspect outputs
5. **Full execution** - Run on production data

See [GITHUB_PIPELINES_TESTING.md](GITHUB_PIPELINES_TESTING.md) for complete testing guide.

---

## Monitoring

### Check Pipeline Results
```bash
# View verification report
cat .wave/workspaces/{pipeline}/verify-*/artifact.json | jq .

# Count operations
jq '.total_enhanced' .wave/workspaces/github-issue-enhancer/apply-enhancements/artifact.json

# View assessment
jq '.overall_assessment' .wave/workspaces/github-pr-review-automation/synthesize-review/artifact.json
```

### Audit Logs
```bash
# List recent runs
ls -lt .wave/traces/

# View specific run
cat .wave/traces/$(ls -t .wave/traces/ | head -1)
```

---

## Troubleshooting

### Common Issues

| Error | Solution |
|-------|----------|
| `gh: command not found` | Install GitHub CLI from https://cli.github.com/ |
| `HTTP 401: Unauthorized` | Run `gh auth login` to authenticate |
| `Rate limit exceeded` | Wait for reset or use smaller batches |
| `Issue/PR not found` | Verify repo format: "owner/repo" and number exists |
| `Contract validation failed` | Check artifact.json for missing/invalid fields |

### Debug Commands
```bash
# Check authentication
gh auth status

# Check rate limits
gh api rate_limit

# Test repository access
gh repo view owner/repo

# View pipeline artifacts
ls -la .wave/workspaces/{pipeline}/

# Check audit logs
tail -f .wave/traces/latest.log
```

---

## Integration

### CI/CD Integration
See [GITHUB_PIPELINES_QUICK_START.md](GITHUB_PIPELINES_QUICK_START.md) for GitHub Actions examples.

### Custom Workflows
Combine pipelines for complete automation:
1. Issue enhancement → Feature implementation → PR creation → Automated review
2. Continuous issue cleanup + cross-linking
3. Automated PR review on all new PRs

---

## Performance

### Expected Performance
- **Issue Enhancer**: ~5-10 issues/minute (GitHub API limited)
- **Feature Implementation**: ~5-15 minutes per feature (depends on complexity)
- **Cross Linker**: ~50-100 issues/minute (analysis only)
- **PR Review**: ~2-5 minutes per PR (depends on size)

### Optimization Tips
- Use appropriate thresholds to limit scope
- Process in batches during off-hours
- Monitor rate limits with `gh api rate_limit`
- Run multiple pipelines in parallel (different repos)

---

## Extending Pipelines

### Customization Options
- Adjust quality thresholds
- Modify persona system prompts
- Add custom validation rules
- Extend contract schemas
- Create new pipeline steps

### Creating New Pipelines
Use these as templates for:
- Release automation
- Changelog generation
- Stale issue management
- Label synchronization
- Dependency updates

---

## Support

### Resources
- [Wave Documentation](../../README.md)
- [GitHub CLI Docs](https://cli.github.com/manual/)
- [JSON Schema](https://json-schema.org/)

### Getting Help
1. Review pipeline YAML files for step details
2. Check contract schemas in `.wave/contracts/`
3. Inspect artifacts in `.wave/workspaces/`
4. Review audit logs in `.wave/traces/`
5. Consult documentation files in this directory

---

## File Inventory

### Pipelines (4)
- `github-issue-enhancer.yaml` (8.4 KB)
- `github-feature-implementation.yaml` (12 KB)
- `github-issue-cross-linker.yaml` (9.2 KB)
- `github-pr-review-automation.yaml` (17 KB)

### Contracts (10 GitHub-specific)
- `github-issue-analysis.schema.json`
- `github-enhancement-plan.schema.json`
- `github-enhancement-results.schema.json`
- `github-issues-data.schema.json`
- `github-issue-relationships.schema.json`
- `github-cross-link-results.schema.json`
- `github-pr-data.schema.json`
- `github-pr-review.schema.json`
- `github-pr-info.schema.json`
- `github-verification-report.schema.json`

### Documentation (5)
- `GITHUB_PIPELINES_README.md` (14 KB) - Complete reference
- `GITHUB_PIPELINES_QUICK_START.md` (6.4 KB) - Quick reference
- `GITHUB_PIPELINES_TESTING.md` (17 KB) - Testing guide
- `EXAMPLES.md` (18 KB) - Real-world examples
- `INDEX.md` (This file) - Navigation and overview

### Total
- **4 production-ready pipelines**
- **10 JSON schema contracts**
- **5 comprehensive documentation files**
- **All with proper error handling, validation, and verification**

---

**Status**: Production Ready ✓

**Last Updated**: 2026-02-03

**Wave Version**: Compatible with Wave v1+

**GitHub CLI Version**: Requires gh CLI 2.0+

---

## Next Steps

1. **Read** [GITHUB_PIPELINES_QUICK_START.md](GITHUB_PIPELINES_QUICK_START.md) for commands
2. **Try** examples from [EXAMPLES.md](EXAMPLES.md) on test repositories
3. **Test** using guide in [GITHUB_PIPELINES_TESTING.md](GITHUB_PIPELINES_TESTING.md)
4. **Deploy** to production after successful testing
5. **Monitor** results and adjust parameters as needed

Happy automating! 🚀
