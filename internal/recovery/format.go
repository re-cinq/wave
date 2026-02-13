package recovery

import "strings"

// FormatRecoveryBlock renders a RecoveryBlock as a human-readable text block
// suitable for printing to stderr. The output includes a header and each hint
// with its label and command, totaling no more than 8 content lines.
func FormatRecoveryBlock(block *RecoveryBlock) string {
	if block == nil || len(block.Hints) == 0 {
		return ""
	}

	var sb strings.Builder

	// Leading blank separator line
	sb.WriteString("\n")

	// Header
	sb.WriteString("Recovery options:\n")

	for _, hint := range block.Hints {
		sb.WriteString("  ")
		sb.WriteString(hint.Label)
		sb.WriteString(":\n")
		sb.WriteString("    ")
		sb.WriteString(hint.Command)
		sb.WriteString("\n")
	}

	return sb.String()
}
