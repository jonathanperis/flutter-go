// Command apiparity measures how much of Flutter's public Dart API flutter-go
// exposes: top-level declarations, their members, and the parameters of
// every constructor, method and function.
//
// The Dart side is the Dart analyzer's view of dart:ui and package:flutter's
// libraries (dart/bin/dump.dart, run with the Flutter checkout's own Dart
// SDK): each library's export namespace, each public class's, mixin's,
// enum's and extension's declared members, and every parameter. A
// declaration belongs to the library whose src/<lib>/ directory declares it
// (re-exports count where they are declared); a third-party package a library
// re-exports (reexportedPackages: foundation's package:listen) counts for
// that library.
//
// The Go side is the repository type-checked with go/packages. Each Dart
// library maps to the Go package beside its Dart sources (libraryEntry), and
// each name maps to Go by the rules of src/docs/API_MAPPING.md (match.go).
// What the rules don't reach counts as exposed only through an entry of the
// divergence table, src/docs/api_divergences.json, each citing the
// API_MAPPING section that documents it; those are listed separately. A
// declaration or member is tested when a _test.go file anywhere in the
// repository refers to its Go declaration (resolved by go/types); for test
// packages that don't type-check, a mention of the name counts.
//
// Usage:
//
//	go run ./src/cmd/apiparity -flutter ~/Projetos/jonathanperis/flutter -o src/docs/PARITY.md
//
// writes PARITY.md and the per-library worklists in src/docs/parity/.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

// Libraries in dependency order (lowest first), as Flutter layers them.
var libraries = []string{
	"dart:ui", "foundation", "physics", "scheduler", "semantics", "animation",
	"services", "gestures", "painting", "rendering", "widgets",
	"cupertino", "material",
}

// reexportedPackages are the third-party packages whose re-exported names
// are counted, with the library re-exporting them and the directory of their
// Go port (API_MAPPING "Libraries").
var reexportedPackages = map[string]struct{ library, dir string }{
	"listen": {"foundation", "src/pub/listen"},
}

// Library is one Dart library's results.
type Library struct {
	Name  string  `json:"name"`
	GoPkg string  `json:"goPackage"`
	Decls []*Decl `json:"decls"`
	Counts
}

// Counts are a library's totals.
type Counts struct {
	Declarations     int `json:"declarations"`
	Exposed          int `json:"exposed"`
	Tested           int `json:"tested"`
	Members          int `json:"members"`
	MembersExposed   int `json:"membersExposed"`
	MembersTested    int `json:"membersTested"`
	Params           int `json:"params"`
	ParamsExposed    int `json:"paramsExposed"`
	Divergent        int `json:"divergent"` // declarations, members and parameters exposed through the divergence table
	ParamsDivergent  int `json:"paramsDivergent"`
	MembersDivergent int `json:"membersDivergent"`
}

func (c *Counts) add(o Counts) {
	c.Declarations += o.Declarations
	c.Exposed += o.Exposed
	c.Tested += o.Tested
	c.Members += o.Members
	c.MembersExposed += o.MembersExposed
	c.MembersTested += o.MembersTested
	c.Params += o.Params
	c.ParamsExposed += o.ParamsExposed
	c.Divergent += o.Divergent
	c.ParamsDivergent += o.ParamsDivergent
	c.MembersDivergent += o.MembersDivergent
}

func main() {
	flutter := flag.String("flutter", os.Getenv("FLUTTER_SRC"), "Flutter checkout (the repo root)")
	out := flag.String("o", "", "write the Markdown summary here (default stdout); the per-library worklists go to parity/ beside it")
	jsonOut := flag.String("json", "", "also write the full results as JSON")
	goRoot := flag.String("go", ".", "flutter-go repository root")
	dartIn := flag.String("dart-json", "", "read the Dart API from this dump.dart output instead of running it")
	dartOut := flag.String("dart-json-out", "", "save dump.dart's output here")
	flag.Parse()
	if *flutter == "" && *dartIn == "" {
		fatalf("set -flutter or FLUTTER_SRC to a Flutter checkout")
	}
	var dump []byte
	var err error
	if *dartIn != "" {
		dump, err = os.ReadFile(*dartIn)
	} else {
		abs, _ := filepath.Abs(*flutter)
		dump, err = runDartDump(*goRoot, abs, libraries)
	}
	if err != nil {
		fatalf("%v", err)
	}
	if *dartOut != "" {
		if err := os.WriteFile(*dartOut, dump, 0o644); err != nil {
			fatalf("%v", err)
		}
	}
	dartLibs, err := readDartDump(dump)
	if err != nil {
		fatalf("%v", err)
	}
	div, err := loadDivergences(filepath.Join(*goRoot, "src/docs/api_divergences.json"), filepath.Join(*goRoot, "src/docs/API_MAPPING.md"))
	if err != nil {
		fatalf("%v", err)
	}
	ix, err := loadGo(*goRoot, "./...")
	if err != nil {
		fatalf("%v", err)
	}
	libs, errs := measure(dartLibs, ix, div, libraryGoDir)
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, "apiparity:", e)
	}
	if len(errs) > 0 {
		os.Exit(1)
	}
	if len(ix.mentioned) > 0 {
		fmt.Fprintf(os.Stderr, "apiparity: some test packages did not type-check; their %d identifiers count by name\n", len(ix.mentioned))
	}
	summary := markdown(libs)
	if *out == "" {
		fmt.Print(summary)
	} else {
		if err := os.WriteFile(*out, []byte(summary), 0o644); err != nil {
			fatalf("%v", err)
		}
		dir := filepath.Join(filepath.Dir(*out), "parity")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatalf("%v", err)
		}
		for _, l := range libs {
			if err := os.WriteFile(filepath.Join(dir, detailFile(l.Name)), []byte(detail(l, div)), 0o644); err != nil {
				fatalf("%v", err)
			}
		}
	}
	if *jsonOut != "" {
		b, _ := json.MarshalIndent(libs, "", " ")
		if err := os.WriteFile(*jsonOut, append(b, '\n'), 0o644); err != nil {
			fatalf("%v", err)
		}
	}
}

// measure matches every library and returns the results, with the
// divergence table's problems (stale, unneeded or unused entries).
func measure(dartLibs []DartLibrary, ix *goIndex, div *divergenceTable, goDir func(lib string) string) ([]*Library, []error) {
	used := map[*Divergence]bool{}
	var errs []error
	var libs []*Library
	for _, dl := range dartLibs {
		dir := goDir(dl.Name)
		lib := &Library{Name: dl.Name, GoPkg: dir}
		pkg := ix.pkgs[dir]
		if pkg == nil {
			pkg = types.NewPackage(dir, filepath.Base(dir)) // not started: everything is missing
		}
		m := &matcher{pkg: pkg, ix: ix, div: div, lib: dl.Name, used: used}
		for _, d := range dl.Decls {
			lib.Decls = append(lib.Decls, m.matchDecl(d))
		}
		lib.Counts = count(lib.Decls)
		errs = append(errs, m.errs...)
		libs = append(libs, lib)
	}
	if div != nil {
		for _, e := range div.entries {
			if !used[e] {
				errs = append(errs, fmt.Errorf("divergence %s %s: no such Dart declaration, member or parameter", e.Library, e.Dart))
			}
		}
	}
	return libs, errs
}

func count(decls []*Decl) Counts {
	var c Counts
	params := func(ps []*Param) {
		for _, p := range ps {
			c.Params++
			if p.Exposed {
				c.ParamsExposed++
			}
			if p.Divergence != nil {
				c.ParamsDivergent++
				c.Divergent++
			}
		}
	}
	for _, d := range decls {
		c.Declarations++
		if d.Exposed {
			c.Exposed++
		}
		if d.Tested {
			c.Tested++
		}
		if d.Divergence != nil {
			c.Divergent++
		}
		params(d.Params)
		for _, m := range d.Members {
			c.Members++
			if m.Exposed {
				c.MembersExposed++
			}
			if m.Tested {
				c.MembersTested++
			}
			if m.Divergence != nil {
				c.MembersDivergent++
				c.Divergent++
			}
			params(m.Params)
		}
	}
	return c
}

func fatalf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "apiparity: "+f+"\n", a...)
	os.Exit(1)
}

// libraryGoDir returns the directory of a library's Go package. Go files
// sit beside the Dart files they replace; dart:ui's go in a go/
// subdirectory, because Go can't build a package in a directory holding C++
// sources.
func libraryGoDir(name string) string {
	if name == "dart:ui" {
		return "engine/src/flutter/lib/ui/go"
	}
	return "packages/flutter/lib/src/" + name
}

// detailFile is a library's worklist file name (dart:ui → dart_ui.md).
func detailFile(lib string) string { return strings.ReplaceAll(lib, ":", "_") + ".md" }
