---
name: ddd
description: "Designs bounded contexts, defines aggregate boundaries, implements repository patterns, and models domain events. Use when the user asks about domain modeling, designing aggregates, structuring a domain layer, implementing DDD patterns, or separating bounded contexts."
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Workflow: Designing an Aggregate

1. Identify the core invariant the aggregate protects
2. Define the aggregate root — the single entry point for all mutations
3. Keep it small: only include entities that must be consistent within one transaction
4. Add domain events for cross-aggregate communication
5. Write the repository interface in the domain layer
6. Validate: can each command complete in a single transaction without locking other aggregates?

## Key Patterns

### Aggregate Root with Domain Events (Go)

```go
type Order struct {
    ID     OrderID
    Items  []LineItem
    Status OrderStatus
    events []DomainEvent
}

func (o *Order) Confirm() error {
    if o.Status != OrderPending {
        return fmt.Errorf("cannot confirm order in status %s", o.Status)
    }
    if len(o.Items) == 0 {
        return errors.New("cannot confirm empty order")
    }
    o.Status = OrderConfirmed
    o.events = append(o.events, OrderConfirmed{OrderID: o.ID, At: time.Now()})
    return nil
}

func (o *Order) DomainEvents() []DomainEvent { return o.events }
```

### Repository Interface (Go)

```go
type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id OrderID) (*Order, error)
}
```

## Best Practices

- **Keep aggregates small** — reference other aggregates by ID, not by embedding
- **Name events in past tense**: `OrderConfirmed`, `PaymentReceived`
- **One repository per aggregate root** — domain interface, infrastructure implementation
- **Avoid anemic models** — encapsulate behavior in entities, not services
- Use separate packages per bounded context with anti-corruption layers at boundaries

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
