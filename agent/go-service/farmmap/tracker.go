// Package farmmap — tracker.go ports MaaEnd's MapLocator algorithms to Go:
// InferArrowAngle, GenerateMinimapMask, and MotionTracker.
package farmmap

import (
	"image"
	"image/color"
	"math"
	"time"
)

// ─── Arrow Angle Detection (ported from MaaEnd InferYellowArrowRotation) ────

// InferArrowAngle detects the player's facing direction from the minimap arrow.
// Returns angle in degrees [0, 360), or -1 on failure.
//
// Algorithm: extract white pixels near center → connected components →
// find the arrow component → compute moments for centroid →
// find farthest point from centroid (the arrow tip) → atan2.
func InferArrowAngle(minimap *image.RGBA) float64 {
	if minimap == nil {
		return -1
	}

	w, h := minimap.Rect.Dx(), minimap.Rect.Dy()
	cx, cy := w/2, h/2
	radius := 12
	if cx-radius < 0 || cy-radius < 0 || cx+radius > w || cy+radius > h {
		return -1
	}

	// Extract center patch
	patch := minimap.SubImage(image.Rect(cx-radius, cy-radius, cx+radius, cy+radius)).(*image.RGBA)
	pw, ph := patch.Rect.Dx(), patch.Rect.Dy()

	// Find white pixels
	whiteMask := make([]bool, pw*ph)
	for y := range ph {
		for x := range pw {
			r, g, b, _ := patch.At(patch.Rect.Min.X+x, patch.Rect.Min.Y+y).RGBA()
			rv, gv, bv := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			whiteMask[y*pw+x] = rv > 220 && gv > 220 && bv > 220
		}
	}

	// Connected-component labelling (two-pass union-find)
	comp, nComp := labelComponents(whiteMask, pw, ph)
	if nComp == 0 {
		return -1
	}

	// Build component lists
	type compInfo struct {
		pixels []int
		mx, my float64
	}
	comps := make([]compInfo, nComp)
	for i := range comps {
		comps[i].pixels = make([]int, 0)
	}
	for idx, c := range comp {
		if c >= 0 && c < nComp {
			comps[c].pixels = append(comps[c].pixels, idx)
		}
	}

	// Compute centroids and pick the component nearest to center
	pxCx, pxCy := float64(pw)/2, float64(ph)/2
	bestComp := -1
	bestDist := math.MaxFloat64
	for i, c := range comps {
		if len(c.pixels) < 3 {
			continue
		}
		var sx, sy float64
		for _, idx := range c.pixels {
			sx += float64(idx % pw)
			sy += float64(idx / pw)
		}
		n := float64(len(c.pixels))
		comps[i].mx = sx / n
		comps[i].my = sy / n
		d := math.Hypot(comps[i].mx-pxCx, comps[i].my-pxCy)
		if d < bestDist {
			bestDist = d
			bestComp = i
		}
	}

	if bestComp < 0 || bestDist > 5.0 {
		return -1
	}

	// Upscale the arrow component mask 16× for subpixel accuracy
	arrow := comps[bestComp]
	upscale := 16
	uw, uh := pw*upscale, ph*upscale
	upMask := make([]bool, uw*uh)
	for _, idx := range arrow.pixels {
		ox, oy := idx%pw, idx/pw
		for dy := range upscale {
			for dx := range upscale {
				ux, uy := ox*upscale+dx, oy*upscale+dy
				if ux < uw && uy < uh {
					upMask[uy*uw+ux] = true
				}
			}
		}
	}

	// Compute image moments of the upscaled mask
	var m00, m10, m01 float64
	for uy := range uh {
		for ux := range uw {
			if upMask[uy*uw+ux] {
				m00++
				m10 += float64(ux)
				m01 += float64(uy)
			}
		}
	}
	if m00 <= 0 {
		return -1
	}
	centroidX := m10 / m00
	centroidY := m01 / m00

	// Find the pixel farthest from the centroid → the arrow tip
	var maxDist float64
	var tipX, tipY float64
	for uy := range uh {
		for ux := range uw {
			if upMask[uy*uw+ux] {
				d := math.Hypot(float64(ux)-centroidX, float64(uy)-centroidY)
				if d > maxDist {
					maxDist = d
					tipX = float64(ux)
					tipY = float64(uy)
				}
			}
		}
	}

	dx := tipX - centroidX
	dy := tipY - centroidY
	angleDeg := math.Atan2(dx, -dy) * 180 / math.Pi
	if angleDeg < 0 {
		angleDeg += 360
	}
	return angleDeg
}

// labelComponents performs two-pass 4-connected component labelling.
func labelComponents(mask []bool, w, h int) ([]int, int) {
	labels := make([]int, w*h)
	for i := range labels {
		labels[i] = -1
	}
	uf := newUnionFind(w*h/2 + 1)
	nextLabel := 1

	// First pass
	for y := range h {
		for x := range w {
			if !mask[y*w+x] {
				continue
			}
			var neighbors []int
			if y > 0 && labels[(y-1)*w+x] >= 0 {
				neighbors = append(neighbors, labels[(y-1)*w+x])
			}
			if x > 0 && labels[y*w+x-1] >= 0 {
				neighbors = append(neighbors, labels[y*w+x-1])
			}
			if len(neighbors) == 0 {
				labels[y*w+x] = nextLabel
				uf.ensure(nextLabel)
				nextLabel++
			} else {
				minL := neighbors[0]
				for _, l := range neighbors {
					rl := uf.find(l)
					if rl < minL || uf.find(minL) > rl {
						minL = rl
					}
				}
				labels[y*w+x] = minL
				for _, l := range neighbors {
					uf.unite(minL, l)
				}
			}
		}
	}

	// Second pass: flatten labels and remap to 0..N
	remap := make(map[int]int)
	nComp := 0
	for i, l := range labels {
		if l < 0 {
			continue
		}
		root := uf.find(l)
		if _, ok := remap[root]; !ok {
			remap[root] = nComp
			nComp++
		}
		labels[i] = remap[root]
	}
	return labels, nComp
}

type unionFind struct {
	parent []int
}

func newUnionFind(n int) *unionFind {
	return &unionFind{parent: make([]int, n)}
}

func (u *unionFind) ensure(x int) {
	for x >= len(u.parent) {
		u.parent = append(u.parent, len(u.parent))
	}
	if u.parent[x] == 0 {
		u.parent[x] = x
	}
}

func (u *unionFind) find(x int) int {
	u.ensure(x)
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) unite(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		if ra < rb {
			u.parent[rb] = ra
		} else {
			u.parent[ra] = rb
		}
	}
}

// ─── Enhanced Minimap Mask (ported from MaaEnd GenerateMinimapMask) ─────────

// MinimapMaskConfig controls which pixels to exclude from the minimap template.
type MinimapMaskConfig struct {
	BorderMargin         int
	CenterMaskRadius     int
	WhiteThreshold       uint8
	IconDiffThreshold    int
	MinimapDarkThreshold uint8
	DilateRadius         int
	FilterWhite          bool
	FilterColored        bool
	FilterDark           bool
}

// DefaultMinimapMaskConfig returns sensible defaults for WuWa's minimap.
func DefaultMinimapMaskConfig() MinimapMaskConfig {
	return MinimapMaskConfig{
		BorderMargin:         10,
		CenterMaskRadius:     14,
		WhiteThreshold:       200,
		IconDiffThreshold:    40,
		MinimapDarkThreshold: 20,
		DilateRadius:         3,
		FilterWhite:          true,
		FilterColored:        true,
		FilterDark:           true,
	}
}

// GenerateMinimapMask builds a mask for the minimap template that excludes
// UI elements, colored icons, dark areas, and the player arrow center.
// Pixels with mask value 0xFF are kept; 0x00 are excluded.
func GenerateMinimapMask(minimap *image.RGBA, cfg MinimapMaskConfig) []uint8 {
	w, h := minimap.Rect.Dx(), minimap.Rect.Dy()
	cx, cy := w/2, h/2
	radius := min(w, h)/2 - cfg.BorderMargin
	if radius < 0 {
		radius = 0
	}

	mask := make([]uint8, w*h)

	// Fill circle
	for y := range h {
		dy := y - cy
		for x := range w {
			dx := x - cx
			if dx*dx+dy*dy <= radius*radius {
				mask[y*w+x] = 0xFF
			}
		}
	}

	px := minimap.Pix
	stride := minimap.Stride
	ox := minimap.Rect.Min.X
	oy := minimap.Rect.Min.Y

	// Filter bright white UI pixels
	if cfg.FilterWhite {
		for y := range h {
			for x := range w {
				if mask[y*w+x] == 0 {
					continue
				}
				off := (y+oy)*stride + (x+ox)*4
				r, g, b := px[off], px[off+1], px[off+2]
				if r >= cfg.WhiteThreshold && g >= cfg.WhiteThreshold && b >= cfg.WhiteThreshold {
					mask[y*w+x] = 0
				}
			}
		}
	}

	// Filter colored icon pixels (warm/cool UI colors)
	if cfg.FilterColored {
		thresh := cfg.IconDiffThreshold
		for y := range h {
			for x := range w {
				if mask[y*w+x] == 0 {
					continue
				}
				off := (y+oy)*stride + (x+ox)*4
				r, g, b := int(px[off]), int(px[off+1]), int(px[off+2])
				minRG := r
				if g < minRG {
					minRG = g
				}
				// Warm colors: high R/G with significant diff from B
				if r > 100 && g > 100 && minRG-b > thresh {
					mask[y*w+x] = 0
					continue
				}
				// Cool colors: high B exceeding R
				if b > 140 && b > r+50 {
					mask[y*w+x] = 0
				}
			}
		}
	}

	// Dilate mask edges to cover anti-aliased UI fringes
	if cfg.DilateRadius > 0 {
		dilateMask(mask, w, h, cfg.DilateRadius)
	}

	// Filter dark / unexplored areas
	if cfg.FilterDark {
		thresh := cfg.MinimapDarkThreshold
		for y := range h {
			for x := range w {
				if mask[y*w+x] == 0 {
					continue
				}
				off := (y+oy)*stride + (x+ox)*4
				gray := (int(px[off]) + int(px[off+1]) + int(px[off+2])) / 3
				if gray < int(thresh) {
					mask[y*w+x] = 0
				}
			}
		}
	}

	// Cut out center (player arrow)
	r2 := cfg.CenterMaskRadius * cfg.CenterMaskRadius
	for y := cy - cfg.CenterMaskRadius; y <= cy+cfg.CenterMaskRadius; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := cx - cfg.CenterMaskRadius; x <= cx+cfg.CenterMaskRadius; x++ {
			if x < 0 || x >= w {
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				mask[y*w+x] = 0
			}
		}
	}

	return mask
}

// dilateMask erodes the mask region: pixels near the boundary are set to 0.
func dilateMask(mask []uint8, w, h, radius int) {
	if radius <= 0 {
		return
	}
	tmp := make([]uint8, w*h)
	copy(tmp, mask)
	for y := range h {
		for x := range w {
			if mask[y*w+x] == 0 {
				continue
			}
			for dy := -radius; dy <= radius; dy++ {
				ny := y + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -radius; dx <= radius; dx++ {
					nx := x + dx
					if nx < 0 || nx >= w {
						continue
					}
					if mask[ny*w+nx] == 0 && dx*dx+dy*dy <= radius*radius {
						tmp[y*w+x] = 0
						goto nextPixel
					}
				}
			}
		nextPixel:
		}
	}
	copy(mask, tmp)
}

// ─── Motion Tracker (ported from MaaEnd MotionTracker) ──────────────────────

// MotionTracker provides cross-frame position tracking with velocity
// prediction, teleport detection, and stuck detection.
type MotionTracker struct {
	lastPos      *pointBox
	velocityX    float64
	velocityY    float64
	lastTime     time.Time
	lostCount    int
	maxSpeed     float64
	smoothAlpha  float64
	stuckHistory []float64
}

// NewMotionTracker creates a tracker with reasonable defaults.
func NewMotionTracker() *MotionTracker {
	return &MotionTracker{
		maxSpeed:    60.0,
		smoothAlpha: 0.5,
	}
}

// Update records a new position and updates velocity estimate (EMA).
func (t *MotionTracker) Update(pos pointBox) {
	now := time.Now()
	if t.lastPos != nil {
		dt := now.Sub(t.lastTime).Seconds()
		if dt > 0 && dt < 5.0 {
			rawVX := (float64(pos.X) - float64(t.lastPos.X)) / dt
			rawVY := (float64(pos.Y) - float64(t.lastPos.Y)) / dt
			t.velocityX = t.smoothAlpha*rawVX + (1-t.smoothAlpha)*t.velocityX
			t.velocityY = t.smoothAlpha*rawVY + (1-t.smoothAlpha)*t.velocityY
		}
	}
	t.lastPos = &pos
	t.lastTime = now
	t.lostCount = 0
	t.stuckHistory = append(t.stuckHistory, float64(pos.X)+float64(pos.Y))
	if len(t.stuckHistory) > 5 {
		t.stuckHistory = t.stuckHistory[1:]
	}
}

// PredictNextSearchRect returns a search rectangle predicted from the last
// known position and velocity. radius is the base search radius.
func (t *MotionTracker) PredictNextSearchRect(radius int) pointBox {
	if t.lastPos == nil {
		return pointBox{}
	}
	dt := time.Since(t.lastTime).Seconds()
	if dt > 3.0 {
		dt = 3.0
	}
	predX := float64(t.lastPos.X) + t.velocityX*dt
	predY := float64(t.lastPos.Y) + t.velocityY*dt
	return pointBox{
		X: int(predX) - radius,
		Y: int(predY) - radius,
		W: radius * 2,
		H: radius * 2,
	}
}

// IsTracking returns true if we haven't lost tracking for too many frames.
func (t *MotionTracker) IsTracking(maxLost int) bool {
	return t.lastPos != nil && t.lostCount <= maxLost
}

// MarkLost increments the lost-frame counter.
func (t *MotionTracker) MarkLost() {
	t.lostCount++
}

// ForceLost resets all tracking state.
func (t *MotionTracker) ForceLost() {
	t.lostCount = 0
	t.lastPos = nil
	t.velocityX = 0
	t.velocityY = 0
	t.stuckHistory = nil
}

// IsTeleport returns true if the displacement between lastPos and newPos
// exceeds the max-speed threshold, suggesting a teleport.
func (t *MotionTracker) IsTeleport(newPos pointBox) bool {
	if t.lastPos == nil {
		return false
	}
	dt := time.Since(t.lastTime).Seconds()
	if dt <= 0 || dt > 5.0 {
		return false
	}
	dist := math.Hypot(float64(newPos.X-t.lastPos.X), float64(newPos.Y-t.lastPos.Y))
	speed := dist / dt
	return speed > t.maxSpeed
}

// IsStuck checks if position hasn't changed significantly over recent frames.
func (t *MotionTracker) IsStuck() bool {
	if len(t.stuckHistory) < 3 {
		return false
	}
	first := t.stuckHistory[0]
	for _, v := range t.stuckHistory[1:] {
		if math.Abs(v-first) > 3.0 {
			return false
		}
	}
	return true
}

// LastPos returns the last known position, or nil.
func (t *MotionTracker) LastPos() *pointBox {
	return t.lastPos
}

// ─── Mask Application ───────────────────────────────────────────────────────

// ApplyMinimapMask creates a circle-masked minimap template with UI filtering,
// ported from MaaEnd's GenerateMinimapMask approach.
// Mask color is 0x00FF00 (green) for compatibility with minicv NCC-with-mask.
func ApplyMinimapMask(src *image.RGBA, cfg MinimapMaskConfig) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	mask := GenerateMinimapMask(src, cfg)

	const maskColorRGB888 = 0x00FF00
	maskR := uint8((uint32(maskColorRGB888) >> 16) & 0xFF)
	maskG := uint8((uint32(maskColorRGB888) >> 8) & 0xFF)
	maskB := uint8(uint32(maskColorRGB888) & 0xFF)

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if mask[y*w+x] != 0 {
				dst.SetRGBA(x, y, src.At(src.Rect.Min.X+x, src.Rect.Min.Y+y).(color.RGBA))
			} else {
				dst.SetRGBA(x, y, color.RGBA{R: maskR, G: maskG, B: maskB, A: 255})
			}
		}
	}
	return dst
}
