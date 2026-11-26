package main

import (
	"go/types"
	"strings"
	"unicode"
)

// The matcher maps each Dart declaration, member and parameter to Go by the
// rules of src/docs/API_MAPPING.md, citing the section each rule comes from.
// A Dart name the rules don't reach counts as exposed only through an entry
// of the divergence table (src/docs/api_divergences.json).

// Decl is a top-level declaration's result.
type Decl struct {
	Name       string      `json:"name"`
	Kind       string      `json:"kind"`
	File       string      `json:"file"`
	Package    string      `json:"package,omitempty"`
	Deprecated bool        `json:"deprecated,omitempty"`
	Go         string      `json:"go,omitempty"`
	Exposed    bool        `json:"exposed"`
	Tested     bool        `json:"tested"`
	Divergence *Divergence `json:"divergence,omitempty"`
	Members    []*Member   `json:"members,omitempty"`
	Params     []*Param    `json:"params,omitempty"`
}

// Member is a member's result. Key is how the member is named in reports and
// in the divergence table: Class.member, Class.new (the unnamed constructor),
// Class.member= (a setter), Class.== (an operator).
type Member struct {
	Key        string      `json:"key"`
	Kind       string      `json:"kind"`
	Static     bool        `json:"static,omitempty"`
	Deprecated bool        `json:"deprecated,omitempty"`
	Go         []string    `json:"go,omitempty"` // the Go identifiers it maps to
	Exposed    bool        `json:"exposed"`
	Tested     bool        `json:"tested"`
	Divergence *Divergence `json:"divergence,omitempty"`
	Params     []*Param    `json:"params,omitempty"`
}

// Param is a parameter's result.
type Param struct {
	Name       string      `json:"name"`
	Named      bool        `json:"named,omitempty"`
	Required   bool        `json:"required,omitempty"`
	Go         string      `json:"go,omitempty"` // the Go parameter or option field
	Exposed    bool        `json:"exposed"`
	Divergence *Divergence `json:"divergence,omitempty"`
}

// goName maps a Dart identifier to the exported Go identifier (API_MAPPING
// "Declarations": fooBar → FooBar).
func goName(dart string) string {
	r := []rune(strings.TrimPrefix(dart, "$"))
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// operatorMethods are API_MAPPING's "Operators" table. `==` is handled
// separately (an Equal method or a comparable value type).
var operatorMethods = map[string]string{
	"+": "Add", "-": "Sub", "*": "Mul", "/": "Div", "unary-": "Neg",
	"~/": "TruncDiv", "%": "Mod", "&": "And",
	"<": "Less", "<=": "LessEq", ">": "Greater", ">=": "GreaterEq",
}

// matcher matches one Dart library against its Go package.
type matcher struct {
	pkg  *types.Package
	ix   *goIndex // for divergences naming other packages and for tests; may be nil
	div  *divergenceTable
	lib  string
	used map[*Divergence]bool // divergence entries that applied
	errs []error              // stale or unneeded divergence entries
}

// lookup finds a package-level Go declaration, spelled with Go's
// initialisms when the Dart spelling isn't declared.
func (m *matcher) lookup(name string) types.Object {
	if name == "" {
		return nil
	}
	for _, n := range spellings(name) {
		if o := m.pkg.Scope().Lookup(n); o != nil {
			return o
		}
	}
	return nil
}

// spellings are a Go name as goName gives it, then with Go's initialisms,
// then (for a snake_case Dart name, platform_views_2) in MixedCaps:
// Go names have no underscores (API_MAPPING "Declarations").
func spellings(name string) []string {
	out := []string{name}
	add := func(n string) {
		for _, o := range out {
			if o == n {
				return
			}
		}
		out = append(out, n)
	}
	add(initialisms(name))
	mixed := mixedCaps(name)
	add(mixed)
	add(initialisms(mixed))
	return out
}

// mixedCaps drops underscores, capitalizing what follows them.
func mixedCaps(name string) string {
	if !strings.Contains(name, "_") {
		return name
	}
	var b strings.Builder
	up := false
	for _, r := range name {
		if r == '_' {
			up = true
			continue
		}
		if up {
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
		up = false
	}
	return b.String()
}

// goInitialisms are the words Go spells in capitals (API_MAPPING
// "Declarations": viewId → ViewID, url → URL, toJson → ToJSON).
var goInitialisms = []string{"ID", "URL", "URI", "JSON", "HTTP", "HTML", "API", "UTF", "GPU"}

// initialisms respells the initialisms of a Go identifier (ViewId →
// ViewID, ActionIds → ActionIDs); words are split at capitals and digits.
func initialisms(name string) string {
	var b strings.Builder
	rs := []rune(name)
	for i := 0; i < len(rs); {
		j := i + 1
		for j < len(rs) && !unicode.IsUpper(rs[j]) && !unicode.IsDigit(rs[j]) {
			j++
		}
		w := string(rs[i:j])
		for _, in := range goInitialisms {
			if w == in[:1]+strings.ToLower(in[1:]) {
				w = in
			} else if w == in[:1]+strings.ToLower(in[1:])+"s" {
				w = in + "s"
			}
		}
		b.WriteString(w)
		i = j
	}
	return b.String()
}

// typeCands returns the Go types a Dart class, mixin or enum maps to:
// Foo, and for abstract classes and mixins the embeddable FooBase/FooMixin
// (API_MAPPING "Declarations").
func (m *matcher) typeCands(class string) []*types.TypeName {
	var out []*types.TypeName
	for _, n := range []string{goName(class), goName(class) + "Base", goName(class) + "Mixin"} {
		if tn, ok := m.lookup(n).(*types.TypeName); ok {
			out = append(out, tn)
		}
	}
	return out
}

// fieldOrMethod looks a field or method up on the candidate types (promoted
// ones included, pointer receivers included).
func (m *matcher) fieldOrMethod(cands []*types.TypeName, name string) []types.Object {
	var out []types.Object
	for _, tn := range cands {
		var obj types.Object
		for _, n := range spellings(name) {
			if obj, _, _ = types.LookupFieldOrMethod(tn.Type(), true, m.pkg, n); obj != nil {
				break
			}
		}
		if obj != nil {
			out = append(out, obj)
		}
	}
	return out
}

func isVar(o types.Object) bool { _, ok := o.(*types.Var); return ok }

func isField(o types.Object) bool { v, ok := o.(*types.Var); return ok && v.IsField() }

func isMethodOrFunc(o types.Object) bool { _, ok := o.(*types.Func); return ok }

func anyOf(objs []types.Object, f func(types.Object) bool) []types.Object {
	var out []types.Object
	for _, o := range objs {
		if f(o) {
			out = append(out, o)
		}
	}
	return out
}

// matchDecl matches a top-level declaration and its members.
func (m *matcher) matchDecl(d DartDecl) *Decl {
	r := &Decl{Name: d.Name, Kind: d.Kind, File: d.File, Package: d.Package, Deprecated: d.Deprecated}
	var objs []types.Object
	if obj := m.lookup(goName(d.Name)); obj != nil && kindFits(d.Kind, obj) {
		objs = []types.Object{obj}
		r.Exposed = true
		// A re-exported package's type is an alias of its port's
		// (API_MAPPING "Libraries"): tests of the port count too.
		if tn, ok := obj.(*types.TypeName); ok && tn.IsAlias() {
			if n, ok := types.Unalias(tn.Type()).(*types.Named); ok {
				objs = append(objs, n.Obj())
			}
		}
	}
	if dv := m.div.find(m.lib, d.Name); dv != nil {
		if r.Exposed {
			m.unneeded(dv)
		} else {
			objs = m.applyDivergence(dv)
			r.Divergence = dv
			r.Exposed = true
		}
	}
	if len(objs) > 0 {
		r.Go = objs[0].Name()
	}
	r.Tested = r.Exposed && m.ix != nil && m.ix.tested(objs)
	if d.Kind == "function" {
		r.Params = m.matchParams(d.Name, d.Params, objs, nil)
	}
	for _, mem := range d.Members {
		mr := m.matchMember(d, mem)
		r.Members = append(r.Members, mr)
		// A class is tested through its constructors and members too.
		r.Tested = r.Tested || r.Exposed && mr.Tested
	}
	return r
}

// kindFits reports whether a Go object can stand for a Dart top-level
// declaration of the kind: a type for a class, mixin, enum or typedef; a
// function or variable for a function, getter or variable.
func kindFits(kind string, obj types.Object) bool {
	_, isType := obj.(*types.TypeName)
	switch kind {
	case "class":
		// A class of static members may be a package variable holding a
		// struct (API_MAPPING "Members"; "Zones and statics").
		return isType || isVar(obj)
	case "mixin", "enum", "typedef", "extension type", "extension":
		return isType
	}
	return !isType
}

// memberKey is the report and divergence-table name of a member.
func memberKey(class string, mem DartMember) string {
	switch {
	case mem.Kind == "constructor" && mem.Name == "":
		return class + ".new"
	case mem.Kind == "setter":
		return class + "." + mem.Name + "="
	}
	return class + "." + mem.Name
}

func (m *matcher) matchMember(d DartDecl, mem DartMember) *Member {
	r := &Member{Key: memberKey(d.Name, mem), Kind: mem.Kind, Static: mem.Static, Deprecated: mem.Deprecated}
	var objs []types.Object
	var literal *types.TypeName // a constructor that is a struct literal (or an embedded base's fields)
	objs, literal = m.memberObjects(d, mem)
	r.Exposed = len(objs) > 0
	if dv := m.div.find(m.lib, r.Key); dv != nil {
		if r.Exposed {
			m.unneeded(dv)
		} else {
			objs, literal = m.applyDivergence(dv), nil
			r.Divergence = dv
			r.Exposed = true
		}
	}
	for _, o := range objs {
		r.Go = append(r.Go, qualifiedName(o))
	}
	r.Tested = r.Exposed && m.ix != nil && m.ix.tested(objs)
	if mem.Kind == "constructor" || mem.Kind == "method" || mem.Kind == "operator" {
		r.Params = m.matchParams(r.Key, mem.Params, objs, literal)
	}
	return r
}

// qualifiedName prints a Go object as Type.Member or Name.
func qualifiedName(o types.Object) string {
	if f, ok := o.(*types.Func); ok {
		if recv := f.Type().(*types.Signature).Recv(); recv != nil {
			return typeName(recv.Type()) + "." + f.Name()
		}
	}
	if v, ok := o.(*types.Var); ok && v.IsField() {
		return "(field) " + v.Name()
	}
	return o.Name()
}

func typeName(t types.Type) string {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	switch t := t.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Alias:
		return t.Obj().Name()
	}
	return t.String()
}

// memberObjects finds the Go declarations a member maps to by the rules.
func (m *matcher) memberObjects(d DartDecl, mem DartMember) ([]types.Object, *types.TypeName) {
	cands := m.typeCands(d.Name)
	class := goName(d.Name)
	g := goName(mem.Name)
	switch mem.Kind {
	case "constructor":
		return m.constructor(d, mem, cands)
	case "enumValue":
		// API_MAPPING "Declarations": enum Foo { a } → FooA.
		if o := m.lookup(class + g); o != nil && !isTypeName(o) {
			return []types.Object{o}, nil
		}
		return nil, nil
	case "operator":
		return m.operator(d, mem, cands), nil
	}
	if mem.Static {
		return m.static(d, mem), nil
	}
	switch mem.Kind {
	case "method":
		// API_MAPPING "Declarations": toString() → String().
		// An exception is a Go error, printed by Error() (API_MAPPING
		// "Errors and stack traces").
		if mem.Name == "toString" {
			objs := append(m.fieldOrMethod(cands, "String"), m.fieldOrMethod(cands, "ToString")...)
			return anyOf(append(objs, m.fieldOrMethod(cands, "Error")...), isMethodOrFunc), nil
		}
		objs := anyOf(m.fieldOrMethod(cands, g), isMethodOrFunc)
		if len(objs) == 0 && mem.Generic {
			// A generic method is a generic function Type+Method: Go
			// methods can't take type parameters (API_MAPPING "Futures":
			// `then<R>` → async.Then; "Known divergences": drive<U>,
			// scheduleTask<T>, invokeCallback<T>...).
			for _, n := range []string{class + g, g} {
				if o, ok := m.lookup(n).(*types.Func); ok && o.Type().(*types.Signature).TypeParams().Len() > 0 {
					objs = []types.Object{o}
					break
				}
			}
		}
		return objs, nil
	case "getter":
		// API_MAPPING "Members": getter foo → Foo(); a public final field
		// → an exported struct field. `hashCode` goes with `==`
		// ("Declarations": `==`/hashCode → Equal / comparable values).
		objs := m.fieldOrMethod(cands, g)
		if len(objs) == 0 && mem.Name == "hashCode" {
			objs = m.equality(class, cands)
		}
		return objs, nil
	case "setter":
		// API_MAPPING "Members": setter foo= → SetFoo(v), or an exported
		// (assignable) struct field.
		objs := anyOf(m.fieldOrMethod(cands, "Set"+g), isMethodOrFunc)
		objs = append(objs, anyOf(m.fieldOrMethod(cands, g), isField)...)
		return objs, nil
	case "field":
		// A field is a getter (and a setter unless final): an exported
		// struct field, or the methods Foo() (and SetFoo).
		fields := anyOf(m.fieldOrMethod(cands, g), isField)
		if len(fields) > 0 {
			return fields, nil
		}
		get := anyOf(m.fieldOrMethod(cands, g), isMethodOrFunc)
		if len(get) == 0 {
			return nil, nil
		}
		if mem.Settable {
			set := anyOf(m.fieldOrMethod(cands, "Set"+g), isMethodOrFunc)
			if len(set) == 0 {
				return nil, nil
			}
			get = append(get, set...)
		}
		return get, nil
	}
	return nil, nil
}

func isTypeName(o types.Object) bool { _, ok := o.(*types.TypeName); return ok }

// static maps a static member (API_MAPPING "Members": Foo.bar → FooBar, or
// Bar when bar already contains Foo; a class of statics → a package variable
// holding a struct, Colors.red → Colors.Red; "Zones and statics": a package
// variable whose methods are the statics). A settable static must be
// assignable: a variable or field, or a SetFooBar function.
func (m *matcher) static(d DartDecl, mem DartMember) []types.Object {
	class, g := goName(d.Name), goName(mem.Name)
	pkgs := []*types.Package{m.pkg}
	if rp, ok := reexportedPackages[d.Package]; ok && m.ix != nil && m.ix.pkgs[rp.dir] != nil {
		// A re-exported package's statics may stay in its port
		// (API_MAPPING "Libraries": Listenable.onError is
		// listen.ListenableOnError).
		pkgs = append(pkgs, m.ix.pkgs[rp.dir])
	}
	names := []string{class + g}
	if strings.Contains(strings.ToLower(mem.Name), strings.ToLower(d.Name)) {
		names = append(names, g)
	}
	var found, setters []types.Object
	for _, pkg := range pkgs {
		sub := &matcher{pkg: pkg}
		for _, n := range names {
			if o := sub.lookup(n); o != nil && !isTypeName(o) {
				found = append(found, o)
			}
			if o, ok := sub.lookup("Set" + n).(*types.Func); ok {
				setters = append(setters, o)
			}
		}
		// FooSetBar (API_MAPPING "Members": static setters).
		if o, ok := sub.lookup(class + "Set" + g).(*types.Func); ok {
			setters = append(setters, o)
		}
		if v, ok := sub.lookup(class).(*types.Var); ok {
			cand := []*types.TypeName{types.NewTypeName(0, pkg, class, v.Type())}
			found = append(found, sub.fieldOrMethod(cand, g)...)
			setters = append(setters, anyOf(sub.fieldOrMethod(cand, "Set"+g), isMethodOrFunc)...)
		}
	}
	assignable := anyOf(found, isVar)
	switch {
	case mem.Kind == "setter":
		// The setter alone: a variable or field, or SetFooBar.
		return append(assignable, setters...)
	case mem.Kind == "field" && mem.Settable && len(assignable) == 0:
		// A getter function needs its setter.
		if len(found) == 0 || len(setters) == 0 {
			return nil
		}
		return append(found, setters...)
	}
	return found
}

// equality maps `==` (and hashCode): an Equal method, or a comparable value
// type (API_MAPPING "Declarations" and "Operators": `==` on value types is
// Go's ==; a struct or basic type, not an interface or pointer).
func (m *matcher) equality(class string, cands []*types.TypeName) []types.Object {
	if objs := anyOf(m.fieldOrMethod(cands, "Equal"), isMethodOrFunc); len(objs) > 0 {
		return objs
	}
	for _, tn := range cands {
		if tn.Name() != class {
			continue // the class's own type, not an embeddable base
		}
		switch tn.Type().Underlying().(type) {
		case *types.Struct, *types.Basic:
			if types.Comparable(tn.Type()) {
				return []types.Object{tn}
			}
		}
	}
	return nil
}

func (m *matcher) operator(d DartDecl, mem DartMember, cands []*types.TypeName) []types.Object {
	if mem.Name == "==" {
		return m.equality(goName(d.Name), cands)
	}
	name, ok := operatorMethods[mem.Name]
	if !ok {
		return nil // no rule: only the divergence table can map it
	}
	return anyOf(m.fieldOrMethod(cands, name), isMethodOrFunc)
}

// constructor maps a constructor (API_MAPPING "Constructors"): Foo(...) →
// NewFoo, Foo.named(...) → NewFooNamed; the unnamed constructor of a struct
// whose fields are all exported may be a struct literal. A constructor of an
// abstract class, or of a class Go types extend, is Dart's super(...): its Go
// form is embedding the class's struct or FooBase/FooMixin, set up by
// Construct or InitFoo (or NewFooBase); an abstract class without state is
// implemented, not constructed ("Constructors"; "Declarations"; "Mixins and
// overridable methods").
func (m *matcher) constructor(d DartDecl, mem DartMember, cands []*types.TypeName) ([]types.Object, *types.TypeName) {
	class := goName(d.Name)
	name := "New" + class + goName(mem.Name)
	if o, ok := m.lookup(name).(*types.Func); ok {
		return []types.Object{o}, nil
	}
	if d.Abstract {
		if mem.Name == "" {
			if o, ok := m.lookup("New" + class + "Base").(*types.Func); ok {
				return []types.Object{o}, nil
			}
		}
		// The base's set-up method: Construct or Init<Class>
		// ("Constructors").
		for _, n := range []string{"Construct" + goName(mem.Name), "Init" + class + goName(mem.Name)} {
			if objs := anyOf(m.fieldOrMethod(cands, n), isMethodOrFunc); len(objs) > 0 {
				return objs, nil
			}
			if o, ok := m.lookup(n).(*types.Func); ok {
				return []types.Object{o}, nil
			}
		}
		// Else the base's fields, when they can all be set (or there is
		// nothing to set).
		for _, tn := range cands {
			if st, ok := tn.Type().Underlying().(*types.Struct); ok && (allExported(st) || len(mem.Params) == 0) {
				return []types.Object{tn}, tn
			}
		}
		// An abstract class without state is an interface alone
		// ("Declarations": FooBase only when it carries state or default
		// methods): implementing it is its parameterless constructor.
		if mem.Name == "" && !mem.Factory && len(mem.Params) == 0 && len(cands) > 0 && types.IsInterface(cands[0].Type()) && cands[0].Name() == class {
			return []types.Object{cands[0]}, nil
		}
		return nil, nil
	}
	if mem.Name != "" {
		return nil, nil
	}
	if objs := anyOf(m.fieldOrMethod(cands, "Construct"), isMethodOrFunc); len(objs) > 0 {
		return objs, nil
	}
	if tn := m.lookupType(class); tn != nil {
		if st, ok := tn.Type().Underlying().(*types.Struct); ok && allExported(st) {
			return []types.Object{tn}, tn
		}
	}
	return nil, nil
}

func (m *matcher) lookupType(name string) *types.TypeName {
	tn, _ := m.lookup(name).(*types.TypeName)
	return tn
}

func allExported(st *types.Struct) bool {
	for i := 0; i < st.NumFields(); i++ {
		if !st.Field(i).Exported() {
			return false
		}
	}
	return true
}

// ---- parameters ----

// goSlot is a Go parameter that can stand for a Dart parameter.
type goSlot struct {
	name     string
	variadic bool
	taken    bool
}

// norm compares parameter names: case and underscores don't matter
// (textureId ~ TextureID).
func norm(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

// optionFields adds the exported fields of an options struct (with the
// fields of embedded structs) to opts.
func optionFields(t types.Type, opts map[string]string) {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f.Embedded() {
			optionFields(f.Type(), opts)
		}
		if f.Exported() {
			if _, dup := opts[norm(f.Name())]; !dup {
				opts[norm(f.Name())] = f.Name()
			}
		}
	}
}

// isOptions reports whether a parameter type is an options struct
// (API_MAPPING "Constructors" and "Nullable values and options": named
// parameters are the fields of a struct named ...Options).
func isOptions(t types.Type) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	var name string
	switch t := t.(type) {
	case *types.Named:
		name = t.Obj().Name()
	case *types.Alias:
		name = t.Obj().Name()
	default:
		return false
	}
	_, isStruct := t.Underlying().(*types.Struct)
	return isStruct && strings.HasSuffix(name, "Options")
}

// holdsNamed reports whether a struct parameter holds named parameters: an
// options struct not named ...Options (`RRect.fromRectAndCorners(rect,
// {topLeft, ...})` → `NewRRectFromRectAndCorners(rect, RRectCorners)`).
// A Dart positional parameter of the Go parameter's name is not one.
func holdsNamed(t types.Type, goParam string, dps []DartParam) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if _, ok := t.Underlying().(*types.Struct); !ok {
		return false
	}
	fields := map[string]string{}
	optionFields(t, fields)
	named := false
	for _, dp := range dps {
		if !dp.Named && norm(dp.Name) == norm(goParam) {
			return false
		}
		if _, ok := fields[norm(dp.Name)]; ok && dp.Named {
			named = true
		}
	}
	return named
}

// signatureOf returns the signature a Go object can be called with: a
// function's or method's, or a function-typed variable's.
func signatureOf(o types.Object) *types.Signature {
	switch o := o.(type) {
	case *types.Func:
		return o.Type().(*types.Signature)
	case *types.Var:
		s, _ := o.Type().Underlying().(*types.Signature)
		return s
	}
	return nil
}

// matchParams matches a Dart callable's parameters against the Go
// declarations it maps to (the callable one exposing the most of them), or
// against the fields of literal (a struct-literal constructor), then applies
// the divergence table where the rules find nothing.
func (m *matcher) matchParams(key string, dps []DartParam, objs []types.Object, literal *types.TypeName) []*Param {
	var out []*Param
	switch {
	case literal != nil:
		opts := map[string]string{}
		optionFields(literal.Type(), opts)
		out = matchSignature(dps, nil, opts)
	case len(objs) > 0 && strings.HasSuffix(key, ".==") && isTypeName(objs[0]):
		// `==` is Go's == on a comparable value: the other operand is
		// there.
		out = newParams(dps)
		for _, p := range out {
			p.Exposed, p.Go = true, "=="
		}
	default:
		best := -1
		for _, o := range objs {
			sig := signatureOf(o)
			if sig == nil {
				continue
			}
			slots, opts := goSlots(o, sig, dps)
			ps := matchSignature(dps, slots, opts)
			if n := exposedCount(ps); n > best {
				out, best = ps, n
			}
		}
		if out == nil {
			out = newParams(dps)
		}
	}
	for _, p := range out {
		if dv := m.div.find(m.lib, key+"("+p.Name+")"); dv != nil {
			if p.Exposed {
				m.unneeded(dv)
			} else {
				p.Divergence, p.Exposed, p.Go = dv, true, dv.Go
				m.applyDivergence(dv) // checks the Go name
			}
		}
	}
	return out
}

func newParams(dps []DartParam) []*Param {
	var out []*Param
	for _, dp := range dps {
		out = append(out, &Param{Name: dp.Name, Named: dp.Named, Required: dp.Required})
	}
	return out
}

func exposedCount(ps []*Param) int {
	n := 0
	for _, p := range ps {
		if p.Exposed {
			n++
		}
	}
	return n
}

// goSlots splits a Go signature into the parameters that can stand for
// Dart parameters and the fields of its options structs.
func goSlots(o types.Object, sig *types.Signature, dps []DartParam) ([]*goSlot, map[string]string) {
	var slots []*goSlot
	opts := map[string]string{}
	ps := sig.Params()
	for i := 0; i < ps.Len(); i++ {
		v := ps.At(i)
		t := v.Type()
		variadic := sig.Variadic() && i == ps.Len()-1
		if variadic {
			t = t.(*types.Slice).Elem()
		}
		if i == 0 && v.Name() == "self" && strings.HasPrefix(o.Name(), "Construct") {
			continue // the outer object (Mixins and overridable methods)
		}
		if isOptions(t) || holdsNamed(t, v.Name(), dps) {
			optionFields(t, opts)
			continue
		}
		slots = append(slots, &goSlot{name: v.Name(), variadic: variadic})
	}
	return slots, opts
}

// matchSignature matches Dart parameters to Go parameters (slots) and option
// fields (opts: normalized name → field). API_MAPPING's rules:
//   - positional parameters are positional, matched by name, else in order;
//   - named parameters are the fields of an options struct ("Constructors",
//     "Nullable values and options"), and required named parameters may be
//     positional ("Zones and statics");
//   - optional positional parameters may be variadic or options ("Known
//     divergences": `Canvas.scale(sx, [sy])` → `Scale(sx, sy...)`;
//     `MethodChannel(name, [codec, binaryMessenger])` → MethodChannelOptions).
func matchSignature(dps []DartParam, slots []*goSlot, opts map[string]string) []*Param {
	out := newParams(dps)
	byName := func(name string) *goSlot {
		for _, s := range slots {
			if !s.taken && norm(s.name) == norm(name) {
				return s
			}
		}
		return nil
	}
	take := func(p *Param, s *goSlot) {
		s.taken = true // a variadic parameter still takes optional ones (step 3)
		p.Exposed, p.Go = true, s.name
		if s.variadic {
			p.Go += "..."
		}
	}
	// 1. By name: named parameters to option fields, else (as positional
	// parameters) to Go parameters of the same name.
	for _, p := range out {
		if p.Named {
			if f, ok := opts[norm(p.Name)]; ok {
				p.Exposed, p.Go = true, f
				continue
			}
		}
		if s := byName(p.Name); s != nil {
			take(p, s)
		}
	}
	// 2. In order: the remaining positional (and required named) parameters
	// to the remaining non-variadic Go parameters.
	next := func() *goSlot {
		for _, s := range slots {
			if !s.taken && !s.variadic {
				return s
			}
		}
		return nil
	}
	for _, p := range out {
		if p.Exposed || p.Named && !p.Required {
			continue
		}
		if s := next(); s != nil {
			take(p, s)
		}
	}
	// 3. Optional positional parameters: an option field of the same name,
	// else a variadic parameter.
	for _, p := range out {
		if p.Exposed || p.Named || p.Required {
			continue
		}
		if f, ok := opts[norm(p.Name)]; ok {
			p.Exposed, p.Go = true, f
			continue
		}
		for _, s := range slots {
			if s.variadic {
				take(p, s)
				break
			}
		}
	}
	// 4. Required positional parameters an options struct holds.
	for _, p := range out {
		if f, ok := opts[norm(p.Name)]; ok && !p.Exposed {
			p.Exposed, p.Go = true, f
		}
	}
	return out
}
