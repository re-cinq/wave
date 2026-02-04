# Examples

Explore real-world Wave workflows and configurations.

## Example 1: Simple Feature Addition

Adding a new feature to a web application.

### Task Description
Add user profile page with avatar upload.

### Pipeline Configuration

```yaml
# .wave/pipelines/add-feature.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: add-feature
  description: "Add a new feature with review"

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      root: ./
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Find where user-related code lives. Look for:
        - User model definitions
        - Profile components
        - Auth middleware
        Report the file paths and current structure.

  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
    exec:
      type: prompt
      source: |
        Based on the navigation report, design the user profile feature:
        1. Define user model with avatar field
        2. Specify profile page components
        3. Design avatar upload endpoint
        4. Create validation rules
        Output as a structured specification.

  - id: implement
    persona: craftsman
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: specification
          as: feature_spec
    exec:
      type: prompt
      source: |
        Implement the user profile feature based on the specification:
        1. Add avatar_url field to User model
        2. Create ProfilePage component with avatar display
        3. Implement POST /users/:id/avatar endpoint
        4. Add file upload handling
        Include error handling and validation.
    handover:
      contract:
        type: test_suite
        command: "npm test -- --testPathPattern=profile.test.js"
        must_pass: true
        max_retries: 2

  - id: review
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Review the implemented feature for:
        1. Security vulnerabilities
        2. Performance issues
        3. Code quality problems
        4. Test coverage gaps
        Report any critical issues found.
```

### Running the Example

```bash
wave run .wave/pipelines/add-feature.yaml "Add user profile with avatar"
```

> **Tip:** You can also use the shorthand `wave run add-feature "Add user profile with avatar"`. Wave will automatically look for pipelines in the `.wave/pipelines/` directory.

### Expected Output

Each step creates artifacts in `/tmp/wave/<pipeline-id>/<step-id>/`:
- `navigate/analysis.json` - File paths and structure
- `specify/specification.md` - Feature design document
- `implement/` - Source code changes
- `review/report.md` - Security and quality findings

## Example 2: Bug Fix Workflow

Debugging and fixing a production issue.

### Task Description
Fix memory leak in user session handler.

### Pipeline Configuration

```yaml
# .wave/pipelines/bug-fix.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: bug-fix
  description: "Debug and fix production issues"

steps:
  - id: investigate
    persona: navigator
    exec:
      type: prompt
      source: |
        Investigate the memory leak:
        1. Find session handler code
        2. Look for resource cleanup patterns
        3. Check for goroutine leaks
        4. Identify leak patterns
        Report findings with specific code locations.

  - id: reproduce
    persona: craftsman
    dependencies: [investigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: investigate
          artifact: findings
          as: leak_info
    exec:
      type: command
      source: "go test -run TestMemoryLeak -v"
    handover:
      contract:
        type: test_suite
        command: "ps aux | grep 'process-name' | wc -l"
        must_pass: true

  - id: fix
    persona: craftsman
    dependencies: [reproduce]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Fix the memory leak:
        1. Add proper resource cleanup
        2. Fix goroutine lifecycle management
        3. Add connection pooling if needed
        4. Include defensive programming
        Write tests to verify the fix.

  - id: verify
    persona: auditor
    dependencies: [fix]
    exec:
      type: prompt
      source: |
        Verify the fix:
        1. Run leak detection tests
        2. Check resource usage under load
        3. Verify no regressions
        4. Document the solution
```

## Example 3: Multi-Persona Workflow

Complex feature requiring multiple specializations.

### Using Matrix Strategy

```yaml
steps:
  - id: plan
    persona: philosopher
    exec:
      type: prompt
      source: |
        Break down microservice migration into tasks:
        - Database migration scripts
        - Service API updates
        - Client library updates
        - Documentation updates
        Output as JSON task list.

  - id: execute
    persona: craftsman
    dependencies: [plan]
    strategy:
      type: matrix
      items_source: plan/tasks.json
      item_key: task
      max_concurrency: 4
    exec:
      type: prompt
      source: |
        Execute your assigned task: {{ task }}
        Follow coding standards and include tests.
```

### Task List Example

```json
{
  "tasks": [
    {"task": "Create migration scripts"},
    {"task": "Update service APIs"},
    {"task": "Update client library"},
    {"task": "Write documentation"}
  ]
}
```

This spawns 4 parallel craftsman instances, each working on one task.

## Example 4: Meta-Pipeline

Self-designing pipeline for unknown task types.

### Meta Pipeline Configuration

```yaml
# .wave/pipelines/auto-design.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: auto-design
  description: "Automatically design and execute pipelines"

steps:
  - id: analyze
    persona: philosopher
    exec:
      type: prompt
      source: |
        Analyze this request and design a pipeline:
        {{ input }}
        
        Requirements:
        - Must start with navigator step
        - Each step needs handover contract
        - Use fresh memory strategy
        - Consider parallel execution opportunities
        
        Output valid YAML pipeline definition.

  - id: execute
    persona: meta-executor
    dependencies: [analyze]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Execute the generated pipeline:
        1. Load the YAML from analyze step
        2. Validate it (schema + semantic)
        3. Execute it with depth tracking
        4. Report results
```

## Best Practices

1. **Always Start with Navigator**: Every pipeline should begin by understanding the codebase
2. **Use Handover Contracts**: Ensure quality gates between steps
3. **Leverage Matrix Strategy**: For parallelizable tasks
4. **Include Review Step**: Critical for security and quality
5. **Persist Artifacts**: Each step should output structured artifacts
6. **Monitor Progress**: Use structured output for CI/CD integration

## Troubleshooting Examples

### Issue: Step Won't Start
```yaml
# Check persona reference
personas:
  mypersona:
    adapter: undefined_adapter  # Bad: adapter not defined
```

### Issue: Permission Denied
```yaml
# Check permissions
personas:
  navigator:
    permissions:
      deny: ["Read(*)"]  # Too restrictive
```

### Issue: Contract Failure
```yaml
# Check contract path
handover:
  contract:
    source: output.json  # Wrong: should be relative to workspace
    source: ./output.json  # Correct
```