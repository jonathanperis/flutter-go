package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// The Dart side: dart/bin/dump.dart lists each library's public API with the
// Dart analyzer (export namespaces, show/hide and parts resolved), and this
// file reads its JSON.

// DartParam is a parameter of a constructor, method or function.
type DartParam struct {
	Name     string `json:"name"`
	Named    bool   `json:"named,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// DartMember is a public member a class, mixin, enum or extension declares.
// Kind is constructor (Name "" for the unnamed one), field, getter, setter,
// method, operator (Name is the operator, "unary-" for negation) or
// enumValue.
type DartMember struct {
	Name       string      `json:"name"`
	Kind       string      `json:"kind"`
	Static     bool        `json:"static,omitempty"`
	Settable   bool        `json:"settable,omitempty"` // a field with a setter (not final or const)
	Factory    bool        `json:"factory,omitempty"`
	Const      bool        `json:"const,omitempty"`
	Generic    bool        `json:"generic,omitempty"`
	Deprecated bool        `json:"deprecated,omitempty"`
	Params     []DartParam `json:"params,omitempty"`
}

// DartDecl is a public top-level declaration of a library.
type DartDecl struct {
	Name       string       `json:"name"`
	Kind       string       `json:"kind"` // class, mixin, enum, extension, extension type, typedef, function, getter, setter, variable
	File       string       `json:"file"`
	Package    string       `json:"package,omitempty"` // the third-party package re-exported, if any
	Abstract   bool         `json:"abstract,omitempty"`
	Sealed     bool         `json:"sealed,omitempty"`
	MixinClass bool         `json:"mixinClass,omitempty"`
	Generic    bool         `json:"generic,omitempty"`
	Settable   bool         `json:"settable,omitempty"`
	Deprecated bool         `json:"deprecated,omitempty"`
	Members    []DartMember `json:"members,omitempty"`
	Params     []DartParam  `json:"params,omitempty"`
}

// DartLibrary is one public library's declarations, sorted by name.
type DartLibrary struct {
	Name  string     `json:"name"`
	Decls []DartDecl `json:"decls"`
}

type dartDump struct {
	Libraries []DartLibrary `json:"libraries"`
}

// readDartDump reads dump.dart's JSON.
func readDartDump(b []byte) ([]DartLibrary, error) {
	var d dartDump
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	for _, l := range d.Libraries {
		sort.SliceStable(l.Decls, func(i, j int) bool { return l.Decls[i].Name < l.Decls[j].Name })
	}
	return d.Libraries, nil
}

// runDartDump runs dump.dart with the Flutter checkout's own Dart SDK. The
// tool's pub dependencies (the analyzer, and package:flutter itself) are
// fetched from the pub cache on first use.
func runDartDump(goRoot, flutter string, libs []string) ([]byte, error) {
	dir := filepath.Join(goRoot, "src/cmd/apiparity/dart")
	if _, err := os.Stat(filepath.Join(dir, ".dart_tool/package_config.json")); err != nil {
		cmd := exec.Command(filepath.Join(flutter, "bin/flutter"), "pub", "get", "--offline")
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("flutter pub get in %s: %v", dir, err)
		}
	}
	args := []string{"run", "bin/dump.dart", flutter}
	args = append(args, libs...)
	var pkgs []string
	for p := range reexportedPackages {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, p := range pkgs {
		args = append(args, "--reexport", p+"="+reexportedPackages[p].library)
	}
	cmd := exec.Command(filepath.Join(flutter, "bin/dart"), args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dump.dart: %v", err)
	}
	return out.Bytes(), nil
}
