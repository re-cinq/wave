package complexity

import (
	"go/ast"
	"go/token"
)

// CognitiveComplexity returns the cognitive complexity score for a function
// per Sonar's specification.
//
// Rules (summary):
//   - Each control-flow break (if/for/range/switch/select, labeled break/
//     continue, goto) adds 1 + current nesting depth.
//   - Else and else-if add 1 (no nesting bonus).
//   - Logical-operator chains: each transition between && and || adds 1; a
//     run of identical logical operators counts once at the first occurrence.
//   - Recursion (call to the enclosing function by name) adds 1.
//   - Function literals reset nesting for their own body but accumulate into
//     the outer total.
func CognitiveComplexity(fn *ast.FuncDecl) int {
	if fn == nil || fn.Body == nil {
		return 0
	}
	w := &cognitiveWalker{name: fn.Name.Name}
	w.walkBlock(fn.Body, 0)
	return w.score
}

// CognitiveComplexityFunc computes cognitive complexity for a function literal
// or block. funcName is used for recursion detection; pass "" if unknown.
func CognitiveComplexityFunc(name string, body *ast.BlockStmt) int {
	if body == nil {
		return 0
	}
	w := &cognitiveWalker{name: name}
	w.walkBlock(body, 0)
	return w.score
}

type cognitiveWalker struct {
	name  string
	score int
}

func (w *cognitiveWalker) walkBlock(b *ast.BlockStmt, nesting int) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		w.walkStmt(s, nesting)
	}
}

func (w *cognitiveWalker) walkStmt(s ast.Stmt, nesting int) {
	if s == nil {
		return
	}
	if w.walkControlFlow(s, nesting) {
		return
	}
	w.walkLeafStmt(s, nesting)
}

// walkControlFlow handles statements that increment cognitive score.
// Returns true if s was handled.
func (w *cognitiveWalker) walkControlFlow(s ast.Stmt, nesting int) bool {
	switch n := s.(type) {
	case *ast.IfStmt:
		w.walkIf(n, nesting)
	case *ast.ForStmt:
		w.score += 1 + nesting
		w.walkExpr(n.Cond)
		w.walkBlock(n.Body, nesting+1)
	case *ast.RangeStmt:
		w.score += 1 + nesting
		w.walkBlock(n.Body, nesting+1)
	case *ast.SwitchStmt:
		w.score += 1 + nesting
		w.walkExpr(n.Tag)
		w.walkCaseList(n.Body, nesting+1)
	case *ast.TypeSwitchStmt:
		w.score += 1 + nesting
		w.walkCaseList(n.Body, nesting+1)
	case *ast.SelectStmt:
		w.score += 1 + nesting
		w.walkCommList(n.Body, nesting+1)
	case *ast.BranchStmt:
		if n.Tok == token.GOTO || n.Label != nil {
			w.score++
		}
	default:
		return false
	}
	return true
}

// walkLeafStmt recurses through statements that don't increment the score.
func (w *cognitiveWalker) walkLeafStmt(s ast.Stmt, nesting int) {
	switch n := s.(type) {
	case *ast.BlockStmt:
		w.walkBlock(n, nesting)
	case *ast.LabeledStmt:
		w.walkStmt(n.Stmt, nesting)
	case *ast.DeferStmt:
		w.walkExpr(n.Call)
	case *ast.GoStmt:
		w.walkExpr(n.Call)
	case *ast.ExprStmt:
		w.walkExpr(n.X)
	case *ast.AssignStmt:
		for _, e := range n.Rhs {
			w.walkExpr(e)
		}
	case *ast.ReturnStmt:
		for _, e := range n.Results {
			w.walkExpr(e)
		}
	case *ast.IncDecStmt:
		w.walkExpr(n.X)
	case *ast.SendStmt:
		w.walkExpr(n.Value)
	}
}

func (w *cognitiveWalker) walkIf(n *ast.IfStmt, nesting int) {
	w.score += 1 + nesting
	w.walkExpr(n.Cond)
	w.walkBlock(n.Body, nesting+1)
	switch e := n.Else.(type) {
	case *ast.BlockStmt:
		w.score++ // else: +1 (no nesting bonus)
		w.walkBlock(e, nesting+1)
	case *ast.IfStmt:
		w.score++ // else-if: +1 (no nesting bonus)
		w.walkExpr(e.Cond)
		w.walkBlock(e.Body, nesting+1)
		w.walkElseChain(e.Else, nesting)
	}
}

func (w *cognitiveWalker) walkCaseList(body *ast.BlockStmt, nesting int) {
	if body == nil {
		return
	}
	for _, c := range body.List {
		cc, ok := c.(*ast.CaseClause)
		if !ok {
			continue
		}
		for _, st := range cc.Body {
			w.walkStmt(st, nesting)
		}
	}
}

func (w *cognitiveWalker) walkCommList(body *ast.BlockStmt, nesting int) {
	if body == nil {
		return
	}
	for _, c := range body.List {
		cc, ok := c.(*ast.CommClause)
		if !ok {
			continue
		}
		for _, st := range cc.Body {
			w.walkStmt(st, nesting)
		}
	}
}

func (w *cognitiveWalker) walkElseChain(s ast.Stmt, nesting int) {
	switch e := s.(type) {
	case *ast.BlockStmt:
		w.score++
		w.walkBlock(e, nesting+1)
	case *ast.IfStmt:
		w.score++
		w.walkExpr(e.Cond)
		w.walkBlock(e.Body, nesting+1)
		w.walkElseChain(e.Else, nesting)
	}
}

func (w *cognitiveWalker) walkExpr(e ast.Expr) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			w.scoreBoolChain(n)
			return
		}
		w.walkExpr(n.X)
		w.walkExpr(n.Y)
	case *ast.ParenExpr:
		w.walkExpr(n.X)
	case *ast.UnaryExpr:
		w.walkExpr(n.X)
	case *ast.CallExpr:
		// Recursion detection: direct call to enclosing function name.
		if w.name != "" {
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == w.name {
				w.score++
			}
		}
		w.walkExpr(n.Fun)
		for _, a := range n.Args {
			w.walkExpr(a)
		}
	case *ast.FuncLit:
		// Function literals contribute to the enclosing total but reset
		// nesting for the lambda body.
		w.walkBlock(n.Body, 0)
	case *ast.IndexExpr:
		w.walkExpr(n.X)
		w.walkExpr(n.Index)
	case *ast.SelectorExpr:
		w.walkExpr(n.X)
	}
}

// scoreBoolChain walks a chain of && / || and adds one increment per group
// of consecutive identical operators. A flat run of `a && b && c` counts as
// one. An alternation `a && b || c` counts as two.
func (w *cognitiveWalker) scoreBoolChain(root *ast.BinaryExpr) {
	var ops []token.Token
	var collect func(e ast.Expr)
	collect = func(e ast.Expr) {
		if be, ok := e.(*ast.BinaryExpr); ok && (be.Op == token.LAND || be.Op == token.LOR) {
			collect(be.X)
			ops = append(ops, be.Op)
			collect(be.Y)
			return
		}
		// recurse into the operand for nested control flow / calls
		w.walkExpr(e)
	}
	collect(root)
	if len(ops) == 0 {
		return
	}
	w.score++
	for i := 1; i < len(ops); i++ {
		if ops[i] != ops[i-1] {
			w.score++
		}
	}
}
