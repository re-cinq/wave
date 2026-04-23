---
name: agentic-coding
description: "Builds multi-agent coding systems with orchestration, goal decomposition, and iterative self-improvement loops. Use when the user asks for autonomous coding workflows, agent orchestration, multi-agent architectures, or AI-driven development pipelines."
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Workflow: Building an Agentic System

1. **Define agent boundaries** — identify distinct responsibilities (code generation, review, testing, deployment)
2. **Design message protocol** — standardize inter-agent communication format
3. **Implement orchestrator** — route tasks, collect results, handle failures
4. **Add feedback loops** — wire evaluation metrics back into improvement cycles
5. **Validate end-to-end** — test agent communication before adding orchestration complexity

## Key Patterns

### Self-Improving Loop (Go)

```go
func (cg *CodeGenerator) improveCode(ctx context.Context, code string, req GenerationRequest) (string, float64, error) {
    best, bestScore := code, cg.evaluateCode(code, req)
    for i := 0; i < 5; i++ {
        select {
        case <-ctx.Done():
            return best, bestScore, ctx.Err()
        default:
        }
        for _, improvement := range cg.generateImprovements(best, req) {
            candidate := cg.applyImprovement(best, improvement)
            if score := cg.evaluateCode(candidate, req); score > bestScore {
                best, bestScore = candidate, score
            }
        }
    }
    return best, bestScore, nil
}
```

## Best Practices

- Log all agent decisions and actions for auditability
- Use explicit dependency graphs to prevent deadlocks
- Maintain human oversight with rollback capabilities
- Build feedback loops — wire evaluation metrics into improvement cycles

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
