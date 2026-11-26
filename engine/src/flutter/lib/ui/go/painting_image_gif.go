package ui

// GIF decoding for dart:ui's codecs, as Skia's SkWuffsCodec does it: the
// codec exists once the first frame's image descriptor has been read; a
// frame counts once its descriptor has been read (its pixel data may be
// truncated, then decoding it fails); frames are composited on their
// required frame (SkFrameHolder::setAlphaAndRequiredFrame, ported); the
// background color is ignored (restoring to it clears to transparent).

import (
	"compress/lzw"
	"errors"
	"io"
)

type gifFrame struct {
	rect        irect // as encoded
	palette     []byte
	transparent int // -1: none
	disposal    disposal
	delayMs     int
	interlaced  bool
	litWidth    int
	dataStart   int // offset of the first sub-block

	// SkFrame state.
	reportsAlpha  bool
	blendSrc      bool
	requiredFrame int
	hasAlpha      bool
}

type gifGenerator struct {
	data          []byte
	width, height int
	loopCount     int // -1: no NETSCAPE extension
	frames        []*gifFrame
}

func newGIFGenerator(data []byte) imageGenerator {
	if len(data) < 13 {
		return nil
	}
	g := &gifGenerator{data: data, loopCount: -1}
	g.width = int(data[6]) | int(data[7])<<8
	g.height = int(data[8]) | int(data[9])<<8
	flags := data[10]
	pos := 13
	var global []byte
	if flags&0x80 != 0 {
		n := 3 * (2 << (flags & 7))
		if pos+n > len(data) {
			return nil
		}
		global = data[pos : pos+n]
		pos += n
	}
	transparent, disp, delay := -1, disposalKeep, 0
parse:
	for pos < len(data) {
		switch data[pos] {
		case 0x21: // extension
			if pos+2 > len(data) {
				break parse
			}
			label := data[pos+1]
			var body []byte
			p := pos + 2
			for {
				if p >= len(data) {
					break parse
				}
				n := int(data[p])
				if n == 0 {
					p++
					break
				}
				if p+1+n > len(data) {
					break parse
				}
				body = append(body, data[p+1:p+1+n]...)
				p += 1 + n
			}
			switch {
			case label == 0xf9 && len(body) >= 4:
				packed := body[0]
				delay = (int(body[1]) | int(body[2])<<8) * 10
				transparent = -1
				if packed&1 != 0 {
					transparent = int(body[3])
				}
				switch (packed >> 2) & 7 {
				case 2:
					disp = disposalRestoreBG
				case 3:
					disp = disposalRestorePrevious
				default:
					disp = disposalKeep
				}
			case label == 0xff && len(body) >= 14 && (string(body[:11]) == "NETSCAPE2.0" || string(body[:11]) == "ANIMEXTS1.0") && body[11] == 1:
				g.loopCount = int(body[12]) | int(body[13])<<8
			}
			pos = p
		case 0x2c: // image descriptor
			if pos+10 > len(data) {
				break parse
			}
			d := data[pos+1:]
			x, y := int(d[0])|int(d[1])<<8, int(d[2])|int(d[3])<<8
			w, h := int(d[4])|int(d[5])<<8, int(d[6])|int(d[7])<<8
			f := &gifFrame{rect: irect{x, y, x + w, y + h}, palette: global, transparent: transparent,
				disposal: disp, delayMs: delay, interlaced: d[8]&0x40 != 0}
			p := pos + 10
			if d[8]&0x80 != 0 {
				n := 3 * (2 << (d[8] & 7))
				if p+n > len(data) {
					break parse
				}
				f.palette = data[p : p+n]
				p += n
			}
			if p >= len(data) {
				break parse
			}
			f.litWidth = int(data[p])
			f.dataStart = p + 1
			f.reportsAlpha = f.transparent >= 0
			f.blendSrc = !f.reportsAlpha
			g.frames = append(g.frames, f)
			g.setAlphaAndRequiredFrame(f, len(g.frames)-1)
			transparent, disp, delay = -1, disposalKeep, 0
			// Skip the image data.
			p++
			for {
				if p >= len(data) {
					break parse
				}
				n := int(data[p])
				p += 1 + n
				if n == 0 {
					break
				}
			}
			pos = p
		case 0x3b:
			break parse
		default:
			break parse
		}
	}
	if len(g.frames) == 0 {
		return nil
	}
	return g
}

// setAlphaAndRequiredFrame is SkFrameHolder::setAlphaAndRequiredFrame.
func (g *gifGenerator) setAlphaAndRequiredFrame(frame *gifFrame, i int) {
	screen := irect{0, 0, g.width, g.height}
	frameRect := frame.rect.intersect(screen)
	if i == 0 {
		frame.hasAlpha = frame.reportsAlpha || frameRect != screen
		frame.requiredFrame = noFrame
		return
	}
	blendWithPrev := !frame.blendSrc
	if (!frame.reportsAlpha || !blendWithPrev) && frameRect == screen {
		frame.hasAlpha = frame.reportsAlpha
		frame.requiredFrame = noFrame
		return
	}
	prev := g.frames[i-1]
	prevID := i - 1
	for prev.disposal == disposalRestorePrevious {
		if prevID == 0 {
			frame.hasAlpha = true
			frame.requiredFrame = noFrame
			return
		}
		prevID--
		prev = g.frames[prevID]
	}
	clearPrev := prev.disposal == disposalRestoreBG
	prevRect := prev.rect.intersect(screen)
	if clearPrev && (prevRect == screen || prev.requiredFrame == noFrame) {
		frame.hasAlpha = true
		frame.requiredFrame = noFrame
		return
	}
	if frame.reportsAlpha && blendWithPrev {
		frame.requiredFrame = prevID
		frame.hasAlpha = prev.hasAlpha || clearPrev
		return
	}
	for frameRect.contains(prevRect) {
		if prev.requiredFrame == noFrame {
			frame.requiredFrame = noFrame
			frame.hasAlpha = true
			return
		}
		prevID = prev.requiredFrame
		prev = g.frames[prevID]
		prevRect = prev.rect.intersect(screen)
	}
	frame.requiredFrame = prevID
	if prev.disposal == disposalRestoreBG {
		frame.hasAlpha = true
		return
	}
	frame.hasAlpha = prev.hasAlpha || (frame.reportsAlpha && !blendWithPrev)
}

func (g *gifGenerator) info() genInfo {
	inf := genInfo{width: g.width, height: g.height, format: descRGBA8888, alpha: alphaOpaque}
	for _, f := range g.frames {
		if f.hasAlpha {
			inf.alpha = alphaPremul
		}
	}
	return inf
}

func (g *gifGenerator) frameCount() int { return len(g.frames) }

// playCount is the codec's repetition count + 1 (SkWuffsCodec: no
// NETSCAPE extension plays once, a loop count of 0 forever).
func (g *gifGenerator) playCount() int {
	switch g.loopCount {
	case -1:
		return 1
	case 0:
		return infinitePlayCount
	}
	return g.loopCount + 1
}

func (g *gifGenerator) frameInfo(i int) genFrameInfo {
	if i < 0 || i >= len(g.frames) {
		return genFrameInfo{}
	}
	f := g.frames[i]
	return genFrameInfo{requiredFrame: f.requiredFrame, duration: f.delayMs, disposal: f.disposal, blendSrc: f.blendSrc}
}

func (g *gifGenerator) scaledDimensions(float32) (int, int)         { return g.width, g.height }
func (g *gifGenerator) getScaledPixels([]byte, int, int, bool) bool { return false }

// gifSubBlocks reads a GIF's data sub-blocks; a missing byte is
// io.ErrUnexpectedEOF.
type gifSubBlocks struct {
	data []byte
	pos  int
	left int
	done bool
}

func (r *gifSubBlocks) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	for r.left == 0 {
		if r.pos >= len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}
		r.left = int(r.data[r.pos])
		r.pos++
		if r.left == 0 {
			r.done = true
			return 0, io.EOF
		}
	}
	n := min(len(p), r.left, len(r.data)-r.pos)
	if n == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	copy(p, r.data[r.pos:r.pos+n])
	r.pos += n
	r.left -= n
	return n, nil
}

// getPixels decodes frame i onto dst, which holds its required frame
// (SkCodec clears the required frame's rect when it restores to the
// background; an independent frame starts transparent).
func (g *gifGenerator) getPixels(dst []byte, premul bool, i, prior int) bool {
	if i < 0 || i >= len(g.frames) {
		return false
	}
	f := g.frames[i]
	w, h := g.width, g.height
	if f.requiredFrame == noFrame {
		clear(dst)
	} else if prior == f.requiredFrame {
		if p := g.frames[prior]; p.disposal == disposalRestoreBG {
			eraseRect(dst, w, h, p.rect)
		}
	}
	fw, fh := f.rect.right-f.rect.left, f.rect.bottom-f.rect.top
	if f.litWidth < 2 || f.litWidth > 8 {
		return false
	}
	if truncated(g.data, f.dataStart) {
		return false
	}
	indices := make([]byte, fw*fh)
	r := lzw.NewReader(&gifSubBlocks{data: g.data, pos: f.dataStart}, lzw.LSB, f.litWidth)
	n, err := io.ReadFull(r, indices)
	r.Close()
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false // an invalid code
	}
	rows := make([]int, fh)
	if f.interlaced {
		k := 0
		for _, pass := range [][2]int{{0, 8}, {4, 8}, {2, 4}, {1, 2}} {
			for y := pass[0]; y < fh; y += pass[1] {
				rows[k] = y
				k++
			}
		}
	} else {
		for y := range rows {
			rows[y] = y
		}
	}
	for k := 0; k < fh; k++ {
		y := f.rect.top + rows[k]
		if y < 0 || y >= h {
			continue
		}
		for x := 0; x < fw; x++ {
			sx := f.rect.left + x
			if sx < 0 || sx >= w {
				continue
			}
			idx := int(indices[k*fw+x])
			if k*fw+x >= n {
				continue // no data: left as it is
			}
			d := dst[(y*w+sx)*4:]
			if idx == f.transparent {
				if f.blendSrc {
					d[0], d[1], d[2], d[3] = 0, 0, 0, 0
				}
				continue
			}
			if 3*idx+2 < len(f.palette) {
				d[0], d[1], d[2], d[3] = f.palette[3*idx], f.palette[3*idx+1], f.palette[3*idx+2], 255
			} else {
				d[0], d[1], d[2], d[3] = 0, 0, 0, 255
			}
		}
	}
	return true
}

// truncated is whether the sub-blocks starting at pos run past the data.
func truncated(data []byte, pos int) bool {
	for {
		if pos >= len(data) {
			return true
		}
		n := int(data[pos])
		if n == 0 {
			return false
		}
		pos += 1 + n
	}
}
