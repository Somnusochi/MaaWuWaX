// Package daily implements daily-routine Custom Actions for Wuthering Waves.
package daily

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

var dailyProgressRe = regexp.MustCompile(`(\d{1,3})\s*/\s*180`)
var dailyPointsRe = regexp.MustCompile(`\d+`)
var lastDailyProgress = -1
var lastDailyRewardReady = false

// DailyNeedsStaminaRecognition gates stamina routing after daily progress has
// been read from the activity book.
type DailyNeedsStaminaRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsStaminaRecognition{}

func (r *DailyNeedsStaminaRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	if lastDailyProgress >= 180 || lastDailyRewardReady {
		log.Info().
			Str("component", "DailyNeedsStamina").
			Int("daily_progress", lastDailyProgress).
			Bool("daily_reward_ready", lastDailyRewardReady).
			Msg("skip stamina routing")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"daily_progress":%d,"reward_ready":%t,"needs_stamina":true}`, lastDailyProgress, lastDailyRewardReady),
	}, true
}

type nightmareModeParam struct {
	NightmareMode string `json:"nightmare_mode"`
	StaminaType   string `json:"stamina_type"`
}

// DailyNeedsNightmareRecognition gates optional NightmareNest routing.
type DailyNeedsNightmareRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsNightmareRecognition{}

func (r *DailyNeedsNightmareRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := nightmareModeParam{NightmareMode: "none"}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DailyNeedsNightmare").Msg("failed to parse param")
		}
	}
	if param.NightmareMode == "" || param.NightmareMode == "none" {
		return nil, false
	}
	if param.NightmareMode == "capture" && param.StaminaType == "tacet" {
		log.Info().
			Str("component", "DailyNeedsNightmare").
			Str("mode", param.NightmareMode).
			Str("stamina_type", param.StaminaType).
			Msg("skip capture-mode nightmare because tacet route already covers daily echo")
		return nil, false
	}
	if lastDailyProgress >= 180 || lastDailyRewardReady {
		log.Info().
			Str("component", "DailyNeedsNightmare").
			Str("mode", param.NightmareMode).
			Str("stamina_type", param.StaminaType).
			Int("daily_progress", lastDailyProgress).
			Bool("daily_reward_ready", lastDailyRewardReady).
			Msg("skip nightmare routing")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"nightmare_mode":%q,"stamina_type":%q,"needs_nightmare":true}`, param.NightmareMode, param.StaminaType),
	}, true
}

// ---------------------------------------------------------------------------
// DailyProgressReader — OCR recognition for daily progress "/180".
// ---------------------------------------------------------------------------

type DailyProgressReader struct{}

var _ maa.CustomRecognitionRunner = &DailyProgressReader{}

func (r *DailyProgressReader) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition("Daily_ProgressOCR", arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Str("component", "DailyProgress").Msg("progress not found")
		return nil, false
	}

	progress := parseDailyProgress(detail.DetailJson)
	if progress < 0 {
		log.Debug().
			Str("component", "DailyProgress").
			Str("text", detail.DetailJson).
			Msg("failed to parse daily progress")
		return nil, false
	}

	lastDailyProgress = progress
	lastDailyRewardReady = readDailyRewardReady(ctx, arg.Img)
	log.Info().
		Str("component", "DailyProgress").
		Int("progress", lastDailyProgress).
		Bool("reward_ready", lastDailyRewardReady).
		Msg("daily progress read")
	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: fmt.Sprintf(`{"progress":%d,"reward_ready":%t}`, lastDailyProgress, lastDailyRewardReady),
	}, true
}

func parseDailyProgress(text string) int {
	text = strings.Trim(text, `"`)
	match := dailyProgressRe.FindStringSubmatch(text)
	if len(match) < 2 {
		return -1
	}
	progress, err := strconv.Atoi(match[1])
	if err != nil {
		return -1
	}
	return progress
}

func readDailyRewardReady(ctx *maa.Context, img image.Image) bool {
	detail, err := ctx.RunRecognition("Daily_RewardPointsOCR", img)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	match := dailyPointsRe.FindString(detail.DetailJson)
	if match == "" {
		return false
	}
	points, err := strconv.Atoi(match)
	if err != nil {
		return false
	}
	return points >= 100
}
