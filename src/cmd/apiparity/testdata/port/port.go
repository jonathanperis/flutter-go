// Package port stands for a third-party package's Go port (src/pub/<pkg>)
// that the fixture library re-exports.
package port

// Alias is re-exported by the fixture as a type alias.
type Alias struct{}

// Other is re-exported too.
type Other struct{}

// Ping is a method tests of the port use.
func (Alias) Ping() {}

// AliasOnError is the assignable static Alias.onError, kept in the port.
var AliasOnError func(error)
