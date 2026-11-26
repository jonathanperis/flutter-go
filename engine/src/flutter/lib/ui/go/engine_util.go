package ui

import (
	"log"
	"math"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// engineLog is the engine's FML_LOG(ERROR): a line on the log.
var engineLog = func(message string) { log.Print(message) }

// mathMax is dart:math's max on doubles: NaN if either is NaN, and 0.0
// over -0.0.
func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	if a < b {
		return b
	}
	if a == 0 {
		return a + b
	}
	if math.IsNaN(b) {
		return b
	}
	return a
}

// assertNoMessage is Dart's assert(cond) without a message: a failure
// throws an AssertionError whose message is null.
func assertNoMessage(cond bool) {
	if !cond {
		panic(dart.NewAssertionError(nil))
	}
}

func math64IsInfOrNaN(v float64) bool { return math.IsInf(v, 0) || math.IsNaN(v) }

// identityMatrix is DlMatrix().
func identityMatrix() dlMatrix { return dlMatrix{0: 1, 5: 1, 10: 1, 15: 1} }
