package main

import (
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// The matcher is tested on a small Dart library (testdata/fixture_dart.json,
// as dump.dart would print it) and its Go "port" (testdata/fixture, which
// imports testdata/port, a re-exported package's port). Each case is one
// mapping rule of API_MAPPING.md.

const fixtureDir = "src/cmd/apiparity/testdata/fixture"

// The API_MAPPING sections the fixture's divergence table cites.
const fixtureMapping = "# x\n## Declarations\n## Libraries\n## Types\n## Operators\n"

const fixtureTable = `[
 {"library": "fixture", "dart": "thing", "kind": "rename", "go": "ThingInstance", "rule": "Declarations"},
 {"library": "fixture", "dart": "Listenable.merge", "kind": "rename", "go": "ListenableMerge", "rule": "Libraries: wrappers"},
 {"library": "fixture", "dart": "runIt(d)", "kind": "impossible", "rule": "Types", "note": "no Go form"},
 {"library": "fixture", "dart": "Point.*", "kind": "impossible", "go": "src/cmd/apiparity/testdata/port:AliasOnError", "rule": "Operators"},
 {"library": "fixture", "dart": "Box.[]", "kind": "rename", "go": "Box.Get", "rule": "Operators"}
]`

var (
	fixtureOnce sync.Once
	fixtureIx   *goIndex
	fixtureErr  error
)

// loadFixture type-checks the fixture packages (with their tests) once.
func loadFixture(t *testing.T) *goIndex {
	t.Helper()
	fixtureOnce.Do(func() {
		fixtureIx, fixtureErr = loadGo("../../..", "./"+fixtureDir, "./src/cmd/apiparity/testdata/port")
	})
	if fixtureErr != nil {
		t.Fatal(fixtureErr)
	}
	return fixtureIx
}

func fixtureDart(t *testing.T) []DartLibrary {
	t.Helper()
	b, err := os.ReadFile("testdata/fixture_dart.json")
	if err != nil {
		t.Fatal(err)
	}
	libs, err := readDartDump(b)
	if err != nil {
		t.Fatal(err)
	}
	return libs
}

// measureFixture matches the fixture library with the given divergence
// table.
func measureFixture(t *testing.T, table string) (*Library, []error) {
	t.Helper()
	ix := loadFixture(t)
	div, err := parseDivergences([]byte(table), []byte(fixtureMapping))
	if err != nil {
		t.Fatal(err)
	}
	reexportedPackages["port"] = struct{ library, dir string }{"fixture", "src/cmd/apiparity/testdata/port"}
	defer delete(reexportedPackages, "port")
	libs, errs := measure(fixtureDart(t), ix, div, func(string) string { return fixtureDir })
	if len(libs) != 1 {
		t.Fatalf("libraries: %d", len(libs))
	}
	return libs[0], errs
}

func fixtureResults(t *testing.T) (*Library, map[string]*Decl, map[string]*Member) {
	t.Helper()
	lib, errs := measureFixture(t, fixtureTable)
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	decls := map[string]*Decl{}
	members := map[string]*Member{}
	for _, d := range lib.Decls {
		decls[d.Name] = d
		for _, m := range d.Members {
			members[m.Key] = m
		}
	}
	return lib, decls, members
}

func TestDeclarations(t *testing.T) {
	_, decls, _ := fixtureResults(t)
	for name, want := range map[string]bool{
		"Point":           true,  // class → type
		"Colors":          true,  // class of statics → package variable
		"debugLevel":      true,  // variable → var
		"runIt":           true,  // function → func
		"RunIt":           false, // a class can't be a func
		"missingFunction": false,
		"thing":           true, // divergence table
		"Mixed":           true,
		"channel":         false, // Channel is a type, not a function
		"Other":           true,  // a re-exported alias
	} {
		if got := decls[name].Exposed; got != want {
			t.Errorf("%s exposed = %v, want %v", name, got, want)
		}
	}
	if d := decls["thing"]; d.Divergence == nil || d.Go != "ThingInstance" {
		t.Errorf("thing: divergence %v, go %q", d.Divergence, d.Go)
	}
	if decls["Point"].Divergence != nil {
		t.Errorf("Point: unexpected divergence")
	}
}

func TestMembers(t *testing.T) {
	_, _, members := fixtureResults(t)
	cases := []struct {
		key     string
		exposed bool
		goName  string // one of the Go identifiers, when exposed
	}{
		// Constructors: NewFoo, NewFooNamed, struct literals, abstract bases.
		{"Point.new", true, "NewPoint"},
		{"Point.polar", true, "NewPointPolar"},
		{"Point.unit", false, ""},
		{"Widget.new", true, "Widget"},
		{"Viewer.new", true, "NewViewer"},
		{"Box.fromCorners", true, "NewBoxFromCorners"},
		{"Shape.new", true, "ShapeBase.Construct"},
		{"Listenable.new", true, "Listenable"},
		{"Listenable.empty", false, ""},
		{"Node.new", false, ""},
		{"Event.new", true, "EventBase.InitEvent"},
		{"Sealed.new", false, ""}, // unexported base state and parameters
		{"Calm.new", true, "CalmBase"},
		{"Channel.new", true, "NewChannel"},
		{"Manager.new", true, "Manager.Construct"},
		{"Pad.of", true, "NewPadOf"},
		{"Pad.all", true, "NewPadAll"},
		// Fields, getters, setters.
		{"Point.x", true, "(field) X"},
		{"Point.y", true, "(field) Y"},
		{"Point.label", false, ""}, // settable, but Go has no SetLabel
		{"Point.tag", true, "Point.SetTag"},
		{"Point.length", true, "Point.Length"},
		{"Point.length=", true, "Point.SetLength"},
		{"Point.width=", false, ""},
		{"Point.label=", false, ""}, // Label() is a getter
		{"Shape.name", true, "ShapeBase.Name"},
		{"Shape.paint", true, "Shape.Paint"},
		{"Viewer.viewId", true, "(field) ViewID"},
		{"Viewer.url", true, "Viewer.SetURL"},
		{"Viewer.actionIds", true, "(field) ActionIDs"},
		// Operators.
		{"Point.+", true, "Point.Add"},
		{"Point.unary-", true, "Point.Neg"},
		{"Point.~/", true, "Point.TruncDiv"},
		{"Point.<", true, "Point.Less"},
		{"Point.[]", false, ""},
		{"Point.==", true, "Point"},
		{"Point.hashCode", true, "Point"},
		{"Box.==", true, "Box.Equal"},
		{"Box.hashCode", true, "Box.Equal"},
		// Methods.
		{"Point.toString", true, "Point.String"},
		{"Fails.toString", true, "Fails.Error"},
		{"Point.scale", true, "Point.Scale"},
		{"Point.moveBy", true, "Point.MoveBy"},
		{"Point.cast", true, "PointCast"},
		{"Point.drive", true, "Drive"},
		{"Point.spin", false, ""}, // Spin is not generic
		{"Point.rotate", false, ""},
		{"Mixed.ping", true, "MixedMixin.Ping"},
		{"Alias.ping", true, "Alias.Ping"},
		// Statics.
		{"Point.origin", true, "PointOrigin"},
		{"Point.zero", true, "PointZero"},
		{"Registry.instance", true, "RegistryInstance"},
		{"Registry.count", true, "RegistryCount"},
		{"Registry.mode=", true, "SetRegistryMode"},
		{"Registry.client=", true, "RegistrySetClient"},
		{"Registry.total=", true, "RegistryTotal"},
		{"Registry.level", false, ""}, // settable, but only a getter function
		{"Registry.registryName", true, "RegistryName"},
		{"Colors.red", true, "(field) Red"},
		{"Colors.green", false, ""},
		{"Alias.onError", true, "AliasOnError"},
		// Enums.
		{"Hue.red", true, "HueRed"},
		{"Hue.darkBlue", true, "HueDarkBlue"},
		{"Hue.url_link", true, "HueURLLink"},
		{"Hue.green", false, ""},
		{"Hue.isDark", true, "Hue.IsDark"},
		// The divergence table.
		{"Listenable.merge", true, "ListenableMerge"},
		{"Point.*", true, "AliasOnError"},
		{"Box.[]", true, "Box.Get"},
		// A concrete class's struct with unexported state is no literal.
		{"Hidden.new", false, ""},
	}
	for _, c := range cases {
		m := members[c.key]
		if m == nil {
			t.Errorf("%s: not listed", c.key)
			continue
		}
		if m.Exposed != c.exposed {
			t.Errorf("%s exposed = %v (go %v), want %v", c.key, m.Exposed, m.Go, c.exposed)
			continue
		}
		if c.exposed && !contains(m.Go, c.goName) {
			t.Errorf("%s → %v, want %s among them", c.key, m.Go, c.goName)
		}
	}
	if members["Listenable.merge"].Divergence == nil || members["Point.+"].Divergence != nil {
		t.Errorf("divergences: merge %v, + %v", members["Listenable.merge"].Divergence, members["Point.+"].Divergence)
	}
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// paramState renders a callable's parameters as name→Go ("-" when missing).
func paramState(ps []*Param) string {
	var b []string
	for _, p := range ps {
		g := "-"
		if p.Exposed {
			g = p.Go
		}
		b = append(b, p.Name+"→"+g)
	}
	return strings.Join(b, " ")
}

func TestParameters(t *testing.T) {
	_, decls, members := fixtureResults(t)
	for key, want := range map[string]string{
		// Positional: by name, else in order.
		"Point.new":   "x→x y→y",
		"Point.polar": "radius→r angle→theta",
		"Viewer.new":  "viewId→viewID",
		// Optional positional: variadic.
		"Point.scale": "f→f fy→fy... fz→fy...",
		// Named: options struct fields; a missing field is a missing
		// parameter.
		"Point.moveBy": "dx→Dx dy→Dy dz→-",
		// A struct not named ...Options holding named parameters.
		"Box.fromCorners": "tl→tl topLeft→TopLeft topRight→TopRight bottomLeft→-",
		// A struct literal: the fields.
		"Widget.new": "key→Key child→Child flex→-",
		// Construct(self, name, opts...): self is skipped.
		"Shape.new": "name→title sides→Sides color→-",
		// Init<Class>(...).
		"Event.new": "id→id at→At",
		// An optional positional parameter in an ...Options struct.
		"Channel.new": "name→name codec→Codec",
		// Of String() and ToString(opts), the one taking the parameter.
		"Point.toString": "minLevel→MinLevel",
		// A struct parameter named like a positional parameter is not
		// options, nor is one holding no named parameter.
		"Pad.of":  "insets→insets left→Left",
		"Pad.all": "value→v",
		// == on a comparable value.
		"Point.==": "other→==",
		"Box.==":   "other→o",
		// Operators' operands.
		"Point.+": "other→o",
		// A missing member's parameters are missing.
		"Node.new": "depth→-",
		// A divergence's Go function is matched too.
		"Listenable.merge": "listenables→ls",
		"Box.[]":           "key→k",
	} {
		if got := paramState(members[key].Params); got != want {
			t.Errorf("%s: %s, want %s", key, got, want)
		}
	}
	// Required named parameters may be positional; the divergence table
	// covers d.
	if got, want := paramState(decls["runIt"].Params), "a→a b→b c→C d→"; got != want {
		t.Errorf("runIt: %s, want %s", got, want)
	}
	if p := decls["runIt"].Params[3]; p.Divergence == nil || p.Divergence.Kind != "impossible" {
		t.Errorf("runIt(d): divergence %v", p.Divergence)
	}
	// A required named parameter in the options, though a positional Go
	// parameter is left.
	if got, want := paramState(decls["place"].Params), "a→a b→B"; got != want {
		t.Errorf("place: %s, want %s", got, want)
	}
	if got := paramState(decls["missingFunction"].Params); got != "a→-" {
		t.Errorf("missingFunction: %s", got)
	}
}

func TestTested(t *testing.T) {
	_, decls, members := fixtureResults(t)
	for key, want := range map[string]bool{
		"Point.new":      true,  // NewPoint
		"Point.+":        true,  // p.Add
		"Point.polar":    false, // NewPointPolar is never called
		"Registry.count": true,
		"Widget.new":     true, // the literal
		"Hue.red":        true,
		"Hue.isDark":     true,
		"Hue.darkBlue":   false,
		"Alias.ping":     true, // by the port's own test
		"Untested.quiet": false,
	} {
		if got := members[key].Tested; got != want {
			t.Errorf("%s tested = %v, want %v", key, got, want)
		}
	}
	for name, want := range map[string]bool{
		"Point":    true,  // through its members
		"Untested": false, // exposed, never used
		"Alias":    true,  // the alias's target, used by the port's test
		"Colors":   false,
		"Other":    true, // only the port's test uses it
	} {
		if got := decls[name].Tested; got != want {
			t.Errorf("%s tested = %v, want %v", name, got, want)
		}
	}
}

func TestCounts(t *testing.T) {
	lib, _, _ := fixtureResults(t)
	// Counted by hand from fixture_dart.json and the expectations above.
	// Missing: 3 declarations (RunIt, channel, missingFunction), 14
	// members (see TestMembers), 9 parameters (Point.[]'s i, moveBy's dz,
	// fromCorners' bottomLeft, Widget's flex, Shape's color, Node's depth,
	// Sealed's v, Hidden's x, missingFunction's a). Tested: Point, Registry,
	// Widget, Hue, Alias, Other and 7 members (TestTested).
	want := Counts{
		Declarations: 29, Exposed: 26, Tested: 6,
		Members: 72, MembersExposed: 58, MembersTested: 7,
		Params: 49, ParamsExposed: 40,
		Divergent: 5, MembersDivergent: 3, ParamsDivergent: 1,
	}
	if got := lib.Counts; got != want {
		t.Errorf("counts:\n got %+v\nwant %+v", got, want)
	}
}

func TestDivergenceTableErrors(t *testing.T) {
	for _, c := range []struct{ table, want string }{
		{`[{"library": "fixture", "dart": "x", "kind": "renamed", "go": "X", "rule": "Types"}]`, "kind must be"},
		{`[{"library": "fixture", "dart": "x", "kind": "rename", "rule": "Types"}]`, "names its Go identifier"},
		{`[{"library": "fixture", "dart": "x", "kind": "impossible", "rule": "Nowhere"}]`, "does not name a section"},
		{`[{"library": "fixture", "dart": "x", "kind": "impossible", "rule": ""}]`, "does not name a section"},
		{`[{"library": "fixture", "dart": "x", "kind": "impossible", "rule": "Types"},
		  {"library": "fixture", "dart": "x", "kind": "impossible", "rule": "Types"}]`, "listed twice"},
		{`[{"library": "", "dart": "x", "kind": "impossible", "rule": "Types"}]`, "required"},
	} {
		_, err := parseDivergences([]byte(c.table), []byte(fixtureMapping))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want %q", c.table, err, c.want)
		}
	}
	if _, err := parseDivergences([]byte(fixtureTable), []byte(fixtureMapping)); err != nil {
		t.Error(err)
	}
}

func TestDivergenceTableChecks(t *testing.T) {
	for _, c := range []struct{ entry, want string }{
		// The rules already find it.
		{`{"library": "fixture", "dart": "Point.+", "kind": "rename", "go": "Point.Add", "rule": "Operators"}`, "not needed"},
		{`{"library": "fixture", "dart": "Point", "kind": "rename", "go": "Point", "rule": "Types"}`, "not needed"},
		{`{"library": "fixture", "dart": "Point.moveBy(dx)", "kind": "impossible", "rule": "Types"}`, "not needed"},
		// A rename whose Go identifier doesn't exist.
		{`{"library": "fixture", "dart": "Point.rotate", "kind": "rename", "go": "Point.Rotate", "rule": "Types"}`, "not found"},
		{`{"library": "fixture", "dart": "Point.width=", "kind": "rename", "go": "nowhere:SetWidth", "rule": "Types"}`, "not found"},
		{`{"library": "fixture", "dart": "Point.moveBy(dz)", "kind": "rename", "go": "PointMoveByOptions.Dz", "rule": "Types"}`, "not found"},
		// No such Dart name.
		{`{"library": "fixture", "dart": "Point.nothing", "kind": "impossible", "rule": "Types"}`, "no such Dart"},
		{`{"library": "other", "dart": "Point", "kind": "impossible", "rule": "Types"}`, "no such Dart"},
	} {
		_, errs := measureFixture(t, "["+c.entry+"]")
		found := false
		for _, e := range errs {
			found = found || strings.Contains(e.Error(), c.want)
		}
		if !found {
			t.Errorf("%s: errors %v, want %q", c.entry, errs, c.want)
		}
	}
}

func TestSpellings(t *testing.T) {
	for in, want := range map[string][]string{
		"ViewId":           {"ViewId", "ViewID"},
		"ActionIds":        {"ActionIds", "ActionIDs"},
		"ToJson":           {"ToJson", "ToJSON"},
		"Identity":         {"Identity"},
		"Url":              {"Url", "URL"},
		"Platform_views_2": {"Platform_views_2", "PlatformViews2"},
		"Url_link":         {"Url_link", "UrlLink", "URLLink"},
	} {
		if got := spellings(in); !reflect.DeepEqual(got, want) {
			t.Errorf("spellings(%s) = %v, want %v", in, got, want)
		}
	}
	if got := goName("$foo"); got != "Foo" {
		t.Errorf("goName($foo) = %s", got)
	}
}

func TestMatchSignature(t *testing.T) {
	slots := func(names ...string) []*goSlot {
		var out []*goSlot
		for _, n := range names {
			out = append(out, &goSlot{name: strings.TrimSuffix(n, "..."), variadic: strings.HasSuffix(n, "...")})
		}
		return out
	}
	req := func(n string) DartParam { return DartParam{Name: n, Required: true} }
	opt := func(n string) DartParam { return DartParam{Name: n} }
	named := func(n string) DartParam { return DartParam{Name: n, Named: true} }
	reqNamed := func(n string) DartParam { return DartParam{Name: n, Named: true, Required: true} }
	for _, c := range []struct {
		name  string
		dps   []DartParam
		slots []*goSlot
		opts  map[string]string
		want  string
	}{
		{"by name beats order", []DartParam{req("a"), req("b")}, slots("b", "a"), nil, "a→a b→b"},
		{"in order", []DartParam{req("a"), req("b")}, slots("x", "y"), nil, "a→x b→y"},
		{"too few Go parameters", []DartParam{req("a"), req("b")}, slots("x"), nil, "a→x b→-"},
		{"optional named is not positional", []DartParam{req("a"), named("b")}, slots("x", "y"), nil, "a→x b→-"},
		{"optional named by name", []DartParam{req("a"), named("b")}, slots("x", "b"), nil, "a→x b→b"},
		{"required named in order", []DartParam{req("a"), reqNamed("b")}, slots("x", "y"), nil, "a→x b→y"},
		{"named to options", []DartParam{named("b"), named("c")}, nil, map[string]string{"b": "B"}, "b→B c→-"},
		{"optional positional to variadic", []DartParam{req("a"), opt("b")}, slots("a", "rest..."), nil, "a→a b→rest..."},
		{"optional positional not to a fixed parameter by order", []DartParam{req("a"), opt("b")}, slots("a", "x"), nil, "a→a b→x"},
		{"optional positional to options", []DartParam{req("a"), opt("b")}, slots("a"), map[string]string{"b": "B"}, "a→a b→B"},
		{"required positional in options", []DartParam{req("a")}, nil, map[string]string{"a": "A"}, "a→A"},
		{"variadic is not a required slot", []DartParam{req("a"), req("b")}, slots("a", "c..."), nil, "a→a b→-"},
		{"case and underscores", []DartParam{named("texture_id")}, nil, map[string]string{"textureid": "TextureID"}, "texture_id→TextureID"},
	} {
		if got := paramState(matchSignature(c.dps, c.slots, c.opts)); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestReports(t *testing.T) {
	lib, _, _ := fixtureResults(t)
	div, _ := parseDivergences([]byte(fixtureTable), []byte(fixtureMapping))
	summary := markdown([]*Library{lib})
	for _, want := range []string{
		"| `fixture` | `" + fixtureDir + "` | 29 | 26 (90%) | 6 | 72 | 58 (81%) | 7 | 49 | 40 (82%) | 5 | [fixture.md](parity/fixture.md) |",
		"**Missing (3):** `RunIt`, `channel`, `missingFunction`",
		"`Untested` (untested)",
		"**Total: 26 / 29 declarations exposed (89.7%), 6 tested; 58 / 72 members exposed (80.6%), 7 tested; 40 / 49 parameters exposed (81.6%); 5 divergent.**",
	} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary lacks %q:\n%s", want, summary)
		}
	}
	d := detail(lib, div)
	for _, want := range []string{
		"## Missing declarations (3)",
		"- `missingFunction` (function, `lib/run.dart`)",
		"- `Point`: `unit` (constructor), `label` (field), `width=` (setter), `label=` (setter), `[]` (operator), `spin` (method), `rotate` (method)",
		"- `Point.moveBy`: `{dz}`",
		"- `Box.fromCorners`: `{bottomLeft}`",
		"- `Widget.new`: `{flex}`",
		"- `Listenable.merge` — rename → `ListenableMerge` (API_MAPPING: Libraries: wrappers)",
		"- `runIt(d)` — impossible (API_MAPPING: Types): no Go form",
		"- `Untested`: (the declaration), `quiet`",
	} {
		if !strings.Contains(d, want) {
			t.Errorf("worklist lacks %q:\n%s", want, d)
		}
	}
	if got := detailFile("dart:ui"); got != "dart_ui.md" {
		t.Errorf("detailFile: %s", got)
	}
}

func TestMappingSectionsExist(t *testing.T) {
	// The real table must load against the real API_MAPPING.md.
	if _, err := loadDivergences(filepath.Join("../../docs/api_divergences.json"), "../../docs/API_MAPPING.md"); err != nil {
		t.Fatal(err)
	}
}

func TestUntypedTestsFallBackToNames(t *testing.T) {
	// A test package that doesn't type-check: the names its test files
	// mention count (the old file-level heuristic, kept as the fallback).
	ix, err := loadGo("../../..", "./src/cmd/apiparity/testdata/broken")
	if err != nil {
		t.Fatal(err)
	}
	if !ix.mentioned["Quiet"] || !ix.mentioned["Undefined"] {
		t.Errorf("mentioned: %v", ix.mentioned)
	}
	quiet := ix.pkgs["src/cmd/apiparity/testdata/broken"].Scope().Lookup("Quiet")
	if !ix.tested([]types.Object{quiet}) {
		t.Error("Quiet: not tested")
	}
	// The type-checked fixture uses no names this way.
	if fx := loadFixture(t); len(fx.mentioned) != 0 {
		t.Errorf("fixture mentioned: %v", fx.mentioned)
	}
}
