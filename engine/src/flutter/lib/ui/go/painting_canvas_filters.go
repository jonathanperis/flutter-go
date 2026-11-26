package ui

// DlImageFilter::get_input_device_bounds for the filters a saveLayer's
// paint can carry (display_list/effects/dl_image_filter.cc and
// image_filters/*.cc), used by the clip tracking of Canvas.saveLayer.

import "math"

// mapVectorsAffine is DlImageFilter::map_vectors_affine.
func mapVectorsAffine(m *dlMatrix, x, y float32) (float32, float32) {
	abs := func(v float32) float32 { return float32(math.Abs(float64(v))) }
	return fmuladd32(x, abs(m[0]), float32(y*abs(m[4]))), fmuladd32(x, abs(m[1]), float32(y*abs(m[5])))
}

func matrixIsFinite(m *dlMatrix) bool {
	for _, v := range m {
		if isInfOrNaN32(v) {
			return false
		}
	}
	return true
}

// outsetDeviceBounds and insetDeviceBounds are the DlImageFilter helpers
// on integer rects (the bounds are already rounded out). ENGINE GAP
// (partial): with a perspective matrix they report no bounds (the
// builder then uses its maximal cull rect).
func outsetDeviceBounds(in dlRect, rx, ry float32, m *dlMatrix, sign float64) (dlRect, bool) {
	if !matrixIsFinite(m) || determinant(m) == 0 || hasPerspective2D(m) {
		return dlRect{}, false
	}
	dx, dy := mapVectorsAffine(m, rx, ry)
	var ex, ey float64
	if sign > 0 {
		ex, ey = math.Ceil(float64(dx)), math.Ceil(float64(dy))
	} else {
		ex, ey = -math.Floor(float64(dx)), -math.Floor(float64(dy))
	}
	return dlRect{float32(float64(in.l) - ex), float32(float64(in.t) - ey),
		float32(float64(in.r) + ex), float32(float64(in.b) + ey)}, true
}

// inputDeviceBounds is filter->get_input_device_bounds(output, ctm, input).
func inputDeviceBounds(f ImageFilter, output dlRect, m *dlMatrix) (dlRect, bool) {
	switch f := f.(type) {
	case *gaussianBlurImageFilter:
		sx, sy := float32(f.sigmaX), float32(f.sigmaY)
		if isInfOrNaN32(sx) || isInfOrNaN32(sy) || (sx < scalarNearlyZero && sy < scalarNearlyZero) {
			return output, true // no filter
		}
		return outsetDeviceBounds(output, sx*3.0, sy*3.0, m, 1)
	case *dilateImageFilter:
		rx, ry := float32(f.radiusX), float32(f.radiusY)
		if !morphologyRadiusValid(rx, ry) {
			return output, true
		}
		return outsetDeviceBounds(output, rx, ry, m, -1)
	case *erodeImageFilter:
		rx, ry := float32(f.radiusX), float32(f.radiusY)
		if !morphologyRadiusValid(rx, ry) {
			return output, true
		}
		return outsetDeviceBounds(output, rx, ry, m, 1)
	case *ColorFilter:
		return output, true
	case *composeImageFilter:
		mid, ok := inputDeviceBounds(f.outer, output, m)
		if !ok {
			return dlRect{}, false
		}
		return inputDeviceBounds(f.inner, mid, m)
	}
	return dlRect{}, false
}
