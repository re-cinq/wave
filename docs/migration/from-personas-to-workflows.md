# Migrating from Personas to Workflows

This guide helps you transition from persona-focused AI interactions to workflow-based Infrastructure-as-Code automation. Learn how to transform ad-hoc persona usage into reproducible, shareable workflows that your team can version control and standardize.

## Understanding the Paradigm Shift

### Old Paradigm: Persona-Focused

**How You Used to Think About Wave:**
```bash
# Ad-hoc persona interactions
wave do navigator "analyze this codebase for security issues"
wave do auditor "review this API for vulnerabilities"
wave do craftsman "implement OAuth2 authentication"
```

**Problems with Persona-Centric Approach:**
- **Inconsistent outputs**: Same prompt, different results each time
- **No validation**: No guarantee of output format or quality
- **Individual usage**: Hard to share patterns with team
- **Manual process**: Each step requires human intervention
- **Context loss**: No memory between steps
- **No reproducibility**: Results vary by user and time

### New Paradigm: Workflow-Focused

**How to Think About Wave Now:**
```yaml
# Declarative workflow definition
apiVersion: v1
kind: WavePipeline
metadata:
  name: security-analysis-workflow
steps:
  - id: analyze-codebase
    persona: navigator
    # ... specific analysis configuration
  - id: security-audit
    persona: auditor
    dependencies: [analyze-codebase]
    # ... audit configuration with contracts
  - id: implement-fixes
    persona: craftsman
    dependencies: [security-audit]
    # ... implementation with validation
```

**Benefits of Workflow-Centric Approach:**
- **Consistent outputs**: Same workflow always produces same results
- **Guaranteed quality**: Contracts validate every step
- **Team collaboration**: Workflows are version-controlled and shareable
- **Automated process**: Complete pipeline runs without intervention
- **Context preservation**: Memory and artifacts flow between steps
- **Full reproducibility**: Same input always produces same output

## Migration Patterns

### Pattern 1: Single Persona → Single Step Workflow

**Before (Persona Command):**
```bash
wave do navigator "analyze the React components in src/components for accessibility issues"
```

**After (Simple Workflow):**
```yaml
# accessibility-check.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: accessibility-analysis
  description: "Analyze React components for accessibility compliance"

input:
  type: file_path
  required: true

steps:
  - id: accessibility-audit
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: "{{ input }}"
          target: /components
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze React components for accessibility issues:

        Components to analyze: {{ input }}

        Check for:
        - Missing alt text on images
        - Improper heading hierarchy
        - Missing ARIA labels
        - Color contrast issues
        - Keyboard navigation support

        Output structured findings as JSON.
    output_artifacts:
      - name: accessibility-report
        path: output/accessibility-findings.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: |
          {
            "type": "object",
            "required": ["components_analyzed", "findings", "summary"],
            "properties": {
              "components_analyzed": {"type": "integer", "minimum": 0},
              "findings": {
                "type": "array",
                "items": {
                  "type": "object",
                  "required": ["component", "issue", "severity", "recommendation"],
                  "properties": {
                    "component": {"type": "string"},
                    "issue": {"type": "string"},
                    "severity": {"enum": ["low", "medium", "high", "critical"]},
                    "recommendation": {"type": "string"}
                  }
                }
              },
              "summary": {
                "type": "object",
                "required": ["total_issues", "critical_count", "compliance_score"],
                "properties": {
                  "total_issues": {"type": "integer"},
                  "critical_count": {"type": "integer"},
                  "compliance_score": {"type": "number", "minimum": 0, "maximum": 100}
                }
              }
            }
          }
        on_failure: retry
        max_retries: 2
```

**Usage:**
```bash
# Old way
wave do navigator "analyze the React components..."

# New way
wave run accessibility-analysis --input src/components/
```

### Pattern 2: Multiple Personas → Multi-Step Pipeline

**Before (Manual Persona Chain):**
```bash
# Step 1: Analysis
wave do navigator "analyze this API for security vulnerabilities" > analysis.txt

# Step 2: Review findings (manual read of analysis.txt)
wave do auditor "review these security findings: $(cat analysis.txt)"

# Step 3: Generate fixes (manual copy/paste)
wave do craftsman "implement fixes for these vulnerabilities: ..."
```

**After (Automated Pipeline):**
```yaml
# security-pipeline.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: api-security-pipeline
  description: "Complete API security analysis and remediation"

input:
  type: file_path
  required: true

steps:
  - id: vulnerability-analysis
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: "{{ input }}"
          target: /api-code
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze API code for security vulnerabilities:

        API Code: {{ input }}

        Focus on:
        - SQL injection risks
        - XSS vulnerabilities
        - Authentication bypasses
        - Input validation gaps
        - Authorization flaws

        Output detailed vulnerability report.
    output_artifacts:
      - name: vulnerability-report
        path: output/vulnerabilities.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/vulnerability-report.schema.json

  - id: security-review
    persona: auditor
    dependencies: [vulnerability-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: vulnerability-analysis
          artifact: vulnerability-report
          as: initial_findings
    exec:
      type: prompt
      source: |
        Review and validate security findings:

        Initial Findings: {{ artifacts.initial_findings }}

        Validate each finding:
        - Confirm exploitability
        - Assess business impact
        - Prioritize by risk level
        - Add remediation guidance

        Output validated security assessment.
    output_artifacts:
      - name: security-assessment
        path: output/security-assessment.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/security-assessment.schema.json

  - id: generate-fixes
    persona: craftsman
    dependencies: [security-review]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security-review
          artifact: security-assessment
          as: validated_findings
    workspace:
      mount:
        - source: "{{ input }}"
          target: /api-code
          mode: readwrite
    exec:
      type: prompt
      source: |
        Generate fixes for validated security issues:

        Security Assessment: {{ artifacts.validated_findings }}
        API Code Location: {{ input }}

        For each critical/high severity issue:
        1. Generate secure code fix
        2. Add input validation
        3. Implement proper authentication checks
        4. Add security tests

        Output complete remediation plan with code.
    output_artifacts:
      - name: remediation-plan
        path: output/remediation.md
        type: markdown
      - name: security-patches
        path: output/patches/
        type: directory
    handover:
      contract:
        type: test_suite
        command: "python .wave/validators/security-fix-validator.py"
        must_pass: true
```

**Usage:**
```bash
# Old way (3 manual steps)
wave do navigator "analyze..." > analysis.txt
wave do auditor "review: $(cat analysis.txt)"
wave do craftsman "implement fixes..."

# New way (1 automated pipeline)
wave run api-security-pipeline --input src/api/
```

### Pattern 3: Complex Interactions → Branching Workflows

**Before (Complex Manual Process):**
```bash
# Conditional logic handled manually
if [[ $language == "javascript" ]]; then
    wave do navigator "analyze JavaScript for performance issues"
elif [[ $language == "python" ]]; then
    wave do navigator "analyze Python for performance issues"
fi

# Different personas based on findings
if [[ $performance_critical == "true" ]]; then
    wave do auditor "deep performance analysis"
else
    wave do reviewer "general code review"
fi
```

**After (Conditional Workflow):**
```yaml
# performance-analysis.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: performance-analysis
  description: "Language-aware performance analysis with conditional depth"

input:
  type: file_path
  required: true

steps:
  - id: detect-language
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Detect the primary programming language in: {{ input }}

        Output: {"language": "javascript|python|go|rust|java", "confidence": 0.95}
    output_artifacts:
      - name: language-detection
        path: output/language.json
        type: json

  - id: javascript-analysis
    persona: navigator
    dependencies: [detect-language]
    condition: "{{ steps.detect-language.output.language == 'javascript' }}"
    exec:
      type: prompt
      source: |
        JavaScript performance analysis for: {{ input }}

        Focus on:
        - Bundle size optimization
        - Async/await patterns
        - Memory leaks
        - Render performance
    output_artifacts:
      - name: js-performance-report
        path: output/js-performance.json
        type: json

  - id: python-analysis
    persona: navigator
    dependencies: [detect-language]
    condition: "{{ steps.detect-language.output.language == 'python' }}"
    exec:
      type: prompt
      source: |
        Python performance analysis for: {{ input }}

        Focus on:
        - Algorithm complexity
        - NumPy optimization
        - GIL impact
        - Memory usage patterns
    output_artifacts:
      - name: python-performance-report
        path: output/python-performance.json
        type: json

  - id: critical-analysis
    persona: auditor
    dependencies: [javascript-analysis, python-analysis]
    condition: "{{ steps[javascript-analysis|python-analysis].output.critical_issues > 0 }}"
    memory:
      inject_artifacts:
        - step: detect-language
          artifact: language-detection
        - step: javascript-analysis
          artifact: js-performance-report
          optional: true
        - step: python-analysis
          artifact: python-performance-report
          optional: true
    exec:
      type: prompt
      source: |
        Deep performance analysis for critical issues:

        Language: {{ artifacts.language-detection.language }}
        Performance Report: {{ artifacts[js-performance-report|python-performance-report] }}

        Provide detailed optimization strategies.

  - id: general-review
    persona: reviewer
    dependencies: [javascript-analysis, python-analysis]
    condition: "{{ steps[javascript-analysis|python-analysis].output.critical_issues == 0 }}"
    exec:
      type: prompt
      source: "Generate general performance recommendations"
```

## Migration Strategies

### Strategy 1: Gradual Migration (Recommended)

**Phase 1: Identify Patterns**
```bash
# Audit your current persona usage
grep -r "wave do" scripts/ | sort | uniq -c | sort -nr

# Common patterns you'll find:
# 15 wave do navigator "analyze"
# 12 wave do auditor "review"
# 8 wave do craftsman "implement"
# 6 wave do reviewer "document"
```

**Phase 2: Start with Most Common Pattern**
Convert your most-used persona command into a simple workflow:

```yaml
# Start simple - direct persona → workflow conversion
metadata:
  name: code-analysis  # Your most common use case

steps:
  - id: analyze
    persona: navigator  # Keep same persona
    exec:
      source: "{{ your_common_prompt_pattern }}"
    # Add basic contract for consistency
```

**Phase 3: Add Contracts**
```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      source: "{{ prompt }}"
    output_artifacts:
      - name: analysis-result
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: basic-analysis.schema.json  # Start simple
```

**Phase 4: Connect Multiple Steps**
```yaml
steps:
  - id: analyze
    persona: navigator
    # ... existing config

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis-result
```

**Phase 5: Optimize and Scale**
```yaml
# Add advanced features:
# - Parallel steps
# - Conditional logic
# - Error handling
# - Performance optimization
```

### Strategy 2: Full Conversion (Advanced Teams)

**Assessment Phase:**
```bash
# Create migration assessment
cat > migration-plan.md << 'EOF'
# Persona → Workflow Migration Plan

## Current Persona Usage
- navigator: 45 uses → 3 workflows
- auditor: 30 uses → 2 workflows
- craftsman: 25 uses → 2 workflows
- reviewer: 15 uses → 1 workflow

## Proposed Workflows
1. code-review-pipeline (navigator → auditor → reviewer)
2. feature-development (navigator → craftsman → auditor)
3. security-audit (navigator → auditor → craftsman)
4. documentation-gen (navigator → reviewer)

## Migration Timeline
- Week 1: Create workflow skeletons
- Week 2: Add contracts and validation
- Week 3: Team training and testing
- Week 4: Switch to workflows, deprecate personas
EOF
```

**Conversion Scripts:**
```bash
#!/bin/bash
# migrate-personas.sh

# Extract persona commands from scripts
find . -name "*.sh" -exec grep -l "wave do" {} \; > persona-scripts.list

# For each script, suggest workflow equivalent
while read script; do
    echo "Converting $script..."

    # Extract persona patterns
    grep "wave do navigator" "$script" > navigator-commands.txt
    grep "wave do auditor" "$script" > auditor-commands.txt

    # Generate workflow template
    cat > "${script%.sh}-workflow.yaml" << 'EOF'
# Auto-generated workflow template
# Review and customize before use
apiVersion: v1
kind: WavePipeline
metadata:
  name: $(basename ${script%.sh})
  description: "Migrated from $script"
# ... template structure
EOF
done < persona-scripts.list
```

## Common Migration Challenges

### Challenge 1: Complex Persona Interactions

**Problem:**
```bash
# Complex conditional persona usage
navigator_output=$(wave do navigator "analyze $file")
if echo "$navigator_output" | grep -q "security concern"; then
    auditor_output=$(wave do auditor "security review: $navigator_output")
    if echo "$auditor_output" | grep -q "critical"; then
        wave do craftsman "urgent fix: $auditor_output"
    else
        wave do reviewer "document findings: $auditor_output"
    fi
fi
```

**Solution:**
```yaml
# Workflow with conditional steps
steps:
  - id: analyze
    persona: navigator
    exec:
      source: "analyze {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json

  - id: security-check
    persona: auditor
    dependencies: [analyze]
    condition: "{{ steps.analyze.output.security_concerns > 0 }}"

  - id: urgent-fix
    persona: craftsman
    dependencies: [security-check]
    condition: "{{ steps.security-check.output.severity == 'critical' }}"

  - id: document-findings
    persona: reviewer
    dependencies: [security-check]
    condition: "{{ steps.security-check.output.severity != 'critical' }}"
```

### Challenge 2: State Management Between Personas

**Problem:**
```bash
# Manual state management
echo "$context" > /tmp/context.txt
wave do navigator "analyze with context: $(cat /tmp/context.txt)"
navigator_result=$(cat output.txt)
echo "$navigator_result" > /tmp/context.txt
wave do auditor "review with context: $(cat /tmp/context.txt)"
```

**Solution:**
```yaml
# Automatic artifact flow
steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    exec:
      source: "analyze with context: {{ input }}"
    output_artifacts:
      - name: analysis-with-context
        path: output/analysis.json

  - id: review
    persona: auditor
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis-with-context
          as: previous_analysis
    exec:
      source: "review: {{ artifacts.previous_analysis }}"
```

### Challenge 3: Output Format Inconsistency

**Problem:**
```bash
# Inconsistent output formats
wave do navigator "analyze code" > output1.txt  # Sometimes JSON, sometimes text
wave do auditor "review code" > output2.txt    # Format varies by persona
```

**Solution:**
```yaml
# Consistent output contracts
steps:
  - id: analyze
    persona: navigator
    handover:
      contract:
        type: json_schema
        schema: |
          {
            "type": "object",
            "required": ["findings", "metadata"],
            "properties": {
              "findings": {"type": "array"},
              "metadata": {"type": "object"}
            }
          }

  - id: review
    persona: auditor
    handover:
      contract:
        type: json_schema
        schema: consistent-review.schema.json
```

## Testing Your Migration

### Validation Tests

**Compare Old vs New:**
```bash
#!/bin/bash
# migration-validation.sh

echo "Testing persona → workflow migration..."

# Run old persona approach
echo "Running old approach..."
old_output=$(wave do navigator "analyze src/api.js")

# Run new workflow approach
echo "Running new approach..."
wave run code-analysis --input src/api.js
new_output=$(cat /tmp/wave/*/analyze/output/analysis.json)

# Compare outputs (semantic comparison, not exact match)
python compare-outputs.py "$old_output" "$new_output"
```

**Regression Testing:**
```bash
# Test known inputs with expected outputs
for test_case in tests/migration/*.input; do
    expected_output="tests/migration/$(basename $test_case .input).expected"

    wave run migrated-workflow --input "$test_case" \
        --output /tmp/test-output.json

    if diff -q "$expected_output" /tmp/test-output.json > /dev/null; then
        echo "✅ $(basename $test_case) passed"
    else
        echo "❌ $(basename $test_case) failed"
        diff "$expected_output" /tmp/test-output.json
    fi
done
```

### Performance Comparison

**Measure Efficiency Gains:**
```bash
# Compare execution time
time wave do navigator "analyze large-codebase/"  # Old approach
time wave run codebase-analysis --input large-codebase/  # New approach

# Measure consistency
for i in {1..10}; do
    wave run code-review --input test-file.js > "output-$i.json"
done

# Check if all outputs are identical
if [[ $(ls output-*.json | xargs sha256sum | awk '{print $1}' | sort -u | wc -l) == 1 ]]; then
    echo "✅ Perfect consistency across runs"
else
    echo "❌ Inconsistent outputs detected"
fi
```

## Team Migration Best Practices

### Training Strategy

**Workshop Agenda (2 hours):**
```markdown
# Personas → Workflows Migration Workshop

## Session 1: Understanding Workflows (30 min)
- What are workflows vs personas?
- Infrastructure-as-Code for AI concept
- Live demo: persona command → workflow conversion

## Session 2: Hands-on Migration (45 min)
- Convert your most common persona usage
- Add basic contracts
- Test and validate output

## Session 3: Advanced Patterns (30 min)
- Multi-step pipelines
- Conditional logic
- Error handling

## Session 4: Team Integration (15 min)
- Git workflow integration
- Code review process
- Migration timeline
```

### Migration Checklist

**Individual Developer:**
- [ ] Audit current persona usage patterns
- [ ] Identify top 3 most common use cases
- [ ] Convert simple persona commands to basic workflows
- [ ] Add contracts for output consistency
- [ ] Test workflows with real code examples
- [ ] Update development scripts to use workflows
- [ ] Share workflows with team for feedback

**Team Migration:**
- [ ] Assess team persona usage patterns
- [ ] Create shared workflow library
- [ ] Establish workflow naming conventions
- [ ] Set up git repository for workflows
- [ ] Define contract standards
- [ ] Create migration timeline
- [ ] Train team on workflow creation
- [ ] Establish code review process for workflows
- [ ] Monitor adoption and provide support

**Organization Migration:**
- [ ] Survey current persona usage across teams
- [ ] Create organization workflow standards
- [ ] Set up central workflow registry
- [ ] Define governance policies
- [ ] Create training materials
- [ ] Establish support process
- [ ] Plan gradual rollout by team
- [ ] Track adoption metrics
- [ ] Collect feedback and iterate

## Troubleshooting Common Issues

### Issue: Workflow Takes Longer Than Persona Commands

**Problem:**
```bash
# Persona command (fast but inconsistent)
wave do navigator "quick analysis" # 30 seconds

# Workflow (slower but reliable)
wave run analysis-workflow --input . # 60 seconds
```

**Solution:**
```yaml
# Optimize workflow for speed
steps:
  - id: quick-analysis
    persona: navigator
    memory:
      strategy: fresh  # Faster than context
    workspace:
      mount:
        - source: "{{ input }}"
          target: /code
          mode: readonly
          # Exclude unnecessary files
          exclude: [node_modules, .git, dist]
    exec:
      type: prompt
      source: |
        Quick analysis of: {{ input }}

        Focus only on high-level patterns, skip details.
        Maximum 5 bullet points.
    timeout: 45  # Set reasonable timeout
```

### Issue: Workflow Outputs Don't Match Persona Results

**Problem:**
Different formatting between persona and workflow outputs.

**Solution:**
```yaml
# Migration compatibility mode
steps:
  - id: analysis
    persona: navigator
    exec:
      source: "{{ prompt }}"
    output_artifacts:
      - name: analysis-result
        path: output/analysis.json
        type: json
      - name: legacy-format
        path: output/analysis.txt
        type: text
        # Keep old format during migration
    handover:
      contract:
        type: custom
        validator: .wave/validators/legacy-compatible.py
        # Ensure new output works with existing scripts
```

### Issue: Complex Persona Logic Hard to Convert

**Problem:**
Complex bash logic with persona commands.

**Solution:**
```bash
# Create hybrid approach during migration
#!/bin/bash
# hybrid-migration.sh

# Use workflow for standard parts
wave run standard-analysis --input "$1"

# Keep bash logic for complex parts (temporarily)
analysis_result=$(cat /tmp/wave/*/standard-analysis/output/analysis.json)
if jq -r '.security_issues' <<< "$analysis_result" | grep -q "critical"; then
    # Complex logic stays in bash during migration
    wave run security-deep-dive --input "$1"
fi
```

The migration from personas to workflows represents a fundamental shift toward Infrastructure-as-Code for AI. Take it gradually, start with simple conversions, and evolve toward fully automated pipelines that your team can version control and depend on.

## Next Steps

- [Creating Workflows](/workflows/creating-workflows) - Build new workflows from scratch
- [Team Adoption](/migration/team-adoption) - Organizational migration patterns
- [Enterprise Patterns](/migration/enterprise-patterns) - Large-scale adoption strategies
- [Sharing Workflows](/workflows/sharing-workflows) - Collaborate on workflows with git