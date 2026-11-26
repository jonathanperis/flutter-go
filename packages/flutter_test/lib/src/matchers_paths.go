package flutter_test

// Port of matchers.dart's coversSameAreaAs (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// CoversSameAreaAsOptions are coversSameAreaAs's named parameters
// (SampleSize 0: 20).
type CoversSameAreaAsOptions struct {
	SampleSize int
}

type coversSameAreaAs struct {
	expectedPath                         *ui.Path
	areaToCompare                        ui.Rect
	sampleSize                           int
	maxHorizontalNoise, maxVerticalNoise float64
	random                               *dart.Random
}

// CoversSameAreaAs matches a *ui.Path covering the same area as
// expectedPath, sampled on a sampleSize×sampleSize grid (plus random
// noise from Random(1), as Flutter's) with the size of areaToCompare.
// As in Flutter, the grid starts at the origin, not at areaToCompare's
// top-left corner. The required named areaToCompare is positional.
func CoversSameAreaAs(expectedPath *ui.Path, areaToCompare ui.Rect, opts ...CoversSameAreaAsOptions) Matcher {
	sampleSize := 20
	if len(opts) > 0 && opts[0].SampleSize != 0 {
		sampleSize = opts[0].SampleSize
	}
	return &coversSameAreaAs{
		expectedPath:       expectedPath,
		areaToCompare:      areaToCompare,
		sampleSize:         sampleSize,
		maxHorizontalNoise: areaToCompare.Width() / float64(sampleSize),
		maxVerticalNoise:   areaToCompare.Height() / float64(sampleSize),
		random:             dart.NewRandom(1),
	}
}

func (m *coversSameAreaAs) Matches(item any, matchState map[any]any) bool {
	actualPath, ok := item.(*ui.Path)
	if !ok {
		matchState["failure"] = "is not a Path"
		return false
	}
	n := float64(m.sampleSize)
	for i := 0; i < m.sampleSize; i++ {
		for j := 0; j < m.sampleSize; j++ {
			offset := ui.NewOffset(float64(i)*(m.areaToCompare.Width()/n), float64(j)*(m.areaToCompare.Height()/n))
			if !m.samplePoint(matchState, actualPath, offset) {
				return false
			}
			noise := ui.NewOffset(m.maxHorizontalNoise*m.random.NextDouble(), m.maxVerticalNoise*m.random.NextDouble())
			if !m.samplePoint(matchState, actualPath, offset.Add(noise)) {
				return false
			}
		}
	}
	return true
}

func (m *coversSameAreaAs) samplePoint(matchState map[any]any, actualPath *ui.Path, offset ui.Offset) bool {
	if m.expectedPath.Contains(offset) == actualPath.Contains(offset) {
		return true
	}
	if actualPath.Contains(offset) {
		matchState["failure"] = offset.String() + " is contained in the actual path but not in the expected path"
	} else {
		matchState["failure"] = offset.String() + " is contained in the expected path but not in the actual path"
	}
	return false
}

func (m *coversSameAreaAs) Describe() string { return "covers expected area and only expected area" }

func (m *coversSameAreaAs) DescribeMismatch(_ any, matchState map[any]any) string {
	s, _ := matchState["failure"].(string)
	return s
}
