// Command apiparity measures how much of Flutter's public Dart API flutter-go
// exposes.
//
// It reads Flutter's public libraries (package:flutter/<lib>.dart and dart:ui)
// from a Flutter checkout, follows their `export` and `part` directives, and
// lists every public top-level declaration: classes, mixins, enums, extension
// types, typedefs, functions, getters, constants and variables. Each Dart
// library maps to the Go package beside its Dart sources (see libraryEntry), and
// a declaration counts as exposed when that package declares the same name
// with its first letter capitalized. A declaration counts as tested when a
// _test.go file in the package or in difftest/ mentions it.
//
// Usage:
//
//	go run ./flutter_go/cmd/apiparity -flutter ~/Projetos/jonathanperis/flutter [-o flutter_go/docs/PARITY.md] [-json parity.json]
//
// The measure is deliberately strict and coarse: it counts top-level names
// only (not members or parameters), and a name counts only in the package that
// mirrors its Dart library.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Libraries in dependency order (lowest first), as Flutter layers them.
var libraries = []string{
	"dart:ui", "foundation", "physics", "scheduler", "semantics", "animation",
	"services", "gestures", "painting", "rendering", "widgets",
	"cupertino", "material",
}

type Decl struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Exposed bool   `json:"exposed"`
	Tested  bool   `json:"tested"`
}

type Library struct {
	Name    string `json:"name"`
	GoPkg   string `json:"goPackage"`
	Decls   []Decl `json:"decls"`
	Exposed int    `json:"exposed"`
	Tested  int    `json:"tested"`
}

func main() {
	flutter := flag.String("flutter", os.Getenv("FLUTTER_SRC"), "Flutter checkout (the repo root)")
	out := flag.String("o", "", "write the Markdown report here (default stdout)")
	jsonOut := flag.String("json", "", "also write the full inventory as JSON")
	goRoot := flag.String("go", ".", "flutter-go repository root")
	flag.Parse()
	if *flutter == "" {
		fatalf("set -flutter or FLUTTER_SRC to a Flutter checkout")
	}
	mentions := testMentions(*goRoot)
	var libs []*Library
	for _, name := range libraries {
		entry, pkg := libraryEntry(*flutter, name)
		decls := inventory(entry, *flutter)
		goNames := goExported(filepath.Join(*goRoot, pkg))
		lib := &Library{Name: name, GoPkg: pkg}
		for _, d := range decls {
			g := goName(d.Name)
			d.Exposed = goNames[g]
			d.Tested = d.Exposed && (mentions[pkg][g] || mentions["flutter_go/difftest"][g])
			if d.Exposed {
				lib.Exposed++
			}
			if d.Tested {
				lib.Tested++
			}
			lib.Decls = append(lib.Decls, d)
		}
		libs = append(libs, lib)
	}
	report := markdown(libs)
	if *out == "" {
		fmt.Print(report)
	} else if err := os.WriteFile(*out, []byte(report), 0o644); err != nil {
		fatalf("%v", err)
	}
	if *jsonOut != "" {
		b, _ := json.MarshalIndent(libs, "", " ")
		if err := os.WriteFile(*jsonOut, append(b, '\n'), 0o644); err != nil {
			fatalf("%v", err)
		}
	}
}

func fatalf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "apiparity: "+f+"\n", a...)
	os.Exit(1)
}

// libraryEntry returns a library's Dart entry file in the Flutter checkout
// and the directory of its Go package in this repository. Go files sit
// beside the Dart files they replace; dart:ui's go in a go/ subdirectory,
// because Go can't build a package in a directory holding C++ sources.
func libraryEntry(root, name string) (file, goDir string) {
	if name == "dart:ui" {
		return filepath.Join(root, "engine/src/flutter/lib/ui/ui.dart"), "engine/src/flutter/lib/ui/go"
	}
	return filepath.Join(root, "packages/flutter/lib", name+".dart"), "packages/flutter/lib/src/" + name
}

// goName maps a Dart identifier to the exported Go identifier.
func goName(dart string) string {
	r := []rune(strings.TrimPrefix(dart, "$"))
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// ---- Dart side ----

var (
	directiveRe = regexp.MustCompile(`(?s)^(export|part)\s+'([^']+)'(.*)$`)
	showHideRe  = regexp.MustCompile(`\b(show|hide)\b([^;]*)`)
	typeDeclRe  = regexp.MustCompile(`^(?:(?:abstract|base|final|sealed|interface|mixin)\s+)*(class|mixin|enum|typedef|extension type(?: const)?|extension)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	keywordRe   = regexp.MustCompile(`^(import|export|part|library|@|//|/\*|\*|}|\)|]|if|for|return)`)
	funcDeclRe  = regexp.MustCompile(`^(?:external\s+)?(?:[A-Za-z_$][\w$<>,?. ]*\s+)?(get\s+)?([A-Za-z_$][A-Za-z0-9_$]*)\s*(<[^>]*>)?\s*(\(|=>|\{)`)
	varDeclRe   = regexp.MustCompile(`^(?:const|final|var|late|external)\b[^=;(]*?\b([A-Za-z_$][A-Za-z0-9_$]*)\s*(=|;)`)
	typedVarRe  = regexp.MustCompile(`^[A-Z][\w$<>,?. ]*\s+([a-zA-Z_$][A-Za-z0-9_$]*)\s*(=|;)`)
)

type filter struct {
	show map[string]bool // nil means everything
	hide map[string]bool
}

func (f filter) allows(name string) bool {
	if f.show != nil && !f.show[name] {
		return false
	}
	return !f.hide[name]
}

func (f filter) and(g filter) filter {
	out := filter{hide: map[string]bool{}}
	for k := range f.hide {
		out.hide[k] = true
	}
	for k := range g.hide {
		out.hide[k] = true
	}
	switch {
	case f.show == nil:
		out.show = g.show
	case g.show == nil:
		out.show = f.show
	default:
		out.show = map[string]bool{}
		for k := range f.show {
			if g.show[k] {
				out.show[k] = true
			}
		}
	}
	return out
}

func names(list string) map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Split(list, ",") {
		if n = strings.TrimSpace(n); n != "" {
			m[n] = true
		}
	}
	return m
}

// inventory returns the public declarations reachable from a library file
// through relative exports and parts. Exports of other packages
// (package:, dart:) are not followed: those names belong to their own library.
func inventory(entry, root string) []Decl {
	seen := map[string]bool{}
	var decls []Decl
	have := map[string]bool{}
	var visit func(path string, f filter)
	visit = func(path string, f filter) {
		key := path + fmt.Sprint(f)
		if seen[key] {
			return
		}
		seen[key] = true
		src, err := os.ReadFile(path)
		if err != nil {
			fatalf("%v", err)
		}
		rel, _ := filepath.Rel(root, path)
		for _, st := range directives(string(src)) {
			m := directiveRe.FindStringSubmatch(st)
			if m == nil || strings.Contains(m[2], ":") {
				continue
			}
			target := filepath.Join(filepath.Dir(path), m[2])
			if path != entry && filepath.Dir(target) == filepath.Dir(entry) {
				continue // another public library re-exported; counted there
			}
			if path == entry && filepath.Dir(m[2]) == "." && m[1] == "export" {
				continue
			}
			g := f
			if m[1] == "export" {
				for _, sh := range showHideRe.FindAllStringSubmatch(m[3], -1) {
					if sh[1] == "show" {
						g = g.and(filter{show: names(sh[2])})
					} else {
						g = g.and(filter{hide: names(sh[2])})
					}
				}
			}
			visit(target, g)
		}
		for _, d := range declarations(string(src)) {
			if strings.HasPrefix(d.Name, "_") || !f.allows(d.Name) || have[d.Name] {
				continue
			}
			have[d.Name] = true
			d.File = rel
			decls = append(decls, d)
		}
	}
	visit(entry, filter{})
	sort.Slice(decls, func(i, j int) bool { return decls[i].Name < decls[j].Name })
	return decls
}

// directives returns the export/part statements of a Dart file, joined onto
// one line each.
func directives(src string) []string {
	var out []string
	var cur strings.Builder
	in := false
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if !in && (strings.HasPrefix(t, "export ") || strings.HasPrefix(t, "part ")) && !strings.HasPrefix(t, "part of") {
			in = true
			cur.Reset()
		}
		if in {
			if i := strings.Index(t, "//"); i >= 0 {
				t = t[:i]
			}
			cur.WriteString(t + " ")
			if strings.Contains(t, ";") {
				out = append(out, strings.TrimSpace(cur.String()))
				in = false
			}
		}
	}
	return out
}

// declarations scans a Dart file's top level (brace depth 0) for declarations.
func declarations(src string) []Decl {
	var out []Decl
	depth := 0 // braces, parens and brackets
	inBlockComment := false
	inMultiString := ""
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		atTop := depth == 0 && !inBlockComment && inMultiString == ""
		if atTop && line != "" && line[0] != ' ' && line[0] != '\t' {
			if d, ok := declOnLine(strings.TrimSpace(line)); ok {
				out = append(out, d)
			}
		}
		depth, inBlockComment, inMultiString = scanDepth(line, depth, inBlockComment, inMultiString)
	}
	return out
}

func declOnLine(t string) (Decl, bool) {
	if m := typeDeclRe.FindStringSubmatch(t); m != nil {
		kind := m[1]
		if strings.HasPrefix(kind, "extension type") {
			kind = "extension type"
		}
		if kind == "extension" && m[2] == "on" { // unnamed extension
			return Decl{}, false
		}
		return Decl{Name: m[2], Kind: kind}, true
	}
	if keywordRe.MatchString(t) {
		return Decl{}, false
	}
	if m := varDeclRe.FindStringSubmatch(t); m != nil {
		return Decl{Name: m[1], Kind: "variable"}, true
	}
	if m := funcDeclRe.FindStringSubmatch(t); m != nil {
		if m[1] != "" {
			return Decl{Name: m[2], Kind: "getter"}, true
		}
		switch m[2] {
		case "get", "set", "operator", "const", "final", "var", "class", "enum":
			return Decl{}, false
		}
		return Decl{Name: m[2], Kind: "function"}, true
	}
	if m := typedVarRe.FindStringSubmatch(t); m != nil {
		return Decl{Name: m[1], Kind: "variable"}, true
	}
	return Decl{}, false
}

// scanDepth tracks bracket depth across a line, skipping comments and strings.
func scanDepth(line string, depth int, inBlock bool, inMulti string) (int, bool, string) {
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inBlock {
			if c == '*' && i+1 < len(line) && line[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if inMulti != "" {
			if strings.HasPrefix(line[i:], inMulti) {
				i += 2
				inMulti = ""
			} else if c == '\\' {
				i++
			}
			continue
		}
		switch {
		case c == '/' && i+1 < len(line) && line[i+1] == '/':
			return depth, inBlock, inMulti
		case c == '/' && i+1 < len(line) && line[i+1] == '*':
			inBlock = true
			i++
		case strings.HasPrefix(line[i:], `'''`) || strings.HasPrefix(line[i:], `"""`):
			inMulti = line[i : i+3]
			i += 2
		case c == '\'' || c == '"':
			raw := i > 0 && line[i-1] == 'r'
			for i++; i < len(line) && line[i] != c; i++ {
				if line[i] == '\\' && !raw {
					i++
				}
			}
		case c == '{' || c == '(' || c == '[':
			depth++
		case c == '}' || c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth, inBlock, inMulti
}

// ---- Go side ----

func goExported(dir string) map[string]bool {
	out := map[string]bool{}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return out
	}
	for _, p := range pkgs {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil && d.Name.IsExported() {
						out[d.Name.Name] = true
					}
				case *ast.GenDecl:
					for _, s := range d.Specs {
						switch s := s.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								out[s.Name.Name] = true
							}
						case *ast.ValueSpec:
							for _, n := range s.Names {
								if n.IsExported() {
									out[n.Name] = true
								}
							}
						}
					}
				}
			}
		}
	}
	return out
}

var identRe = regexp.MustCompile(`[A-Z][A-Za-z0-9_]*`)

// testMentions maps each Go package directory to the identifiers its test
// files mention.
func testMentions(root string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		top := filepath.ToSlash(filepath.Dir(rel)) // the test's package directory
		src, _ := os.ReadFile(path)
		if out[top] == nil {
			out[top] = map[string]bool{}
		}
		for _, id := range identRe.FindAllString(string(src), -1) {
			out[top][id] = true
		}
		return nil
	})
	return out
}

// ---- report ----

func markdown(libs []*Library) string {
	var b strings.Builder
	total, exposed, tested := 0, 0, 0
	for _, l := range libs {
		total += len(l.Decls)
		exposed += l.Exposed
		tested += l.Tested
	}
	b.WriteString("# API parity with Flutter\n\n")
	b.WriteString("Generated by `go run ./flutter_go/cmd/apiparity`; do not edit by hand.\n\n")
	b.WriteString("Counts public **top-level** declarations of each Flutter library (classes,\n")
	b.WriteString("enums, mixins, typedefs, functions, constants) and whether the Go package of\n")
	b.WriteString("the same name declares them. Members and parameters are not counted yet, so\n")
	b.WriteString("\"exposed\" is an upper bound on real parity. \"Tested\" means a Go test mentions\n")
	b.WriteString("the name.\n\n")
	fmt.Fprintf(&b, "**Total: %d / %d exposed (%.1f%%), %d tested.**\n\n", exposed, total, pct(exposed, total), tested)
	b.WriteString("| Flutter library | Go package directory | Declarations | Exposed | Tested |\n|---|---|---:|---:|---:|\n")
	for _, l := range libs {
		fmt.Fprintf(&b, "| `%s` | `%s` | %d | %d (%.0f%%) | %d |\n", l.Name, l.GoPkg, len(l.Decls), l.Exposed, pct(l.Exposed, len(l.Decls)), l.Tested)
	}
	for _, l := range libs {
		fmt.Fprintf(&b, "\n## `%s`\n\n", l.Name)
		var missing, done []string
		for _, d := range l.Decls {
			s := "`" + d.Name + "`"
			if d.Exposed {
				if !d.Tested {
					s += " (untested)"
				}
				done = append(done, s)
			} else {
				missing = append(missing, s)
			}
		}
		fmt.Fprintf(&b, "**Exposed (%d):** %s\n\n", len(done), list(done))
		fmt.Fprintf(&b, "**Missing (%d):** %s\n", len(missing), list(missing))
	}
	return b.String()
}

func list(s []string) string {
	if len(s) == 0 {
		return "none"
	}
	return strings.Join(s, ", ")
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
