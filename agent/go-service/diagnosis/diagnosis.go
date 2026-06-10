// Package diagnosis implements diagnostic Custom Actions.
package diagnosis

import (
	"fmt"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type SnapshotAction struct{}

var _ maa.CustomActionRunner = &SnapshotAction{}

func (a *SnapshotAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	ctrl := ctx.GetTasker().GetController()
	start := time.Now()
	ctrl.PostScreencap().Wait()
	captureCost := time.Since(start)
	img, err := ctrl.CacheImage()
	if err != nil {
		log.Warn().Err(err).Str("component", "Diagnosis").Msg("failed to read cached image")
		return true
	}

	combatDetail, _ := ctx.RunRecognition(
		"__Diagnosis_Combat",
		img,
		`{
			"__Diagnosis_Combat": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "TemplateMatch", "template": "has_target.png", "threshold": 0.6},
					{"recognition": "TemplateMatch", "template": "has_target_cloud.png", "threshold": 0.6}
				]
			}
		}`,
	)
	worldDetail, _ := ctx.RunRecognition(
		"__Diagnosis_World",
		img,
		`{
			"__Diagnosis_World": {
				"recognition": "TemplateMatch",
				"template": "minimap.png",
				"threshold": 0.7,
				"roi": [1050, 20, 200, 160]
			}
		}`,
	)
	charDetail, _ := ctx.RunRecognition(
		"__Diagnosis_Char",
		img,
		`{
			"__Diagnosis_Char": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "TemplateMatch", "template": "char_1_text.png", "threshold": 0.7},
					{"recognition": "TemplateMatch", "template": "char_2_text.png", "threshold": 0.7},
					{"recognition": "TemplateMatch", "template": "char_3_text.png", "threshold": 0.7}
				]
			}
		}`,
	)
	charDetectDetail, _ := ctx.RunRecognition(
		"__Diagnosis_CharacterDetect",
		img,
		`{
			"__Diagnosis_CharacterDetect": {
				"recognition": "Custom",
				"custom_recognition": "CharacterDetect"
			}
		}`,
	)
	staminaDetail, _ := ctx.RunRecognition(
		"__Diagnosis_StaminaReader",
		img,
		`{
			"__Diagnosis_StaminaReader": {
				"recognition": "Custom",
				"custom_recognition": "StaminaReader"
			}
		}`,
	)

	log.Info().
		Str("component", "Diagnosis").
		Str("capture_cost", fmt.Sprintf("%.3fs", captureCost.Seconds())).
		Bool("in_world", worldDetail != nil && worldDetail.Hit).
		Bool("in_combat", combatDetail != nil && combatDetail.Hit).
		Bool("has_team", charDetail != nil && charDetail.Hit).
		Str("character_detect", detailJSONString(charDetectDetail)).
		Str("stamina", detailJSONString(staminaDetail)).
		Msg("diagnosis snapshot")
	time.Sleep(500 * time.Millisecond)
	return true
}

func detailJSONString(detail *maa.RecognitionDetail) string {
	if detail == nil {
		return ""
	}
	return detail.DetailJson
}
