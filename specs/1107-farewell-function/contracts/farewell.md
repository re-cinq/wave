# Contract: `internal/farewell`

## Public API

```go
package farewell

// Farewell returns the farewell line for the given recipient name.
// Empty or whitespace-only name yields the generic default.
func Farewell(name string) string

// WriteFarewell writes Farewell(name) followed by "\n" to w, unless suppress
// is true, in which case it is a no-op. Returns any write error.
func WriteFarewell(w io.Writer, name string, suppress bool) error
```

## Behavioral Contract

| # | Condition                                 | Expected                                                              | Maps to      |
| - | ----------------------------------------- | --------------------------------------------------------------------- | ------------ |
| 1 | `Farewell("")`                            | returns `"Farewell — see you next wave."` (non-empty)                  | FR-001, FR-003 |
| 2 | `Farewell("Alice")`                       | returns `"Farewell, Alice — see you next wave."` (contains `"Alice"`) | FR-002       |
| 3 | `Farewell("  Alice  ")`                   | trims whitespace → same as case 2                                      | FR-002       |
| 4 | `Farewell("")` called twice               | identical output                                                       | SC-003       |
| 5 | `WriteFarewell(buf, "Alice", false)`      | buf receives `"Farewell, Alice — see you next wave.\n"`               | FR-006       |
| 6 | `WriteFarewell(buf, "Alice", true)`       | buf unchanged, no error                                                | FR-005, FR-011 |

## CLI Integration Contract

| # | Condition                                                                     | Expected                                        | Maps to |
| - | ----------------------------------------------------------------------------- | ----------------------------------------------- | ------- |
| C1 | Successful Wave CLI command, stdout is a TTY, `--quiet` not set               | stdout ends with a farewell line                | FR-004, SC-001 |
| C2 | `--quiet` set                                                                 | no farewell line in stdout                      | FR-005, FR-011, SC-002 |
| C3 | stdout piped to file (non-TTY)                                                | no farewell line                                | FR-005, SC-002 |
| C4 | Command fails (non-zero exit)                                                 | no farewell line; error output unaffected       | FR-007 |
| C5 | `$USER=alice`, successful command, TTY                                        | farewell line contains `alice`                  | FR-010 |
| C6 | `$USER` unset/empty, successful command, TTY                                  | generic farewell line (no empty-name artifact)  | FR-010, edge case |
| C7 | CLI and TUI exit paths                                                        | identical farewell wording                      | FR-008 |

## Non-goals / Explicit Exclusions

- No new CLI flag (FR-011).
- No localization/i18n (FR-009).
- No randomised message pool (FR-009).
- No persistence, no config file entry.
