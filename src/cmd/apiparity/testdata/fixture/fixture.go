// Package fixture is a small Go "port" the apiparity tests match a small
// Dart library against (fixture_dart.json). Each declaration exercises one
// mapping rule of API_MAPPING.md; the comments give the Dart it stands for.
package fixture

import "github.com/jonathanperis/flutter-go/src/cmd/apiparity/testdata/port"

// Point: class Point { Point(this.x, this.y); Point.polar(r, theta); ... }
type Point struct{ X, Y float64 }

func NewPoint(x, y float64) Point                  { return Point{x, y} }
func NewPointPolar(r, theta float64) Point         { return Point{r, theta} }
func (p Point) Length() float64                    { return p.X }
func (p *Point) SetLength(v float64)               { p.X = v }
func (p Point) Add(o Point) Point                  { return Point{p.X + o.X, p.Y + o.Y} }
func (p Point) Neg() Point                         { return Point{-p.X, -p.Y} }
func (p Point) TruncDiv(o Point) Point             { return p }
func (p Point) Less(o Point) bool                  { return p.X < o.X }
func (p Point) String() string                     { return "Point" }
func (p *Point) Scale(f float64, fy ...float64)    {}
func (p *Point) MoveBy(opts ...PointMoveByOptions) {}
func (p *Point) Label() string                     { return "" } // getter only: the Dart field is settable
func (p *Point) Tag() string                       { return "" }
func (p *Point) SetTag(string)                     {}

// PointMoveByOptions are moveBy's named parameters.
type PointMoveByOptions struct{ Dx, Dy float64 }

// PointCast is the generic method cast<T>().
func PointCast[T any](p Point) T { var t T; return t }

// PointOrigin is the static field Point.origin; PointZero the static const.
var PointOrigin Point

const PointZero = 0

// Shape: abstract class Shape { Shape(this.name, {this.sides}); String get name; void paint(canvas); }
type Shape interface {
	Name() string
	Paint(canvas any)
}

// ShapeBase is Shape's embeddable state.
type ShapeBase struct{ name string }

func (b *ShapeBase) Construct(self Shape, title string, opts ...ShapeOptions) {}
func (b *ShapeBase) Name() string                                             { return b.name }

// ShapeOptions are Shape's named parameters.
type ShapeOptions struct{ Sides int }

// Hue: enum Hue { red, darkBlue, url_link; bool get isDark; }
type Hue int

const (
	HueRed Hue = iota
	HueDarkBlue
	HueURLLink
)

func (h Hue) IsDark() bool { return h == HueDarkBlue }

// Registry: class Registry { static Registry get instance; static int count;
// static set mode(int); static final String registryName; static int level; }
type Registry struct{}

func RegistryInstance() *Registry { return nil }

var RegistryCount int

func SetRegistryMode(int)  {}
func RegistryLevel() int   { return 0 } // getter only: the settable static is not exposed
func RegistryName() string { return "" }

// Colors: class Colors { static const red; static const blue; } (statics only).
var Colors = struct{ Red, Blue int }{1, 2}

// Widget: class Widget { const Widget({this.key, required this.child, this.flex}); }
type Widget struct {
	Key   string
	Child any
}

// Viewer: class Viewer { Viewer(this.viewId); final int viewId; String url; List<int> actionIds; }
type Viewer struct {
	ViewID    int
	ActionIDs []int
	url       string
}

func NewViewer(viewID int) *Viewer { return &Viewer{ViewID: viewID} }
func (v *Viewer) URL() string      { return v.url }
func (v *Viewer) SetURL(u string)  { v.url = u }

// Box: class Box { Box.fromCorners(Offset tl, {Radius topLeft, Radius topRight}); bool operator ==(other) }
type Box struct{ m map[int]int }

// BoxCorners holds fromCorners' named parameters (not named ...Options).
type BoxCorners struct{ TopLeft, TopRight float64 }

func NewBoxFromCorners(tl float64, c BoxCorners) Box { return Box{} }
func (b Box) Equal(o Box) bool                       { return true }

// Listenable: abstract class Listenable { const Listenable(); }
type Listenable interface{ AddListener(func()) }

// Node: abstract class Node { Node(int depth); } (no Go base: missing).
type Node interface{ Depth() int }

// Thing: Thing get thing (a top-level getter), renamed in the divergence table.
func ThingInstance() int { return 0 }

// RunIt: void runIt(int a, {required int b, int c, int d}).
func RunIt(a, b int, opts ...RunItOptions) {}

// RunItOptions are runIt's optional named parameters.
type RunItOptions struct{ C int }

// DebugLevel: int debugLevel.
var DebugLevel int

// Merge: Listenable.merge, a factory the divergence table renames.
func ListenableMerge(ls []Listenable) Listenable { return nil }

// Alias: a class re-exported from a third-party package.
type Alias = port.Alias

// Fails: class Fails implements Exception { String toString(); }
type Fails struct{}

func (Fails) Error() string { return "" }

// Mixed: mixin Mixed { void ping(); } → interface Mixed + MixedMixin.
type Mixed interface{ Ping() }

// MixedMixin is Mixed's embeddable struct.
type MixedMixin struct{}

func (*MixedMixin) Ping() {}

// Untested: class Untested { void quiet(); } (no test refers to it).
type Untested struct{}

func (Untested) Quiet() {}

// Event: abstract class Event { Event(this.id, {this.at}); } → EventBase set
// up by InitEvent (Dart's super constructor).
type Event interface{ ID() int }

// EventBase is Event's state.
type EventBase struct{ id, at int }

func (e *EventBase) InitEvent(id int, opts ...EventOptions) {}

// EventOptions are Event's named parameters.
type EventOptions struct{ At int }

// Sealed: abstract class Sealed { Sealed(this.v); } — its base can't be set
// up (unexported state, no Construct or Init): missing.
type Sealed interface{ V() int }

// SealedBase is Sealed's state.
type SealedBase struct{ v int }

// Calm: abstract class Calm { const Calm(); } — a base with state, nothing to
// pass.
type Calm interface{ Level() int }

// CalmBase is Calm's state.
type CalmBase struct{ level int }

// RegistrySetClient is the static setter Registry.client=.
func RegistrySetClient(any) {}

// Drive is the generic method Point.drive<U>() (Go spells it without the
// class).
func Drive[U any](p Point) U { var u U; return u }

// Spin is not generic: it can't stand for the generic method Point.spin<T>().
func Spin() {}

// ToString is Point.toString({minLevel}) with its options (Point also has
// String()).
func (p Point) ToString(opts ...PointToStringOptions) string { return "" }

// PointToStringOptions are toString's named parameters.
type PointToStringOptions struct{ MinLevel int }

// Channel: class Channel { Channel(this.name, [this.codec]); }
type Channel struct{ name string }

func NewChannel(name string, opts ...ChannelOptions) *Channel { return nil }

// ChannelOptions holds Channel's optional positional parameter.
type ChannelOptions struct{ Codec any }

// RegistryTotal is the static setter Registry.total= (a variable).
var RegistryTotal int

// Manager: class Manager { Manager(); } — extended by Go types, set up with
// Construct (Dart's super()).
type Manager struct{ x int }

func (m *Manager) Construct(self any) {}

// Insets is a struct that is not an options struct.
type Insets struct{ Left, Value float64 }

// Pad: class Pad { Pad.of(Insets insets, {double left}); Pad.all(double value); }
type Pad struct{}

func NewPadOf(insets Insets, opts ...PadOfOptions) Pad { return Pad{} }
func NewPadAll(v Insets) Pad                           { return Pad{} }

// PadOfOptions are Pad.of's named parameters.
type PadOfOptions struct{ Left float64 }

// Place: void place(int a, {required int b}) → Place(a, extra, opts).
func Place(a, extra int, opts PlaceOptions) {}

// PlaceOptions are place's named parameters.
type PlaceOptions struct{ B int }

// Get is Box's operator [] (the divergence table maps it).
func (b Box) Get(k int) int { return b.m[k] }

// Hidden: class Hidden { Hidden(this.x); } — no NewHidden, and its state is
// unexported, so no struct literal either.
type Hidden struct{ x int }

// Other: a second re-exported class, used only by the port's test.
type Other = port.Other
