// Package echofarm implements echo farm Custom Recognition for Wuthering Waves.
package echofarm

import (
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
// FiveToOneMerge is registered by the echonhance package (full-featured version).
// ---------------------------------------------------------------------------
