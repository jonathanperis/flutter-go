package foundation

// Port of diagnostics.dart (Flutter 3.50.1), part 4 of 5: DiagnosticsProperty
// and its subclasses (MessageProperty, StringProperty, DoubleProperty,
// IntProperty, PercentProperty, FlagProperty, IterableProperty,
// EnumProperty, ObjectFlagProperty, FlagsSummary). See diagnostics.go for
// the self-dispatch pattern and the nullable-value convention.

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// ComputePropertyValueCallback computes a property value on demand; it may
// panic, which the property records as its exception.
type ComputePropertyValueCallback[T any] func() T

// ValueToStringOptions are valueToString's named parameters.
type ValueToStringOptions struct{ ParentConfiguration *TextTreeConfiguration }

func valueParentConfig(opts []ValueToStringOptions) *TextTreeConfiguration {
	if len(opts) > 0 {
		return opts[0].ParentConfiguration
	}
	return nil
}

// valueToStringer is the overridable valueToString.
type valueToStringer interface {
	ValueToString(opts ...ValueToStringOptions) string
}

// propertyTypeNamer lets a subclass override how propertyType prints.
type propertyTypeNamer interface{ propertyTypeName() string }

// DiagnosticsProperty is a property with a value of type T. Dart's null is
// the nil of a nilable T (see diagnostics.go).
type DiagnosticsProperty[T any] struct {
	DiagnosticsNodeBase

	description *string
	// ExpandableValue exposes the value's properties and children as this
	// node's.
	ExpandableValue bool
	allowWrap       bool
	allowNameWrap   bool
	// IfNull describes a null value.
	IfNull *string
	// IfEmpty replaces an empty description.
	IfEmpty *string
	// Tooltip is added in parentheses after the value.
	Tooltip *string
	// MissingIfNull raises the level to warning when the value is null.
	MissingIfNull bool
	// DefaultValue is KNoDefaultValue, nil (Dart's null) or a value; a
	// value equal to it lowers the level to fine.
	DefaultValue any

	value         T
	valueComputed bool
	exception     any
	defaultLevel  DiagnosticLevel
	computeValue  ComputePropertyValueCallback[T]
}

// DiagnosticsPropertyOptions are DiagnosticsProperty's named parameters.
// Unset fields take Dart's defaults: DefaultValue kNoDefaultValue (Null for
// Dart's null), ShowName/ShowSeparator/AllowWrap/AllowNameWrap true, Style
// singleLine, Level info.
type DiagnosticsPropertyOptions struct {
	Description     *string
	IfNull          *string
	IfEmpty         *string
	ShowName        *bool
	ShowSeparator   *bool
	DefaultValue    any
	Tooltip         *string
	MissingIfNull   bool
	LinePrefix      string
	ExpandableValue bool
	AllowWrap       *bool
	AllowNameWrap   *bool
	Style           *DiagnosticsTreeStyle
	Level           *DiagnosticLevel
}

// propertyArgs are the resolved constructor arguments shared by every
// property class.
type propertyArgs struct {
	description     *string
	ifNull          *string
	ifEmpty         *string
	showName        *bool
	showSeparator   *bool
	defaultValue    any // already resolved
	tooltip         *string
	missingIfNull   bool
	linePrefix      string
	expandableValue bool
	allowWrap       *bool
	allowNameWrap   *bool
	style           *DiagnosticsTreeStyle
	level           *DiagnosticLevel
	valueComputed   bool
}

func (o DiagnosticsPropertyOptions) args() propertyArgs {
	return propertyArgs{
		description: o.Description, ifNull: resolveString(o.IfNull, nil), ifEmpty: resolveString(o.IfEmpty, nil),
		showName: o.ShowName, showSeparator: o.ShowSeparator,
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), tooltip: resolveString(o.Tooltip, nil),
		missingIfNull: o.MissingIfNull, linePrefix: o.LinePrefix, expandableValue: o.ExpandableValue,
		allowWrap: o.AllowWrap, allowNameWrap: o.AllowNameWrap, style: o.Style, level: o.Level,
	}
}

// propertyName is a positional property name: "" stands for Dart's null.
func propertyName(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}

func (p *DiagnosticsProperty[T]) initProperty(self DiagnosticsNode, name *string, value T, a propertyArgs) {
	style := styleOr(a.style, DiagnosticsTreeStyleSingleLine)
	p.init(name, &style, boolOr(a.showName, true), boolOr(a.showSeparator, true), a.linePrefix)
	p.SetSelf(self)
	p.description = a.description
	p.ExpandableValue = a.expandableValue
	p.allowWrap = boolOr(a.allowWrap, true)
	p.allowNameWrap = boolOr(a.allowNameWrap, true)
	p.IfNull = a.ifNull
	if p.IfNull == nil && a.missingIfNull {
		p.IfNull = strp("MISSING")
	}
	p.IfEmpty = a.ifEmpty
	p.Tooltip = a.tooltip
	p.MissingIfNull = a.missingIfNull
	p.DefaultValue = a.defaultValue
	p.value = value
	p.valueComputed = a.valueComputed
	p.defaultLevel = levelOr(a.level, DiagnosticLevelInfo)
}

// NewDiagnosticsProperty is DiagnosticsProperty<T>(name, value, {...}).
// An empty name stands for Dart's null name.
func NewDiagnosticsProperty[T any](name string, value T, opts ...DiagnosticsPropertyOptions) *DiagnosticsProperty[T] {
	var o DiagnosticsPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &DiagnosticsProperty[T]{}
	a := o.args()
	a.valueComputed = true
	p.initProperty(p, propertyName(name), value, a)
	return p
}

// NewDiagnosticsPropertyLazy is DiagnosticsProperty<T>.lazy: the value is
// computed on first use; a panic while computing it becomes the property's
// exception.
func NewDiagnosticsPropertyLazy[T any](name string, computeValue ComputePropertyValueCallback[T], opts ...DiagnosticsPropertyOptions) *DiagnosticsProperty[T] {
	var o DiagnosticsPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &DiagnosticsProperty[T]{}
	var zero T
	p.initProperty(p, propertyName(name), zero, o.args())
	p.computeValue = computeValue
	return p
}

func (p *DiagnosticsProperty[T]) AllowWrap() bool     { return p.allowWrap }
func (p *DiagnosticsProperty[T]) AllowNameWrap() bool { return p.allowNameWrap }

// ToJsonMap adds the property's fields to DiagnosticsNode's JSON.
func (p *DiagnosticsProperty[T]) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	self := p.outer()
	v := self.Value()
	var properties []map[string]any
	hasProperties := false
	if d, ok := v.(Diagnosticable); ok && !isNull(v) && delegate.ExpandPropertyValues() && delegate.IncludeProperties() &&
		len(self.GetProperties()) == 0 {
		// Exclude children for expanded nodes to avoid cycles.
		zero, no := 0, false
		delegate = delegate.CopyWith(DiagnosticsSerializationDelegateCopyWithOptions{SubtreeDepth: &zero, IncludeProperties: &no})
		properties = DiagnosticsNodeToJsonList(delegate.FilterProperties(d.ToDiagnosticsNode().GetProperties(), self), self, delegate)
		hasProperties = true
	}
	json := p.DiagnosticsNodeBase.ToJsonMap(delegate)
	if hasProperties {
		json["properties"] = properties
	}
	if _, none := p.DefaultValue.(noDefaultValue); !none {
		json["defaultValue"] = dartToString(p.DefaultValue)
	}
	if p.IfEmpty != nil {
		json["ifEmpty"] = *p.IfEmpty
	}
	if p.IfNull != nil {
		json["ifNull"] = *p.IfNull
	}
	if p.Tooltip != nil {
		json["tooltip"] = *p.Tooltip
	}
	json["missingIfNull"] = p.MissingIfNull
	if ex := p.Exception(); ex != nil {
		json["exception"] = dartToString(ex)
	}
	json["propertyType"] = p.propertyTypeString()
	json["defaultLevel"] = p.defaultLevel.String()
	if !isNull(v) {
		_, isD := v.(Diagnosticable)
		_, isN := v.(DiagnosticsNode)
		if isD || isN {
			json["isDiagnosticableValue"] = true
		}
	}
	if n, ok := jsonNumber(v); ok {
		// Workaround, since JSON.stringify replaces infinity and NaN with null,
		// https://github.com/flutter/flutter/issues/39937#issuecomment-529558033)
		json["value"] = n
	}
	if d := deref(v); d == nil {
		json["value"] = nil
	} else if reflect.TypeOf(d).PkgPath() == "" {
		switch x := d.(type) { // only Dart's String and bool, not named Go types
		case string:
			json["value"] = x
		case bool:
			json["value"] = x
		}
	}
	return json
}

func (p *DiagnosticsProperty[T]) propertyTypeString() string {
	if n, ok := p.outer().(propertyTypeNamer); ok {
		return n.propertyTypeName()
	}
	return dartTypeName(p.PropertyType())
}

// ValueToString describes the value; subclasses override it rather than
// ToDescription. DiagnosticableTree values use their toStringShort.
func (p *DiagnosticsProperty[T]) ValueToString(opts ...ValueToStringOptions) string {
	v := p.outer().Value()
	if t, ok := v.(DiagnosticableTree); ok {
		return t.ToStringShort()
	}
	return dartToString(v)
}

// ToDescription is the description, the exception, ifNull, or the
// valueToString, with the tooltip.
func (p *DiagnosticsProperty[T]) ToDescription(opts ...ToDescriptionOptions) string {
	if p.description != nil {
		return p.addTooltip(*p.description)
	}
	if ex := p.Exception(); ex != nil {
		return "EXCEPTION (" + dartRuntimeType(ex) + ")"
	}
	if p.IfNull != nil && p.outer().Value() == nil {
		return p.addTooltip(*p.IfNull)
	}
	result := p.outer().(valueToStringer).ValueToString(ValueToStringOptions{ParentConfiguration: parentConfig(opts)})
	if result == "" && p.IfEmpty != nil {
		result = *p.IfEmpty
	}
	return p.addTooltip(result)
}

func (p *DiagnosticsProperty[T]) addTooltip(text string) string {
	if p.Tooltip == nil {
		return text
	}
	return text + " (" + *p.Tooltip + ")"
}

// PropertyType is the type argument T (Dart's propertyType).
func (p *DiagnosticsProperty[T]) PropertyType() reflect.Type { return reflect.TypeFor[T]() }

// Value is the property value (computing it if lazy), nil for Dart's null.
func (p *DiagnosticsProperty[T]) Value() any {
	p.maybeCacheValue()
	if p.exception != nil || isNull(p.value) { // a failed lazy value is null even for a non-nilable T
		return nil
	}
	return p.value
}

// TypedValue is Value as a T (Dart's value getter is typed T?; Go's Value
// must return any to implement DiagnosticsNode).
func (p *DiagnosticsProperty[T]) TypedValue() T {
	p.maybeCacheValue()
	return p.value
}

// Exception is what computing a lazy value panicked with, or nil.
func (p *DiagnosticsProperty[T]) Exception() any {
	p.maybeCacheValue()
	return p.exception
}

func (p *DiagnosticsProperty[T]) maybeCacheValue() {
	if p.valueComputed {
		return
	}
	p.valueComputed = true
	func() {
		defer func() {
			if r := recover(); r != nil {
				// The error is reported to inspector; rethrowing would destroy the
				// debugging experience.
				p.exception = r
				var zero T
				p.value = zero
			}
		}()
		p.value = p.computeValue()
	}()
}

// IsInteresting is false when the value equals DefaultValue.
func (p *DiagnosticsProperty[T]) IsInteresting() bool {
	if _, none := p.DefaultValue.(noDefaultValue); none {
		return true
	}
	return !dartEquals(p.outer().Value(), p.DefaultValue)
}

// Level is the constructor's level, raised to error when computing the
// value panicked, to warning when a missingIfNull value is null, and
// lowered to fine when the value equals the default.
func (p *DiagnosticsProperty[T]) Level() DiagnosticLevel {
	if p.defaultLevel == DiagnosticLevelHidden {
		return p.defaultLevel
	}
	if p.Exception() != nil {
		return DiagnosticLevelError
	}
	if p.outer().Value() == nil && p.MissingIfNull {
		return DiagnosticLevelWarning
	}
	if !p.IsInteresting() {
		return DiagnosticLevelFine
	}
	return p.defaultLevel
}

// GetProperties are the value's properties when ExpandableValue is set.
func (p *DiagnosticsProperty[T]) GetProperties() []DiagnosticsNode {
	if p.ExpandableValue {
		object := p.outer().Value()
		if n, ok := object.(DiagnosticsNode); ok {
			return n.GetProperties()
		}
		if d, ok := object.(Diagnosticable); ok {
			style := p.outer().Style()
			return d.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}).GetProperties()
		}
	}
	return []DiagnosticsNode{}
}

// GetChildren are the value's children when ExpandableValue is set.
func (p *DiagnosticsProperty[T]) GetChildren() []DiagnosticsNode {
	if p.ExpandableValue {
		object := p.outer().Value()
		if n, ok := object.(DiagnosticsNode); ok {
			return n.GetChildren()
		}
		if d, ok := object.(Diagnosticable); ok {
			style := p.outer().Style()
			return d.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}).GetChildren()
		}
	}
	return []DiagnosticsNode{}
}

// MessageProperty is a message displayed like a property.
type MessageProperty struct {
	DiagnosticsProperty[*struct{}]
}

// MessagePropertyOptions are MessageProperty's named parameters (nil:
// singleLine, info).
type MessagePropertyOptions struct {
	Style *DiagnosticsTreeStyle
	Level *DiagnosticLevel
}

// NewMessageProperty is MessageProperty(name, message, {style, level}).
func NewMessageProperty(name, message string, opts ...MessagePropertyOptions) *MessageProperty {
	var o MessagePropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &MessageProperty{}
	p.initProperty(p, propertyName(name), nil, propertyArgs{
		description: &message, style: o.Style, level: o.Level, defaultValue: KNoDefaultValue, valueComputed: true,
	})
	return p
}

// StringProperty encloses its string value in quotes.
type StringProperty struct {
	DiagnosticsProperty[*string]
	// Quoted says whether the value is enclosed in double quotes.
	Quoted bool
}

// StringPropertyOptions are StringProperty's named parameters (Quoted nil:
// true).
type StringPropertyOptions struct {
	Description  *string
	Tooltip      *string
	ShowName     *bool
	DefaultValue any
	Quoted       *bool
	IfEmpty      *string
	Style        *DiagnosticsTreeStyle
	Level        *DiagnosticLevel
}

func toStringPtr[V string | *string](v V) *string {
	switch x := any(v).(type) {
	case string:
		return &x
	case *string:
		return x
	}
	return nil
}

func toFloatPtr[V float64 | *float64](v V) *float64 {
	switch x := any(v).(type) {
	case float64:
		return &x
	case *float64:
		return x
	}
	return nil
}

func toIntPtr[V int | *int](v V) *int {
	switch x := any(v).(type) {
	case int:
		return &x
	case *int:
		return x
	}
	return nil
}

func toBoolPtr[V bool | *bool](v V) *bool {
	switch x := any(v).(type) {
	case bool:
		return &x
	case *bool:
		return x
	}
	return nil
}

// NewStringProperty is StringProperty(name, value, {...}); value is a
// string or a *string (nil: Dart's null).
func NewStringProperty[V string | *string](name string, value V, opts ...StringPropertyOptions) *StringProperty {
	var o StringPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &StringProperty{Quoted: boolOr(o.Quoted, true)}
	p.initProperty(p, propertyName(name), toStringPtr(value), propertyArgs{
		description: o.Description, tooltip: resolveString(o.Tooltip, nil), showName: o.ShowName,
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), ifEmpty: resolveString(o.IfEmpty, nil),
		style: o.Style, level: o.Level, valueComputed: true,
	})
	return p
}

func (p *StringProperty) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	json["quoted"] = p.Quoted
	return json
}

func (p *StringProperty) ValueToString(opts ...ValueToStringOptions) string {
	text := p.description
	if text == nil {
		text = p.TypedValue()
	}
	if pc := valueParentConfig(opts); pc != nil && !pc.LineBreakProperties && text != nil {
		// Escape linebreaks in multiline strings to avoid confusing output when
		// the parent of this node is trying to display all properties on the same
		// line.
		escaped := strings.ReplaceAll(*text, "\n", `\n`)
		text = &escaped
	}
	if p.Quoted && text != nil {
		// An empty value would not appear empty after being surrounded with
		// quotes so we have to handle this case separately.
		if p.IfEmpty != nil && *text == "" {
			return *p.IfEmpty
		}
		return `"` + *text + `"`
	}
	if text == nil {
		return "null"
	}
	return *text
}

// numberToStringer is _NumProperty's abstract numberToString.
type numberToStringer interface{ NumberToString() string }

// numProperty is _NumProperty: a number with an optional unit.
type numProperty[N int | float64] struct {
	DiagnosticsProperty[*N]
	// Unit is displayed right after the number.
	Unit *string
}

func (p *numProperty[N]) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if p.Unit != nil {
		json["unit"] = *p.Unit
	}
	json["numberToString"] = p.outer().(numberToStringer).NumberToString()
	return json
}

func (p *numProperty[N]) ValueToString(...ValueToStringOptions) string {
	if p.TypedValue() == nil {
		return "null"
	}
	s := p.outer().(numberToStringer).NumberToString()
	if p.Unit != nil {
		return s + *p.Unit
	}
	return s
}

// DoubleProperty is a double with an optional unit, formatted with
// debugFormatDouble.
type DoubleProperty struct {
	numProperty[float64]
}

// DoublePropertyOptions are DoubleProperty's (and DoubleProperty.lazy's)
// named parameters.
type DoublePropertyOptions struct {
	IfNull       *string
	Unit         *string
	Tooltip      *string
	DefaultValue any
	ShowName     *bool
	Style        *DiagnosticsTreeStyle
	Level        *DiagnosticLevel
}

func (o DoublePropertyOptions) args() propertyArgs {
	return propertyArgs{ifNull: resolveString(o.IfNull, nil), tooltip: resolveString(o.Tooltip, nil),
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), showName: o.ShowName, style: o.Style, level: o.Level}
}

// NewDoubleProperty is DoubleProperty(name, value, {...}); value is a
// float64 or a *float64 (nil: Dart's null).
func NewDoubleProperty[V float64 | *float64](name string, value V, opts ...DoublePropertyOptions) *DoubleProperty {
	var o DoublePropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &DoubleProperty{}
	p.Unit = resolveString(o.Unit, nil)
	a := o.args()
	a.valueComputed = true
	p.initProperty(p, propertyName(name), toFloatPtr(value), a)
	return p
}

// NewDoublePropertyLazy is DoubleProperty.lazy: the value is computed on
// first use.
func NewDoublePropertyLazy[V float64 | *float64](name string, computeValue func() V, opts ...DoublePropertyOptions) *DoubleProperty {
	var o DoublePropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &DoubleProperty{}
	p.Unit = resolveString(o.Unit, nil)
	p.initProperty(p, propertyName(name), nil, o.args())
	p.computeValue = func() *float64 { return toFloatPtr(computeValue()) }
	return p
}

// NumberToString is debugFormatDouble(value).
func (p *DoubleProperty) NumberToString() string { return DebugFormatDouble(p.TypedValue()) }

// IntProperty is an int with an optional unit.
type IntProperty struct {
	numProperty[int]
}

// IntPropertyOptions are IntProperty's named parameters.
type IntPropertyOptions struct {
	IfNull       *string
	ShowName     *bool
	Unit         *string
	DefaultValue any
	Style        *DiagnosticsTreeStyle
	Level        *DiagnosticLevel
}

// NewIntProperty is IntProperty(name, value, {...}); value is an int or an
// *int (nil: Dart's null).
func NewIntProperty[V int | *int](name string, value V, opts ...IntPropertyOptions) *IntProperty {
	var o IntPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &IntProperty{}
	p.Unit = resolveString(o.Unit, nil)
	p.initProperty(p, propertyName(name), toIntPtr(value), propertyArgs{
		ifNull: resolveString(o.IfNull, nil), showName: o.ShowName,
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), style: o.Style, level: o.Level, valueComputed: true,
	})
	return p
}

// NumberToString is value.toString().
func (p *IntProperty) NumberToString() string {
	if v := p.TypedValue(); v != nil {
		return strconv.Itoa(*v)
	}
	return "null"
}

// PercentProperty clamps a double to [0, 1] and shows it as a percentage.
type PercentProperty struct {
	DoubleProperty
}

// PercentPropertyOptions are PercentProperty's named parameters.
type PercentPropertyOptions struct {
	IfNull   *string
	ShowName *bool
	Tooltip  *string
	Unit     *string
	Level    *DiagnosticLevel
}

// NewPercentProperty is PercentProperty(name, fraction, {...}).
func NewPercentProperty[V float64 | *float64](name string, fraction V, opts ...PercentPropertyOptions) *PercentProperty {
	var o PercentPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &PercentProperty{}
	p.Unit = resolveString(o.Unit, nil)
	p.initProperty(p, propertyName(name), toFloatPtr(fraction), propertyArgs{
		ifNull: resolveString(o.IfNull, nil), showName: o.ShowName, tooltip: resolveString(o.Tooltip, nil),
		level: o.Level, defaultValue: KNoDefaultValue, valueComputed: true,
	})
	return p
}

func (p *PercentProperty) ValueToString(...ValueToStringOptions) string {
	if p.TypedValue() == nil {
		return "null"
	}
	if p.Unit != nil {
		return p.NumberToString() + " " + *p.Unit
	}
	return p.NumberToString()
}

// NumberToString is the clamped fraction as a percentage with one decimal.
func (p *PercentProperty) NumberToString() string {
	v := p.TypedValue()
	if v == nil {
		return "null"
	}
	return dart.ToStringAsFixed(clampDouble(*v, 0.0, 1.0)*100.0, 1) + "%"
}

// clampDouble is dart:ui's clampDouble (unlike Go's min/max it keeps -0.0).
func clampDouble(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// FlagProperty describes a bool as ifTrue or ifFalse.
type FlagProperty struct {
	DiagnosticsProperty[*bool]
	// IfTrue describes a true value (nil: hidden when true).
	IfTrue *string
	// IfFalse describes a false value (nil: hidden when false).
	IfFalse *string
}

// FlagPropertyOptions are FlagProperty's named parameters (ShowName nil:
// false; DefaultValue nil: Dart's null, FlagProperty's default).
type FlagPropertyOptions struct {
	IfTrue       *string
	IfFalse      *string
	ShowName     *bool
	DefaultValue any
	Level        *DiagnosticLevel
}

// NewFlagProperty is FlagProperty(name, value: value, {...}); value is a
// bool or a *bool. At least one of IfTrue and IfFalse must be set.
func NewFlagProperty[V bool | *bool](name string, value V, opts ...FlagPropertyOptions) *FlagProperty {
	var o FlagPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.IfTrue == nil && o.IfFalse == nil {
		panic("foundation: FlagProperty needs ifTrue or ifFalse")
	}
	p := &FlagProperty{IfTrue: o.IfTrue, IfFalse: o.IfFalse}
	showName := boolOr(o.ShowName, false)
	p.initProperty(p, propertyName(name), toBoolPtr(value), propertyArgs{
		showName: &showName, defaultValue: resolveDefault(o.DefaultValue, nil), level: o.Level, valueComputed: true,
	})
	return p
}

func (p *FlagProperty) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if p.IfTrue != nil {
		json["ifTrue"] = *p.IfTrue
	}
	if p.IfFalse != nil {
		json["ifFalse"] = *p.IfFalse
	}
	return json
}

func (p *FlagProperty) ValueToString(opts ...ValueToStringOptions) string {
	v := p.TypedValue()
	switch {
	case v != nil && *v && p.IfTrue != nil:
		return *p.IfTrue
	case v != nil && !*v && p.IfFalse != nil:
		return *p.IfFalse
	}
	return p.DiagnosticsProperty.ValueToString(opts...)
}

// ShowName is true when the value has no description (the property is then
// hidden).
func (p *FlagProperty) ShowName() bool {
	v := p.TypedValue()
	if v == nil || (*v && p.IfTrue == nil) || (!*v && p.IfFalse == nil) {
		// We are missing a description for the flag value so we need to show the
		// flag name. The property will have DiagnosticLevel.hidden for this case
		// so users will not see this property in this case unless they are
		// displaying hidden properties.
		return true
	}
	return p.DiagnosticsProperty.ShowName()
}

// Level is hidden when the value has no description.
func (p *FlagProperty) Level() DiagnosticLevel {
	v := p.TypedValue()
	switch {
	case v != nil && *v && p.IfTrue == nil:
		return DiagnosticLevelHidden
	case v != nil && !*v && p.IfFalse == nil:
		return DiagnosticLevelHidden
	}
	return p.DiagnosticsProperty.Level()
}

// IterableProperty describes an iterable value (a Go slice; nil is Dart's
// null, an empty non-nil slice is empty).
type IterableProperty[T any] struct {
	DiagnosticsProperty[[]T]
}

// IterablePropertyOptions are IterableProperty's named parameters (IfEmpty
// nil: "[]"; NullString: Dart's null).
type IterablePropertyOptions struct {
	DefaultValue  any
	IfNull        *string
	IfEmpty       *string
	Style         *DiagnosticsTreeStyle
	ShowName      *bool
	ShowSeparator *bool
	Level         *DiagnosticLevel
}

// NewIterableProperty is IterableProperty<T>(name, value, {...}).
func NewIterableProperty[T any](name string, value []T, opts ...IterablePropertyOptions) *IterableProperty[T] {
	var o IterablePropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &IterableProperty[T]{}
	p.initProperty(p, propertyName(name), value, propertyArgs{
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), ifNull: resolveString(o.IfNull, nil),
		ifEmpty: resolveString(o.IfEmpty, strp("[]")), style: o.Style, showName: o.ShowName,
		showSeparator: o.ShowSeparator, level: o.Level, valueComputed: true,
	})
	return p
}

func (p *IterableProperty[T]) propertyTypeName() string {
	return "Iterable<" + dartTypeName(reflect.TypeFor[T]()) + ">"
}

func (p *IterableProperty[T]) ValueToString(opts ...ValueToStringOptions) string {
	value := p.TypedValue()
	if value == nil {
		return "null"
	}
	if len(value) == 0 {
		if p.IfEmpty != nil {
			return *p.IfEmpty
		}
		return "[]"
	}
	formattedValues := make([]string, len(value))
	isDouble := reflect.TypeFor[T]() == reflect.TypeFor[float64]()
	for i, v := range value {
		if isDouble {
			f := any(v).(float64)
			formattedValues[i] = DebugFormatDouble(&f)
		} else {
			formattedValues[i] = dartToString(v)
		}
	}
	if pc := valueParentConfig(opts); pc != nil && !pc.LineBreakProperties {
		// Always display the value as a single line and enclose the iterable
		// value in brackets to avoid ambiguity.
		return "[" + strings.Join(formattedValues, ", ") + "]"
	}
	if isSingleLine(p.Style()) {
		return strings.Join(formattedValues, ", ")
	}
	return strings.Join(formattedValues, "\n")
}

// Level is fine for an empty value when IfEmpty is null.
func (p *IterableProperty[T]) Level() DiagnosticLevel {
	value := p.TypedValue()
	if p.IfEmpty == nil && value != nil && len(value) == 0 && p.DiagnosticsProperty.Level() != DiagnosticLevelHidden {
		return DiagnosticLevelFine
	}
	return p.DiagnosticsProperty.Level()
}

func (p *IterableProperty[T]) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if value := p.TypedValue(); value != nil {
		values := make([]string, len(value))
		for i, v := range value {
			values[i] = dartToString(v)
		}
		json["values"] = values
	}
	return json
}

// EnumProperty shows an enum value by its name (a Go enum's String()).
type EnumProperty[T any] struct {
	DiagnosticsProperty[T]
}

// EnumPropertyOptions are EnumProperty's named parameters.
type EnumPropertyOptions struct {
	DefaultValue any
	Level        *DiagnosticLevel
}

// NewEnumProperty is EnumProperty<T>(name, value, {defaultValue, level});
// use a pointer T for a nullable enum.
func NewEnumProperty[T any](name string, value T, opts ...EnumPropertyOptions) *EnumProperty[T] {
	var o EnumPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &EnumProperty[T]{}
	p.initProperty(p, propertyName(name), value, propertyArgs{
		defaultValue: resolveDefault(o.DefaultValue, KNoDefaultValue), level: o.Level, valueComputed: true,
	})
	return p
}

func (p *EnumProperty[T]) ValueToString(...ValueToStringOptions) string {
	v := deref(p.Value())
	if v == nil {
		return "null"
	}
	if isEnum(v) {
		return v.(fmt.Stringer).String() // value.name
	}
	return dartToString(v)
}

// ObjectFlagProperty describes whether a value is present (ifPresent) or
// null (ifNull), rather than the value itself.
type ObjectFlagProperty[T any] struct {
	DiagnosticsProperty[T]
	// IfPresent describes a non-null value (nil: hidden when present).
	IfPresent *string
}

// ObjectFlagPropertyOptions are ObjectFlagProperty's named parameters
// (ShowName nil: false). At least one of IfPresent and IfNull is required.
type ObjectFlagPropertyOptions struct {
	IfPresent *string
	IfNull    *string
	ShowName  *bool
	Level     *DiagnosticLevel
}

// NewObjectFlagProperty is ObjectFlagProperty<T>(name, value, {...}).
func NewObjectFlagProperty[T any](name string, value T, opts ...ObjectFlagPropertyOptions) *ObjectFlagProperty[T] {
	var o ObjectFlagPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	ifNull := resolveString(o.IfNull, nil)
	if o.IfPresent == nil && ifNull == nil {
		panic("foundation: ObjectFlagProperty needs ifPresent or ifNull")
	}
	p := &ObjectFlagProperty[T]{IfPresent: o.IfPresent}
	showName := boolOr(o.ShowName, false)
	p.initProperty(p, propertyName(name), value, propertyArgs{
		ifNull: ifNull, showName: &showName, level: o.Level, defaultValue: KNoDefaultValue, valueComputed: true,
	})
	return p
}

// ObjectFlagPropertyHasOptions are ObjectFlagProperty.has's named
// parameters.
type ObjectFlagPropertyHasOptions struct{ Level *DiagnosticLevel }

// NewObjectFlagPropertyHas is ObjectFlagProperty<T>.has: "has <name>" when
// the value is present.
func NewObjectFlagPropertyHas[T any](name string, value T, opts ...ObjectFlagPropertyHasOptions) *ObjectFlagProperty[T] {
	var o ObjectFlagPropertyHasOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &ObjectFlagProperty[T]{IfPresent: strp("has " + name)}
	p.initProperty(p, propertyName(name), value, propertyArgs{
		showName: pFalse, level: o.Level, defaultValue: KNoDefaultValue, valueComputed: true,
	})
	return p
}

func (p *ObjectFlagProperty[T]) ValueToString(opts ...ValueToStringOptions) string {
	if p.Value() != nil {
		if p.IfPresent != nil {
			return *p.IfPresent
		}
	} else {
		if p.IfNull != nil {
			return *p.IfNull
		}
	}
	return p.DiagnosticsProperty.ValueToString(opts...)
}

// ShowName is true when the value has no description (the property is then
// hidden).
func (p *ObjectFlagProperty[T]) ShowName() bool {
	if (p.Value() != nil && p.IfPresent == nil) || (p.Value() == nil && p.IfNull == nil) {
		// We are missing a description for the flag value so we need to show the
		// flag name. The property will have DiagnosticLevel.hidden for this case
		// so users will not see this property in this case unless they are
		// displaying hidden properties.
		return true
	}
	return p.DiagnosticsProperty.ShowName()
}

// Level is hidden when the value has no description.
func (p *ObjectFlagProperty[T]) Level() DiagnosticLevel {
	if p.Value() != nil {
		if p.IfPresent == nil {
			return DiagnosticLevelHidden
		}
	} else {
		if p.IfNull == nil {
			return DiagnosticLevelHidden
		}
	}
	return p.DiagnosticsProperty.Level()
}

func (p *ObjectFlagProperty[T]) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if p.IfPresent != nil {
		json["ifPresent"] = *p.IfPresent
	}
	return json
}

// FlagsSummary summarizes several flags: the keys of the non-null entries.
// Dart's Map<String, T?> keeps insertion order, so the value is an ordered
// slice of entries.
type FlagsSummary[T any] struct {
	DiagnosticsProperty[[]dart.MapEntry[string, T]]
}

// FlagsSummaryOptions are FlagsSummary's named parameters.
type FlagsSummaryOptions struct {
	IfEmpty       *string
	ShowName      *bool
	ShowSeparator *bool
	Level         *DiagnosticLevel
}

// NewFlagsSummary is FlagsSummary<T>(name, value, {...}).
func NewFlagsSummary[T any](name string, value []dart.MapEntry[string, T], opts ...FlagsSummaryOptions) *FlagsSummary[T] {
	var o FlagsSummaryOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &FlagsSummary[T]{}
	p.initProperty(p, propertyName(name), value, propertyArgs{
		ifEmpty: resolveString(o.IfEmpty, nil), showName: o.ShowName, showSeparator: o.ShowSeparator,
		level: o.Level, defaultValue: KNoDefaultValue, valueComputed: true,
	})
	return p
}

func (p *FlagsSummary[T]) ValueToString(opts ...ValueToStringOptions) string {
	if !p.hasNonNullEntry() && p.IfEmpty != nil {
		return *p.IfEmpty
	}
	formattedValues := p.formattedValues()
	if pc := valueParentConfig(opts); pc != nil && !pc.LineBreakProperties {
		// Always display the value as a single line and enclose the iterable
		// value in brackets to avoid ambiguity.
		return "[" + strings.Join(formattedValues, ", ") + "]"
	}
	if isSingleLine(p.Style()) {
		return strings.Join(formattedValues, ", ")
	}
	return strings.Join(formattedValues, "\n")
}

// Level is hidden when no entry is non-null and IfEmpty is null.
func (p *FlagsSummary[T]) Level() DiagnosticLevel {
	if !p.hasNonNullEntry() && p.IfEmpty == nil {
		return DiagnosticLevelHidden
	}
	return p.DiagnosticsProperty.Level()
}

func (p *FlagsSummary[T]) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if len(p.TypedValue()) != 0 {
		json["values"] = p.formattedValues()
	}
	return json
}

func (p *FlagsSummary[T]) hasNonNullEntry() bool {
	for _, e := range p.TypedValue() {
		if !isNull(e.Value) {
			return true
		}
	}
	return false
}

// formattedValues are the keys of the non-null entries, in order.
func (p *FlagsSummary[T]) formattedValues() []string {
	values := []string{}
	for _, e := range p.TypedValue() {
		if !isNull(e.Value) {
			values = append(values, e.Key)
		}
	}
	return values
}
