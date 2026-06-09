// Package combat — screenanalyzer.go provides batch combat-state detection
// by running TemplateMatch recognitions against a single screenshot.
package combat

import (
	"fmt"
	"image"
	"sync"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// Detection labels
// ---------------------------------------------------------------------------

const (
	LabelHasTarget    = "HasTarget"
	LabelDodgePrompt  = "DodgePrompt"
	LabelConFull      = "ConFull"
	LabelLiberation   = "Liberation"
	LabelChar1Alive   = "Char1Alive"
	LabelChar2Alive   = "Char2Alive"
	LabelChar3Alive   = "Char3Alive"
	LabelPickUpF      = "PickUpF"
	LabelInWorld      = "InWorld"
	LabelDead         = "Dead"
)

// ---------------------------------------------------------------------------
// Template recognition specs
// ---------------------------------------------------------------------------

type templateSpec struct {
	label     string
	template  string
	roi       [4]int
	threshold float64
}

var templateSpecs = []templateSpec{
	{LabelHasTarget, "has_target.png", [4]int{400, 200, 800, 600}, 0.7},
	{LabelDodgePrompt, "dodge_prompt.png", [4]int{500, 300, 280, 420}, 0.6},
	{LabelConFull, "con_full_spectro.png", [4]int{0, 500, 400, 220}, 0.5},
	{LabelLiberation, "box_liberation.png", [4]int{1000, 500, 280, 220}, 0.6},
	{LabelChar1Alive, "char_1_text.png", [4]int{10, 570, 100, 130}, 0.5},
	{LabelChar2Alive, "char_2_text.png", [4]int{10, 570, 100, 130}, 0.5},
	{LabelChar3Alive, "char_3_text.png", [4]int{10, 570, 100, 130}, 0.5},
	{LabelPickUpF, "pick_up_f.png", [4]int{300, 200, 680, 480}, 0.65},
	{LabelInWorld, "minimap.png", [4]int{1050, 20, 200, 160}, 0.7},
	{LabelDead, "dead_indicator.png", [4]int{400, 300, 480, 200}, 0.6},
}

// ---------------------------------------------------------------------------
// Detection result
// ---------------------------------------------------------------------------

type detection struct {
	label string
	box   maa.Rect
	hit   bool
}

// ---------------------------------------------------------------------------
// ScreenAnalyzer — caches detection results for a single frame.
// ---------------------------------------------------------------------------

type ScreenAnalyzer struct {
	mu         sync.Mutex
	detections map[string]detection
	frameTime  time.Time
}

var screenAnalyzer = &ScreenAnalyzer{
	detections: make(map[string]detection),
}

// Update captures a screenshot and runs all template recognitions.
func (sa *ScreenAnalyzer) Update(ctx *maa.Context, img image.Image) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	sa.detections = make(map[string]detection, len(templateSpecs))
	sa.frameTime = time.Now()

	for _, spec := range templateSpecs {
		d := runTemplateDetect(ctx, img, spec)
		if d.hit {
			sa.detections[d.label] = d
		}
	}

	log.Debug().
		Str("component", "ScreenAnalyzer").
		Int("hits", len(sa.detections)).
		Msg("frame analyzed")
}

// Has returns true if the given label was detected in the latest frame.
func (sa *ScreenAnalyzer) Has(label string) bool {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	_, ok := sa.detections[label]
	return ok
}

// HasTarget returns true if an enemy target is locked.
func (sa *ScreenAnalyzer) HasTarget() bool { return sa.Has(LabelHasTarget) }

// HasDodge returns true if a dodge prompt is visible.
func (sa *ScreenAnalyzer) HasDodge() bool { return sa.Has(LabelDodgePrompt) }

// HasConFull returns true if concerto energy is full.
func (sa *ScreenAnalyzer) HasConFull() bool { return sa.Has(LabelConFull) }

// HasLiberation returns true if liberation skill is available.
func (sa *ScreenAnalyzer) HasLiberation() bool { return sa.Has(LabelLiberation) }

// HasPickUp returns true if the F-key pickup icon is visible.
func (sa *ScreenAnalyzer) HasPickUp() bool { return sa.Has(LabelPickUpF) }

// IsInWorld returns true if the minimap is detected (character is in the open world).
func (sa *ScreenAnalyzer) IsInWorld() bool { return sa.Has(LabelInWorld) }

// IsDead returns true if the death/revive screen is detected.
func (sa *ScreenAnalyzer) IsDead() bool { return sa.Has(LabelDead) }

// CharAlive checks if a character slot (1-3) shows alive text.
func (sa *ScreenAnalyzer) CharAlive(slot int) bool {
	switch slot {
	case 1:
		return sa.Has(LabelChar1Alive)
	case 2:
		return sa.Has(LabelChar2Alive)
	case 3:
		return sa.Has(LabelChar3Alive)
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Internal: run a single template recognition
// ---------------------------------------------------------------------------

func runTemplateDetect(ctx *maa.Context, img image.Image, spec templateSpec) detection {
	entry := "__SA_" + spec.label
	roi := spec.roi

	detail, err := ctx.RunRecognition(
		entry,
		img,
		formatOverride(entry, spec.template, spec.threshold, roi),
	)
	if err != nil || detail == nil || !detail.Hit {
		return detection{label: spec.label, hit: false}
	}
	return detection{
		label: spec.label,
		box:   detail.Box,
		hit:   true,
	}
}

func formatOverride(entry, template string, threshold float64, roi [4]int) string {
	return formatString(`{
		"%s": {
			"recognition": "TemplateMatch",
			"template": "%s",
			"threshold": %.2f,
			"roi": [%d, %d, %d, %d]
		}
	}`, entry, template, threshold, roi[0], roi[1], roi[2], roi[3])
}

func formatString(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
