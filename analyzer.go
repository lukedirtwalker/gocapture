package main

// Code here is heavily inspired by golang tools loopclosure.
// (see https://github.com/golang/tools/blob/master/go/analysis/passes/loopclosure/loopclosure.go)

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "gocapture",
	Doc:      "check for problematic variable captures in for loops",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var loopFilter = []ast.Node{
	(*ast.RangeStmt)(nil),
	(*ast.ForStmt)(nil),
}

type analyzer struct {
	pass *analysis.Pass
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	a := &analyzer{pass: pass}
	inspect.Preorder(loopFilter, a.processLoop)
	return nil, nil
}

func (a *analyzer) processLoop(n ast.Node) {
	body, vars := extractBodyAndVars(n)
	if vars == nil {
		return
	}
	a.analyzeBody(body, vars)
}

// extractBodyAndVars extracts the body of the loop and also the variables it is updating.
func extractBodyAndVars(n ast.Node) (*ast.BlockStmt, map[*ast.Object]struct{}) {
	vars := make(map[*ast.Object]struct{})
	var body *ast.BlockStmt
	switch n := n.(type) {
	case *ast.RangeStmt:
		body = n.Body
		addRefObj(n.Key, vars)
		addRefObj(n.Value, vars)
	case *ast.ForStmt:
		body = n.Body
		// check post iteration statement for assignment or inc/dec.
		switch p := n.Post.(type) {
		case *ast.AssignStmt:
			for i := range p.Lhs {
				lhs := p.Lhs[i]
				addRefObj(lhs, vars)
			}
		case *ast.IncDecStmt:
			addRefObj(p.X, vars)
		}
	}
	return body, vars
}

func addRefObj(expr ast.Expr, vars map[*ast.Object]struct{}) {
	if id, ok := expr.(*ast.Ident); ok {
		vars[id.Obj] = struct{}{}
	}
}

func (a *analyzer) analyzeBody(body *ast.BlockStmt, vars map[*ast.Object]struct{}) {
	if len(body.List) == 0 {
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.GoStmt:
			a.analyzeCall(s.Call, vars)
		case *ast.DeferStmt:
			a.analyzeCall(s.Call, vars)
		}
		return true
	})
}

func (a *analyzer) analyzeCall(call *ast.CallExpr, vars map[*ast.Object]struct{}) {
	funcLit, ok := call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Obj == nil {
			return true
		}
		if _, captured := vars[id.Obj]; captured {
			a.pass.Reportf(id.Pos(), "'%s' is captured by func literal!", id.Name)
		}
		return true
	})
}
