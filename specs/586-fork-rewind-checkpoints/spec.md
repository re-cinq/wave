# feat: fork/rewind from checkpoints for non-destructive experimentation

**Issue**: [re-cinq/wave#586](https://github.com/re-cinq/wave/issues/586)
**Labels**: enhancement
**Author**: nextlevelshit
**Complexity**: complex

## Context

Fabro supports **non-destructive forking** from any checkpoint — creating a new independent run that branches from a specific point in a completed run. Also supports **destructive rewind** to replay from a checkpoint. Combined with their dual-branch Git storage (run branch for code, metadata branch for state), this makes execution history fully inspectable and branchable.

Wave's ResumeManager creates subpipelines from failure points but only supports linear forward resumption — you can't fork a successful run to try a different approach from step 3.

## Design

### Fork

Create a new run branching from a specific step of a completed run:

```bash
wave fork <run-id> --from-step plan     # fork from after "plan" step
wave fork <run-id> --from-step 3        # fork from after step 3
wave fork <run-id> --list               # list available fork points
```

Fork creates a new run that:
1. Copies artifacts from all steps up to the fork point
2. Copies workspace state at the fork point
3. Starts execution from the step after the fork point
4. Is fully independent — original run is untouched

### Rewind

Reset a run to an earlier checkpoint (destructive):

```bash
wave rewind <run-id> --to-step plan     # rewind to after "plan"
wave resume <run-id>                     # then resume from there
```

### Use Cases

- **A/B testing approaches**: Fork after planning, try two different implementation strategies
- **Debugging**: Fork a failed run, change one variable, re-run from failure point
- **Iterative refinement**: Rewind to plan step, revise the plan, re-implement
- **Cost saving**: Don't re-run expensive early steps when only late steps need changes

### Checkpoint Enrichment

Current state DB needs to store enough for fork/rewind:
- Workspace snapshot (Git commit SHA at each step boundary)
- Full artifact state at each step boundary
- Context/environment at each step boundary

## Implementation Scope

1. `wave fork` CLI command
2. `wave rewind` CLI command
3. Checkpoint enrichment in state DB (workspace SHA, artifact snapshot per step)
4. Fork executor — creates new run from checkpoint state
5. Rewind — resets state DB and workspace to checkpoint

## Research Sources

- Fabro fork: `fabro fork <RUN_ID> plan@2` — non-destructive, creates new run
- Fabro rewind: `fabro rewind <RUN_ID> plan@2` — destructive, resets original
- Fabro dual-branch: run branch (code) + metadata branch (state as JSON in Git)

## Acceptance Criteria

1. `wave fork <run-id> --from-step <step>` creates a new independent run starting from the step after the fork point, with all prior artifacts and workspace state preserved
2. `wave fork <run-id> --list` displays available fork points (completed steps) with their checkpoint metadata
3. `wave rewind <run-id> --to-step <step>` destructively resets a run's state to the specified checkpoint, allowing `wave resume` to re-execute from that point
4. Checkpoint data (git commit SHA, artifact paths, artifact content hashes) is captured at each step boundary during pipeline execution
5. Fork from a completed or failed run works; fork from a running pipeline is rejected
6. Rewind on a running pipeline is rejected
7. Original run state is completely untouched after a fork operation
8. Step index resolution works for both step names and numeric indices
9. All new functionality has unit and integration tests
10. Migration v13 adds the checkpoint table and forked_from_run_id column with proper indexes
11. Both commands support `--json` output format
12. Existing `wave resume` behavior is not broken
