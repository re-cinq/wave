package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// GateDecision captures the result of a human gate interaction.
type GateDecision struct {
	Choice    string    // Selected choice key
	Label     string    // Human-readable label
	Text      string    // Freeform text (empty if not provided)
	Timestamp time.Time // When the decision was made
	Target    string    // Resolved target step (from choice definition)
}

// GateHandler is the interface for interaction channels that present gate
// choices to a human and return their decision.
type GateHandler interface {
	Prompt(ctx context.Context, gate *GateConfig) (*GateDecision, error)
}

// AutoApproveHandler immediately returns the default choice without user interaction.
// Used for --auto-approve mode and CI environments.
type AutoApproveHandler struct{}

// Prompt returns the default choice from the gate configuration.
func (h *AutoApproveHandler) Prompt(_ context.Context, gate *GateConfig) (*GateDecision, error) {
	if gate.Default == "" {
		return nil, fmt.Errorf("auto-approve requires a default choice but gate has none")
	}

	choice := gate.FindChoiceByKey(gate.Default)
	if choice == nil {
		return nil, fmt.Errorf("auto-approve: default key %q not found in choices", gate.Default)
	}

	return &GateDecision{
		Choice:    choice.Key,
		Label:     choice.Label,
		Timestamp: time.Now(),
		Target:    choice.Target,
	}, nil
}

// CLIGateHandler presents gate choices in the terminal using stdin/stdout.
type CLIGateHandler struct {
	In  io.Reader // Defaults to os.Stdin
	Out io.Writer // Defaults to os.Stderr
}

// Prompt displays choices and reads the user's selection from the terminal.
func (h *CLIGateHandler) Prompt(ctx context.Context, gate *GateConfig) (*GateDecision, error) {
	in := h.In
	if in == nil {
		in = os.Stdin
	}
	out := h.Out
	if out == nil {
		out = os.Stderr
	}

	// Check for TTY when using real stdin
	if in == os.Stdin {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("gate requires interactive terminal (stdin is not a TTY); use --auto-approve for non-interactive mode")
		}
	}

	// Display prompt
	if gate.Prompt != "" {
		fmt.Fprintf(out, "\n  %s\n\n", gate.Prompt)
	} else if gate.Message != "" {
		fmt.Fprintf(out, "\n  %s\n\n", gate.Message)
	}

	// Display choices
	for _, c := range gate.Choices {
		fmt.Fprintf(out, "  [%s] %s", c.Key, c.Label)
		if c.Target == "_fail" {
			fmt.Fprintf(out, " (abort)")
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out)

	scanner := bufio.NewScanner(in)

	// Read choice
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fmt.Fprintf(out, "  Choice: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("reading input: %w", err)
			}
			return nil, fmt.Errorf("unexpected end of input")
		}

		key := strings.TrimSpace(scanner.Text())
		choice := gate.FindChoiceByKey(key)
		if choice == nil {
			fmt.Fprintf(out, "  Invalid choice %q. Try again.\n", key)
			continue
		}

		decision := &GateDecision{
			Choice:    choice.Key,
			Label:     choice.Label,
			Timestamp: time.Now(),
			Target:    choice.Target,
		}

		// Read freeform text if enabled
		if gate.Freeform {
			fmt.Fprintf(out, "  Additional notes (press Enter to skip): ")
			if scanner.Scan() {
				text := strings.TrimSpace(scanner.Text())
				if text != "" {
					decision.Text = text
				}
			}
		}

		return decision, nil
	}
}
