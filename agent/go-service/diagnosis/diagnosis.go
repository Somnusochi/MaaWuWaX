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

	combatDetail, _ := ctx.RunRecognition("Diagnosis_Combat", img)
	worldDetail, _ := ctx.RunRecognition("Diagnosis_World", img)
	charDetail, _ := ctx.RunRecognition("Diagnosis_Char", img)
	charDetectDetail, _ := ctx.RunRecognition("Diagnosis_CharacterDetect", img)
	staminaDetail, _ := ctx.RunRecognition("Diagnosis_StaminaReader", img)

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
