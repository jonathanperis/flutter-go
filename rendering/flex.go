package rendering

import "math"

type Axis int

const (
	Horizontal Axis = iota
	Vertical
)

// MainAxisAlignment zero value is Start, as in Flutter.
type MainAxisAlignment int

const (
	MainStart MainAxisAlignment = iota
	MainEnd
	MainCenter
	MainSpaceBetween
	MainSpaceAround
	MainSpaceEvenly
)

// CrossAxisAlignment zero value is Center, as in Flutter.
type CrossAxisAlignment int

const (
	CrossCenter CrossAxisAlignment = iota
	CrossStart
	CrossEnd
	CrossStretch
)

// MainAxisSize zero value is Max, as in Flutter.
type MainAxisSize int

const (
	MainMax MainAxisSize = iota
	MainMin
)

type FlexFit int

const (
	FitTight FlexFit = iota // Expanded
	FitLoose                // Flexible
)

// RenderFlex lays children out in a row or column (Row, Column). It follows
// Flutter's RenderFlex for left-to-right, top-to-bottom layouts; baseline
// alignment and reversed directions are not implemented.
type RenderFlex struct {
	Box
	MultiChild
	direction Axis
	main      MainAxisAlignment
	cross     CrossAxisAlignment
	mainSize  MainAxisSize
}

func NewFlex(d Axis, main MainAxisAlignment, cross CrossAxisAlignment, size MainAxisSize) *RenderFlex {
	return &RenderFlex{direction: d, main: main, cross: cross, mainSize: size}
}

func (r *RenderFlex) Set(d Axis, main MainAxisAlignment, cross CrossAxisAlignment, size MainAxisSize) {
	if d != r.direction || main != r.main || cross != r.cross || size != r.mainSize {
		r.direction, r.main, r.cross, r.mainSize = d, main, cross, size
		MarkNeedsLayout(r)
	}
}

// Axis helpers: work in (main, cross) space and convert at the edges.
func (r *RenderFlex) mc(s Size) (float64, float64) {
	if r.direction == Horizontal {
		return s.W, s.H
	}
	return s.H, s.W
}

func (r *RenderFlex) size(main, cross float64) Size {
	if r.direction == Horizontal {
		return Size{main, cross}
	}
	return Size{cross, main}
}

func (r *RenderFlex) offset(main, cross float64) Offset {
	if r.direction == Horizontal {
		return Offset{main, cross}
	}
	return Offset{cross, main}
}

func (r *RenderFlex) childConstraints(minMain, maxMain, maxCross float64) Constraints {
	minCross := 0.0
	if r.cross == CrossStretch {
		minCross = maxCross
	}
	if r.direction == Horizontal {
		return Constraints{minMain, maxMain, minCross, maxCross}
	}
	return Constraints{minCross, maxCross, minMain, maxMain}
}

func (r *RenderFlex) PerformLayout(c Constraints) Size {
	maxMain, maxCross := c.MaxW, c.MaxH
	if r.direction == Vertical {
		maxMain, maxCross = c.MaxH, c.MaxW
	}
	canFlex := maxMain < Inf

	// 1. Inflexible children get unbounded main-axis space.
	totalFlex, allocated, crossSize := 0, 0.0, 0.0
	var lastFlex RenderBox
	for _, child := range r.children {
		if f := child.Base().Flex; f > 0 {
			totalFlex += f
			lastFlex = child
			continue
		}
		m, x := r.mc(Layout(child, r.childConstraints(0, Inf, maxCross)))
		allocated += m
		crossSize = math.Max(crossSize, x)
	}

	// 2. Flexible children share the remaining space by flex factor.
	freeSpace := 0.0
	if canFlex {
		freeSpace = math.Max(0, maxMain-allocated)
	}
	if totalFlex > 0 {
		perFlex := freeSpace / float64(totalFlex)
		allocatedFlex := 0.0
		for _, child := range r.children {
			b := child.Base()
			if b.Flex == 0 {
				continue
			}
			extent := Inf
			if canFlex {
				if child == lastFlex {
					extent = freeSpace - allocatedFlex
				} else {
					extent = perFlex * float64(b.Flex)
				}
			}
			minExtent := 0.0
			if b.Fit == FitTight {
				minExtent = extent
			}
			m, x := r.mc(Layout(child, r.childConstraints(minExtent, extent, maxCross)))
			allocated += m
			allocatedFlex += extent
			crossSize = math.Max(crossSize, x)
		}
	}

	// 3. Our size.
	idealMain := allocated
	if canFlex && r.mainSize == MainMax {
		idealMain = maxMain
	}
	size := c.Constrain(r.size(idealMain, crossSize))
	actualMain, actualCross := r.mc(size)

	// 4. Position children.
	remaining := math.Max(0, actualMain-allocated)
	n := float64(len(r.children))
	leading, between := 0.0, 0.0
	switch r.main {
	case MainEnd:
		leading = remaining
	case MainCenter:
		leading = remaining / 2
	case MainSpaceBetween:
		if n > 1 {
			between = remaining / (n - 1)
		}
	case MainSpaceAround:
		if n > 0 {
			between = remaining / n
		}
		leading = between / 2
	case MainSpaceEvenly:
		if n > 0 {
			between = remaining / (n + 1)
		}
		leading = between
	}
	pos := leading
	for _, child := range r.children {
		m, x := r.mc(child.Base().Size)
		crossPos := 0.0
		switch r.cross {
		case CrossCenter:
			crossPos = (actualCross - x) / 2
		case CrossEnd:
			crossPos = actualCross - x
		}
		child.Base().Offset = r.offset(pos, crossPos)
		pos += m + between
	}
	return size
}

func (r *RenderFlex) Paint(cv Canvas, offset Offset) {
	// Flutter clips (and warns) on overflow; we only clip.
	m, _ := r.mc(r.Size)
	used := 0.0
	for _, c := range r.children {
		cm, _ := r.mc(c.Base().Size)
		used += cm
	}
	overflow := used > m+epsilon
	if overflow {
		cv.Save()
		cv.ClipRect(RectFrom(offset, r.Size))
	}
	for _, c := range r.children {
		PaintChild(cv, c, offset)
	}
	if overflow {
		cv.Restore()
	}
}
