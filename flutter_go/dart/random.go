package dart

import ()

// Random is dart:math's Random with a seed, as the Dart VM implements it
// (sdk/lib/_internal/vm/lib/math_patch.dart, class _Random, Dart 3.50.1's
// SDK): a multiply-with-carry generator with base 2^32 whose seed goes
// through Thomas Wang's 64-bit mix. The same seed gives the same sequence
// as Flutter on the VM (Flutter's Curve2D.generateSamples depends on it).
// Integers are Dart's 64-bit ints, which wrap on overflow as Go's do.
type Random struct{ state uint64 }

// NewRandom is Random(seed).
func NewRandom(seed int) *Random {
	r := &Random{uint64(randomSetupSeed(int64(seed)))}
	// Crank a couple of times to distribute the seed bits a bit further.
	r.nextState()
	r.nextState()
	r.nextState()
	r.nextState()
	return r
}

const (
	randomPow2_32  = 1 << 32
	randomPow2_53D = 1.0 * (1 << 53)
	randomPow2_27D = 1.0 * (1 << 27)
)

// randomSetupSeed is _Random._setupSeed (Thomas Wang 64-bit mix).
func randomSetupSeed(n int64) int64 {
	n = (^n) + (n << 21) // n = (n << 21) - n - 1;
	n = n ^ int64(uint64(n)>>24)
	n = n * 265 // n = (n + (n << 3)) + (n << 8);
	n = n ^ int64(uint64(n)>>14)
	n = n * 21 // n = (n + (n << 2)) + (n << 4);
	n = n ^ int64(uint64(n)>>28)
	n = n + (n << 31)
	if n == 0 {
		n = 0x5a17
	}
	return n
}

// nextState is _Random._nextState: multiply with carry, A from "Numerical
// Recipes 3rd Edition" p.348 B1.
func (r *Random) nextState() {
	const a = 0xffffda61
	stateLo := r.state & 0xFFFFFFFF
	stateHi := r.state >> 32
	r.state = (a * stateLo) + stateHi
}

// NextInt is nextInt(max): a non-negative integer below max, uniformly
// distributed. It panics with a RangeError unless 1 <= max <= 2^32.
func (r *Random) NextInt(max int) int {
	const limit = 0x3FFFFFFF
	if (max <= 0) || ((max > limit) && (max > randomPow2_32)) {
		panic(NewRangeErrorRange(max, 1, randomPow2_32, "max", "Must be positive and <= 2^32"))
	}
	if (max & -max) == max {
		// Fast case for powers of two.
		r.nextState()
		return int(r.state & 0xFFFFFFFF & uint64(max-1))
	}
	var rnd32, result int
	for {
		r.nextState()
		rnd32 = int(r.state & 0xFFFFFFFF)
		result = rnd32 % max
		if !((rnd32 - result + max) > randomPow2_32) {
			return result
		}
	}
}

// NextDouble is nextDouble(): a double in [0, 1) with 53 random bits.
func (r *Random) NextDouble() float64 {
	hi := r.NextInt(1 << 26)
	lo := r.NextInt(1 << 27)
	return (float64(float64(hi)*randomPow2_27D) + float64(lo)) / randomPow2_53D
}

// NextBool is nextBool().
func (r *Random) NextBool() bool { return r.NextInt(2) == 0 }
