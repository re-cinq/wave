// Package complexity provides in-tree Go AST complexity analysis for the
// wave audit pipeline. It computes cyclomatic and cognitive complexity scores
// per function and emits findings compatible with the shared-findings JSON
// schema used by audit pipelines.
//
// # Cyclomatic complexity
//
// The classic McCabe metric: count of linearly independent paths through a
// function. Implementation counts one for the function entry plus one for each
// of: if, else if, for, range, case clause, communication clause, &&, ||.
// Range clauses inside switch/select are counted once. Default cases are not
// counted.
//
// # Cognitive complexity
//
// Sonar's cognitive complexity rule set (https://www.sonarsource.com/docs/
// CognitiveComplexity.pdf):
//
//   - Increments: if, else if, else, ternary, switch, for, range, case clauses,
//     catch (defer), goto, break/continue with label, recursion, &&, ||.
//   - Nesting bonus: control-flow constructs add (1 + nesting depth) instead of
//     just 1, where nesting is the count of enclosing control-flow nodes.
//   - Shorthand for boolean sequences: a chain of && or || counts once at the
//     first occurrence; alternations between && and || each add one.
//
// # Thresholds
//
// Default thresholds: cyclomatic and cognitive complexity must be ≤ 15 to
// pass; ≥ 10 emits a medium-severity finding (warn). Both are configurable
// via Options.
package complexity
