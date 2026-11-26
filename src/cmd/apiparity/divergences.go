package main

import (
	"encoding/json"
	"fmt"
	"go/types"
	"os"
	"regexp"
	"strings"
)

// Divergence is an entry of the divergence table (src/docs/api_divergences.json):
// a Dart declaration, member or parameter whose Go form API_MAPPING.md
// documents but the matcher's general rules don't reach.
type Divergence struct {
	Library string `json:"library"`
	// Dart names the declaration as the reports do: Foo, Foo.bar,
	// Foo.new (the unnamed constructor), Foo.bar= (a setter), Foo.==, and
	// a parameter as Foo.bar(param) or foo(param).
	Dart string `json:"dart"`
	// Kind is "rename" (Go has it under another name: Go must resolve) or
	// "impossible" (Go can't express it; Go may name what stands for it).
	Kind string `json:"kind"`
	// Go is the Go identifier: Name or Type.Member in the library's
	// package, or <directory>:Name for another package. For a parameter,
	// the Go parameter or option field (not resolved).
	Go string `json:"go,omitempty"`
	// Rule cites the API_MAPPING.md section (## heading) documenting it,
	// optionally followed by ": " and the item within it.
	Rule string `json:"rule"`
	Note string `json:"note,omitempty"`
}

type divergenceTable struct {
	entries []*Divergence
	byKey   map[string]*Divergence
}

func (t *divergenceTable) find(lib, dart string) *Divergence {
	if t == nil {
		return nil
	}
	return t.byKey[lib+" "+dart]
}

var headingRe = regexp.MustCompile(`(?m)^## (.+)$`)

// loadDivergences reads the table and checks every entry: a known kind, and
// a rule naming a section of API_MAPPING.md.
func loadDivergences(tablePath, mappingPath string) (*divergenceTable, error) {
	b, err := os.ReadFile(tablePath)
	if err != nil {
		return nil, err
	}
	mapping, err := os.ReadFile(mappingPath)
	if err != nil {
		return nil, err
	}
	return parseDivergences(b, mapping)
}

func parseDivergences(table, mapping []byte) (*divergenceTable, error) {
	var entries []*Divergence
	if err := json.Unmarshal(table, &entries); err != nil {
		return nil, fmt.Errorf("divergence table: %v", err)
	}
	sections := map[string]bool{}
	for _, m := range headingRe.FindAllStringSubmatch(string(mapping), -1) {
		sections[strings.TrimSpace(m[1])] = true
	}
	t := &divergenceTable{entries: entries, byKey: map[string]*Divergence{}}
	for _, e := range entries {
		key := e.Library + " " + e.Dart
		switch {
		case e.Library == "" || e.Dart == "":
			return nil, fmt.Errorf("divergence %+v: library and dart are required", *e)
		case e.Kind != "rename" && e.Kind != "impossible":
			return nil, fmt.Errorf("divergence %s: kind must be rename or impossible, not %q", key, e.Kind)
		case e.Kind == "rename" && e.Go == "":
			return nil, fmt.Errorf("divergence %s: a rename names its Go identifier", key)
		case t.byKey[key] != nil:
			return nil, fmt.Errorf("divergence %s: listed twice", key)
		}
		section, _, _ := strings.Cut(e.Rule, ": ")
		if !sections[section] {
			return nil, fmt.Errorf("divergence %s: rule %q does not name a section of API_MAPPING.md", key, e.Rule)
		}
		t.byKey[key] = e
	}
	return t, nil
}

// isParamKey reports whether a divergence names a parameter.
func isParamKey(dart string) bool { return strings.HasSuffix(dart, ")") }

// applyDivergence resolves a divergence's Go identifier. A rename must
// resolve (the tool fails otherwise, so the table can't go stale); an
// impossibility resolves when it names a Go stand-in.
func (m *matcher) applyDivergence(dv *Divergence) []types.Object {
	m.used[dv] = true
	if dv.Go == "" {
		return nil
	}
	objs := m.resolve(dv.Go)
	if len(objs) == 0 {
		m.errs = append(m.errs, fmt.Errorf("divergence %s %s: Go identifier %s not found", dv.Library, dv.Dart, dv.Go))
	}
	return objs
}

// resolve finds a Go identifier: Name or Type.Member, in the library's
// package or, with a <directory>: prefix, another package of the repository.
func (m *matcher) resolve(id string) []types.Object {
	pkg := m.pkg
	if dir, rest, ok := strings.Cut(id, ":"); ok {
		if m.ix == nil || m.ix.pkgs[dir] == nil {
			return nil
		}
		pkg, id = m.ix.pkgs[dir], rest
	}
	typ, member, isMember := strings.Cut(id, ".")
	obj := pkg.Scope().Lookup(typ)
	if obj == nil {
		return nil
	}
	if !isMember {
		return []types.Object{obj}
	}
	o, _, _ := types.LookupFieldOrMethod(obj.Type(), true, pkg, member)
	if o == nil {
		return nil
	}
	return []types.Object{o}
}

// unneeded records an entry the general rules already cover: the table
// lists only what they don't.
func (m *matcher) unneeded(dv *Divergence) {
	m.used[dv] = true
	m.errs = append(m.errs, fmt.Errorf("divergence %s %s: not needed, the mapping rules already find it", dv.Library, dv.Dart))
}
