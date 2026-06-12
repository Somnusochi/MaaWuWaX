// Package pickup implements auto-pickup Custom Recognition and Action for Wuthering Waves.
package pickup

import (
	"image"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

// CheckWhitePctRecognition checks if the F icon is white enough.
type CheckWhitePctRecognition struct{}

var _ maa.CustomRecognitionRunner = &CheckWhitePctRecognition{}

func (r *CheckWhitePctRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition("AutoPick_FIcon", arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}
	whitePct := checkFIconWhitePct(arg.Img, detail.Box)
	if whitePct < 0.5 {
		return nil, false
	}
	return &maa.CustomRecognitionResult{Box: detail.Box}, true
}

// checkFIconWhitePct approximates ok-ww's white_color_percentage check.
// Returns the ratio of near-white pixels in a sampled region around the F icon.
func checkFIconWhitePct(img image.Image, box maa.Rect) float64 {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 1.0 // can't check, allow
	}
	bounds := img.Bounds()
	x1 := int(box[0])
	y1 := int(box[1])
	x2 := x1 + int(box[2])
	y2 := y1 + int(box[3])
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}
	if x2 <= x1 || y2 <= y1 {
		return 1.0
	}
	total := 0
	white := 0
	threshold := uint32(235 << 8) // RGB threshold for "near white"
	for y := y1; y < y2; y += 2 {
		for x := x1; x < x2; x += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			if (r>>8) > threshold>>8 && (g>>8) > threshold>>8 && (b>>8) > threshold>>8 {
				white++
			}
			total++
		}
	}
	if total == 0 {
		return 1.0
	}
	return float64(white) / float64(total)
}
