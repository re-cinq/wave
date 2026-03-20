# docs: update README command table to include all 22 registered commands

**Issue**: [#522](https://github.com/re-cinq/wave/issues/522)
**Author**: nextlevelshit
**State**: OPEN

## Description

The README command table is incomplete. Currently missing 8 commands:

- do
- meta
- validate
- clean
- status
- logs
- cancel
- chat

The README should document all 22 registered commands for comprehensive CLI reference.

## Assessment Notes

The issue body lists 8 commands as missing but several (do, meta, validate, clean, status, logs, cancel) are already present in the README. The body may be stale. The actual audit reveals the following gaps:

### Registered Commands (22 total from `cmd/wave/main.go`)

1. init, 2. validate, 3. run, 4. resume, 5. do, 6. meta, 7. clean, 8. list, 9. status, 10. logs, 11. cancel, 12. artifacts, 13. migrate, 14. serve, 15. chat, 16. compose, 17. doctor, 18. suggest, 19. skills, 20. postmortem, 21. agent, 22. bench

### Currently in README "Commands" Tables (12)

init, run, do, meta, cancel, status, logs, artifacts, list, bench, validate, clean

### Missing from "Commands" Tables (10)

| Command | Short Description |
|---------|-------------------|
| `resume` | Resume a failed pipeline run |
| `chat` | Open interactive analysis of a pipeline run |
| `compose` | Validate and execute a pipeline sequence |
| `doctor` | Check project health and environment setup |
| `suggest` | Propose pipeline runs based on codebase state |
| `skills` | Skill lifecycle management |
| `postmortem` | Analyse a failed pipeline run and suggest recovery steps |
| `agent` | Persona-to-agent compiler utilities |
| `serve` | Start the web dashboard server |
| `migrate` | Database migration management |

### Missing from CLI Reference Block (7)

resume, compose, doctor, suggest, skills, postmortem, agent

### Also Missing (built-in cobra commands, already in CLI block)

completion, help — these are already shown in the CLI Reference block.

## Acceptance Criteria

- [ ] All 22 registered commands appear in the "Commands" tables section
- [ ] The CLI Reference text block lists all registered commands
- [ ] Commands are grouped logically in appropriate table sections
- [ ] Descriptions match the `Short` field from the Go source code
