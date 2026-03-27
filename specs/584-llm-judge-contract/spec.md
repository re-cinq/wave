# LLM-as-Judge Contract Type for Subjective Quality Assessment

**Issue**: [re-cinq/wave#584](https://github.com/re-cinq/wave/issues/584)
**Labels**: enhancement
**Author**: nextlevelshit

## Context

Wave's existing contract types (`json_schema`, `typescript_interface`, `test_suite`, `markdown_spec`, `format`, `non_empty_file`) handle objective validation — schema conformance, test passes, format rules. They cannot assess subjective quality like code clarity, architectural consistency, or security posture.

An **LLM-as-judge** contract type uses a cheap LLM to evaluate step output against human-defined criteria, complementing the existing contract types.

## Design

### Contract YAML Schema

```yaml
contracts:
  - name: code-quality
    type: llm_judge
    model: claude-haiku-4-5     # cheap model for judging
    criteria:
      - "Code follows existing project patterns and conventions"
      - "No security vulnerabilities (injection, XSS, hardcoded secrets)"
      - "Error handling is comprehensive - no swallowed errors"
      - "Functions are focused - single responsibility"
    threshold: 0.8               # 80% of criteria must pass
    on_failure: rework           # or fail/skip/continue
```

### Judge Response Schema

The judge LLM returns structured assessment:

```json
{
  "criteria_results": [
    {"criterion": "Code follows existing patterns", "pass": true, "reasoning": "..."},
    {"criterion": "No security vulnerabilities", "pass": true, "reasoning": "..."},
    {"criterion": "Error handling comprehensive", "pass": false, "reasoning": "Missing error check on DB query at line 42"}
  ],
  "overall_pass": true,
  "score": 0.85,
  "summary": "Code is solid with one minor error handling gap"
}
```

### Integration with Existing Contracts

LLM judge runs alongside other contract types — a step can have both `test_suite` AND `llm_judge` contracts:

```yaml
steps:
  - name: implement
    contracts: [tests-pass, code-quality]
```

### Use Cases

- **Code review quality** - assess if PR review feedback is actionable and specific
- **Plan completeness** - evaluate if a plan covers all acceptance criteria
- **Security posture** - check for common vulnerability patterns
- **Documentation quality** - assess if docs are clear and complete
- **Architectural alignment** - check if changes follow project architecture

## Acceptance Criteria

1. New `llm_judge` contract type registered in `internal/contract/`
2. Judge executor sends step output + criteria to a cheap LLM and parses the structured response
3. Threshold evaluation: pass if `(criteria_passed / total_criteria) >= threshold`
4. Integration with existing contract validation pipeline (`Validate()`, `NewValidator()`)
5. New fields added to manifest schema (`model`, `criteria`, `threshold`)
6. Unit tests covering: valid pass, threshold failure, API error handling, missing config
7. Works alongside existing contract types on the same step
