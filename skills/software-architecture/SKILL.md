---
name: software-architecture
description: "Evaluates system architectures, recommends patterns (microservices, event-driven, hexagonal), creates ADRs, and designs scalable distributed systems. Use when the user asks about choosing architectural patterns, designing system architecture, making architectural trade-offs, or conducting architecture reviews."
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Workflow: Architecture Decision

1. **Capture requirements** — functional, non-functional (latency, throughput, availability targets)
2. **Identify constraints** — team size, existing infrastructure, compliance, budget
3. **Evaluate candidate patterns** — score each against requirements and constraints
4. **Document in an ADR** — record decision, alternatives considered, and trade-offs
5. **Validate** — prototype the riskiest assumption before committing

## Key Patterns

### Circuit Breaker (Python)
```python
from enum import Enum
import time

class CircuitState(Enum):
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"

class CircuitBreaker:
    def __init__(self, failure_threshold: int = 5, timeout: int = 60):
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.failure_count = 0
        self.last_failure_time = None
        self.state = CircuitState.CLOSED
    
    def call(self, func, *args, **kwargs):
        if self.state == CircuitState.OPEN:
            if time.time() - self.last_failure_time > self.timeout:
                self.state = CircuitState.HALF_OPEN
            else:
                raise Exception("Circuit breaker is OPEN")
        try:
            result = func(*args, **kwargs)
            if self.state == CircuitState.HALF_OPEN:
                self.reset()
            return result
        except Exception as e:
            self.record_failure()
            raise e
    
    def record_failure(self):
        self.failure_count += 1
        self.last_failure_time = time.time()
        if self.failure_count >= self.failure_threshold:
            self.state = CircuitState.OPEN
    
    def reset(self):
        self.failure_count = 0
        self.state = CircuitState.CLOSED
```

### Event Bus (Go)
```go
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Data      map[string]interface{} `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
    Source    string                 `json:"source"`
}

type EventBus struct {
    handlers map[string][]EventHandler
}

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
    go func() {
        for _, handler := range eb.handlers[event.Type] {
            handler.Handle(event)
        }
    }()
}
```

### Architecture Decision Record (ADR)
```markdown
# ADR-001: <Title>

## Status
Proposed | Accepted | Deprecated

## Context
<What problem are we solving and what constraints exist>

## Decision
<What we chose and why>

## Alternatives Considered
- Option A: <pros/cons>
- Option B: <pros/cons>

## Consequences
- Positive: <benefits>
- Negative: <trade-offs>
```

## Decision Priorities

Always weigh:
1. Business requirements and constraints
2. Non-functional requirements (performance, security, scalability)
3. Team capabilities and expertise
4. Operational complexity and maintenance burden
5. Long-term maintainability and evolution

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
