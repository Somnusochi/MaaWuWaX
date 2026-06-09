// Package echofarm implements echo farm Custom Recognition for Wuthering Waves.
package echofarm

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// EchoOrbDetect — detects echo orbs (声骸球) on the ground for collection.
// Uses template matching with echo_orb.png.
// ---------------------------------------------------------------------------

type EchoOrbDetect struct{}

var _ maa.CustomRecognitionRunner = &EchoOrbDetect{}

func (r *EchoOrbDetect) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition(
		"__EchoOrb_Detect",
		arg.Img,
		`{
			"__EchoOrb_Detect": {
				"recognition": "TemplateMatch",
				"template": "echo_orb.png",
				"threshold": 0.5
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Str("component", "EchoOrbDetect").Msg("no echo orb found")
		return nil, false
	}

	log.Info().
		Str("component", "EchoOrbDetect").
		Int32("x", int32(detail.Box[0])).
		Int32("y", int32(detail.Box[1])).
		Msg("echo orb detected")

	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: detail.DetailJson,
	}, true
}

// ---------------------------------------------------------------------------
// FiveToOneMergeAction — batch merges echoes in data dock 5-to-1.
// ---------------------------------------------------------------------------

type FiveToOneMergeAction struct{}

var _ maa.CustomActionRunner = &FiveToOneMergeAction{}

func (a *FiveToOneMergeAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "FiveToOneMerge").Msg("starting batch merge loop")
	ctrl := ctx.GetTasker().GetController()

	merged := 0
	for i := 0; i < 50; i++ {
		if ctx.GetTasker().Stopping() {
			return true
		}

		// Click "Select All".
		ctrl.PostClick(330, 655).Wait()
		time.Sleep(500 * time.Millisecond)

		// Click "Merge".
		ctrl.PostClick(1000, 650).Wait()
		time.Sleep(1000 * time.Millisecond)

		// Check if we got an echo result.
		resultDetail, _ := ctx.RunRecognition(
			"__FiveToOne_Result",
			nil,
			`{
				"__FiveToOne_Result": {
					"recognition": "OCR",
					"expected": "获得声骸"
				}
			}`,
		)
		if resultDetail != nil && resultDetail.Hit {
			// Dismiss result.
			ctrl.PostClick(680, 40).Wait()
			time.Sleep(500 * time.Millisecond)
			merged++
			continue
		}

		// Check if "Batch Fuse" still exists (not enough echoes).
		availDetail, _ := ctx.RunRecognition(
			"__FiveToOne_Avail",
			nil,
			`{
				"__FiveToOne_Avail": {
					"recognition": "OCR",
					"expected": "批量融合",
					"roi": [900, 600, 380, 120]
				}
			}`,
		)
		if availDetail != nil && availDetail.Hit {
			log.Info().Str("component", "FiveToOneMerge").Msg("not enough echoes for more merges")
			break
		}

		// Handle any confirm dialogs.
		confirmDetail, _ := ctx.RunRecognition(
			"__FiveToOne_Confirm",
			nil,
			`{
				"__FiveToOne_Confirm": {
					"recognition": "OCR",
					"expected": "确认",
					"roi": [800, 500, 400, 200]
				}
			}`,
		)
		if confirmDetail != nil && confirmDetail.Hit {
			box := confirmDetail.Box
			ctrl.PostClick(
				int32(box[0]+box[2]/2),
				int32(box[1]+box[3]/2),
			).Wait()
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Info().
		Str("component", "FiveToOneMerge").
		Int("merged", merged).
		Msg("batch merge completed")

	return true
}
