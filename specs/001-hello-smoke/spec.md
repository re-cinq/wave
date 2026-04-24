# smoke: add hello() function in hello.go

**Issue:** [nextlevelshit/wave-testing#1](https://github.com/nextlevelshit/wave-testing/issues/1)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** (none)

## Body

Seed issue for wave impl-issue smoke runs.

### Task
Create file `hello.go` at repo root with:
```go
package main

func hello() string {
    return "hello, world"
}
```

### Why
Small, deterministic task to validate impl-issue pipeline end-to-end.

## Acceptance Criteria

- [ ] `hello.go` exists at repo root
- [ ] `hello()` returns `"hello, world"`
- [ ] PR opened

## Metadata

- Complexity: trivial
- Quality score: 95
- Skipped speckit steps: specify, clarify, checklist, analyze
