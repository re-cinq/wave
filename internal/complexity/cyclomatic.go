package complexity

import (
	"go/ast"
	"go/token"
)

// CyclomaticComplexity returns the McCabe cyclomatic complexity for a function
// declaration or function literal body. Returns 1 for an empty body.
//
// Counted decision points:
//   - *ast.IfStmt (if, else-if)
//   - *ast.ForStmt
//   - *ast.RangeStmt
//   - *ast.CaseClause (non-default)
//   - *ast.CommClause (non-default, select)
//   - *ast.BinaryExpr with token.LAND or token.LOR
//
// Function literals nested inside the body are walked too — their decision
// points contribute to the enclosing function's score, matching gocyclo.
func CyclomaticComplexity(body *ast.BlockStmt) int {
	if body == nil {
		return 1
	}
	score := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.IfStmt:
			score++
		case *ast.ForStmt:
			score++
		case *ast.RangeStmt:
			score++
		case *ast.CaseClause:
			if len(v.List) > 0 {
				score++
			}
		case *ast.CommClause:
			if v.Comm != nil {
				score++
			}
		case *ast.BinaryExpr:
			if v.Op == token.LAND || v.Op == token.LOR {
				score++
			}
		}
		return true
	})
	return score
}
