package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

// The Go side: the repository's packages, type-checked with go/packages, and
// the declarations their tests use.

// goIndex holds the type-checked packages of the repository (by directory
// relative to its root) and what its _test.go files use.
type goIndex struct {
	fset *token.FileSet
	pkgs map[string]*types.Package // directory → package (non-test variant)
	// used holds the declaration positions (objKey) of every object a
	// _test.go file refers to, resolved by go/types.
	used map[string]bool
	// mentioned holds the identifiers of the test files of packages that
	// did not type-check (the fallback heuristic: a name mentioned there
	// counts as tested).
	mentioned map[string]bool
}

// loadGo type-checks the packages matching patterns (with their tests) in
// dir, the repository root.
func loadGo(dir string, patterns ...string) (*goIndex, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:   dir,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	ix := &goIndex{pkgs: map[string]*types.Package{}, used: map[string]bool{}, mentioned: map[string]bool{}}
	absRoot, _ := filepath.Abs(dir)
	for _, p := range pkgs {
		if ix.fset == nil {
			ix.fset = p.Fset
		}
		if p.Types == nil {
			continue
		}
		if p.ID == p.PkgPath && len(p.GoFiles) > 0 {
			rel, err := filepath.Rel(absRoot, filepath.Dir(p.GoFiles[0]))
			if err == nil && !strings.HasPrefix(rel, "..") {
				ix.pkgs[filepath.ToSlash(rel)] = p.Types
			}
		}
		ok := len(p.Errors) == 0 && !p.IllTyped
		for i, f := range p.Syntax {
			name := p.Fset.Position(f.Pos()).Filename
			if i < len(p.CompiledGoFiles) {
				name = p.CompiledGoFiles[i]
			}
			if !strings.HasSuffix(name, "_test.go") {
				continue
			}
			if !ok {
				src, _ := os.ReadFile(name)
				for _, id := range identRe.FindAllString(string(src), -1) {
					ix.mentioned[id] = true
				}
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if id, isID := n.(*ast.Ident); isID {
					if obj := p.TypesInfo.Uses[id]; obj != nil {
						ix.used[ix.objKey(obj)] = true
					}
				}
				return true
			})
		}
	}
	return ix, nil
}

var identRe = regexp.MustCompile(`[A-Z][A-Za-z0-9_]*`)

// objKey identifies a declaration across the package variants go/packages
// type-checks separately (a package and the same package with its in-package
// tests): its name and source position.
func (ix *goIndex) objKey(obj types.Object) string {
	if !obj.Pos().IsValid() {
		return obj.Name() // builtins and universe objects
	}
	return obj.Name() + "@" + ix.fset.Position(obj.Pos()).String()
}

// tested reports whether a test uses any of the objects.
func (ix *goIndex) tested(objs []types.Object) bool {
	for _, o := range objs {
		if ix.used[ix.objKey(o)] || ix.mentioned[o.Name()] {
			return true
		}
	}
	return false
}
