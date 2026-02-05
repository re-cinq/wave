# Tasks: Issue Research Pipeline

## Task List

### Phase 1: Personas

- [x] **T1.1** Create `.wave/personas/researcher.md`
- [x] **T1.2** Create `.wave/personas/github-commenter.md`
- [x] **T1.3** Add personas to `wave.yaml`

### Phase 2: Contracts

- [x] **T2.1** Create `.wave/contracts/issue-content.schema.json`
- [x] **T2.2** Create `.wave/contracts/research-topics.schema.json`
- [x] **T2.3** Create `.wave/contracts/research-findings.schema.json`
- [x] **T2.4** Create `.wave/contracts/research-report.schema.json`
- [x] **T2.5** Create `.wave/contracts/comment-result.schema.json`

### Phase 3: Pipeline

- [x] **T3.1** Create `.wave/pipelines/issue-research.yaml`

### Phase 4: Validation

- [x] **T4.1** Run `go test ./...` to verify no regressions
- [ ] **T4.2** Test pipeline with `wave run issue-research "re-cinq/CFOAgent 112"`

## Dependencies

```
T1.1, T1.2 → T1.3 → T3.1
T2.1, T2.2, T2.3, T2.4, T2.5 → T3.1
T3.1 → T4.1 → T4.2
```

## Status

- **Completed**: T1.1, T1.2, T1.3, T2.1, T2.2, T2.3, T2.4, T2.5, T3.1, T4.1
- **Pending**: T4.2 (manual testing)
