// Command fmacheck lists floating-point expressions that Go may compile to
// a fused multiply-add: `a*b + c`, `c + a*b`, `a*b - c` and `c - a*b`.
//
// Go's spec lets the compiler fuse these (and it does on arm64), rounding
// once instead of twice. Dart never fuses, so ported code that must match
// Flutter bit for bit wraps the product in an explicit conversion,
// `float64(a*b) + c`, which the spec says prevents fusion.
//
//	go run ./src/cmd/fmacheck [-fix] ./packages/... ./engine/src/flutter/lib/ui/go
//
// It exits with status 1 when it finds any. With -fix it wraps each fusable
// product in a conversion to its own type instead.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	fix := flag.Bool("fix", false, "wrap fusable products in explicit conversions")
	flag.Parse()
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	found := 0
	type insert struct {
		off  int
		text string
		end  bool
	}
	edits := map[string][]insert{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			name := p.Fset.Position(f.Pos()).Filename
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				// A product stored in a variable can still fuse with a later
				// add or subtract of that variable (the compiler's SSA form
				// has no statement boundaries): `t *= n; return t - trunc(t)`
				// became one multiply-subtract.
				var stored []ast.Expr
				switch s := n.(type) {
				case *ast.AssignStmt:
					if s.Tok == token.MUL_ASSIGN && len(s.Lhs) == 1 && isFloat(p.TypesInfo, s.Lhs[0]) {
						found++
						fmt.Printf("%s: %s stores an unrounded product (write v = float64(v * x))\n",
							p.Fset.Position(s.Pos()), types.ExprString(s.Lhs[0])+" *= "+types.ExprString(s.Rhs[0]))
					}
					if s.Tok == token.ASSIGN || s.Tok == token.DEFINE {
						stored = s.Rhs
					}
				case *ast.ValueSpec:
					stored = s.Values
				}
				for _, rhs := range stored {
					if !isMul(p.TypesInfo, rhs) {
						continue
					}
					found++
					fmt.Printf("%s: %s is stored unrounded and may fuse with a later add\n",
						p.Fset.Position(rhs.Pos()), types.ExprString(rhs))
					typ := types.TypeString(p.TypesInfo.Types[rhs].Type.Underlying(), nil)
					edits[name] = append(edits[name],
						insert{p.Fset.Position(rhs.Pos()).Offset, typ + "(", false},
						insert{p.Fset.Position(rhs.End()).Offset, ")", true})
				}
				if be, ok := n.(*ast.BinaryExpr); ok && isFloatConstOp(p.TypesInfo, be) {
					fmt.Printf("%s: %s is a constant expression Go evaluates exactly; Dart rounds each step\n",
						p.Fset.Position(be.Pos()), types.ExprString(be))
					found++
					return false
				}
				be, ok := n.(*ast.BinaryExpr)
				if !ok || (be.Op != token.ADD && be.Op != token.SUB) || !isFloat(p.TypesInfo, be) {
					return true
				}
				for _, side := range []ast.Expr{be.X, be.Y} {
					if !isMul(p.TypesInfo, side) {
						continue
					}
					found++
					fmt.Printf("%s: %s may fuse into a multiply-add\n", p.Fset.Position(be.Pos()), types.ExprString(be))
					typ := types.TypeString(p.TypesInfo.Types[side].Type.Underlying(), nil)
					edits[name] = append(edits[name],
						insert{p.Fset.Position(side.Pos()).Offset, typ + "(", false},
						insert{p.Fset.Position(side.End()).Offset, ")", true})
				}
				return true
			})
		}
	}
	if *fix {
		for file, ins := range edits {
			src, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			// Apply from the end so offsets stay valid; at equal offsets put
			// closing parens before openings.
			sort.SliceStable(ins, func(i, j int) bool {
				if ins[i].off != ins[j].off {
					return ins[i].off > ins[j].off
				}
				return !ins[i].end && ins[j].end
			})
			for _, e := range ins {
				src = append(src[:e.off], append([]byte(e.text), src[e.off:]...)...)
			}
			if err := os.WriteFile(file, src, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		return
	}
	if found > 0 {
		os.Exit(1)
	}
}

// isFloatConstOp reports whether e is an arithmetic constant expression of
// float type whose exact value differs from what float64 arithmetic gives.
func isFloatConstOp(info *types.Info, e *ast.BinaryExpr) bool {
	switch e.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO:
	default:
		return false
	}
	tv, ok := info.Types[e]
	if !ok || tv.Value == nil {
		return false
	}
	if k := tv.Value.Kind(); k != constant.Float && k != constant.Int {
		return false // e.g. an untyped string constant "a" + "b"
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	if !ok || b.Info()&(types.IsFloat|types.IsUntyped) == 0 || b.Info()&types.IsInteger != 0 {
		return false
	}
	return !exactInFloat64(info, e)
}

// exactInFloat64 reports whether evaluating e step by step in float64
// gives the same value as Go's exact constant arithmetic.
func exactInFloat64(info *types.Info, e ast.Expr) bool {
	tv := info.Types[e]
	exact, _ := constant.Float64Val(tv.Value)
	var stepwise float64
	var walk func(ast.Expr) float64
	walk = func(x ast.Expr) float64 {
		switch v := x.(type) {
		case *ast.ParenExpr:
			return walk(v.X)
		case *ast.BinaryExpr:
			l, r := walk(v.X), walk(v.Y)
			switch v.Op {
			case token.ADD:
				return l + r
			case token.SUB:
				return l - r
			case token.MUL:
				return l * r
			case token.QUO:
				return l / r
			}
		case *ast.UnaryExpr:
			if v.Op == token.SUB {
				return -walk(v.X)
			}
		}
		f, _ := constant.Float64Val(info.Types[x].Value)
		return f
	}
	stepwise = walk(e)
	return stepwise == exact
}

func isFloat(info *types.Info, e ast.Expr) bool {
	tv, ok := info.Types[e]
	if !ok || tv.Value != nil { // constant expressions are evaluated exactly
		return false
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsFloat != 0
}

// isMul reports whether e is a (parenthesized) non-constant float product.
func isMul(info *types.Info, e ast.Expr) bool {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			break
		}
		e = p.X
	}
	be, ok := e.(*ast.BinaryExpr)
	return ok && be.Op == token.MUL && isFloat(info, be)
}
