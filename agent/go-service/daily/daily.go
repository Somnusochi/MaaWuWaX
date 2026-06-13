package daily

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

const (
	dailyProgressOCRNode = "Daily_ProgressRawOCR"
	dailyPointsOCRNode   = "Daily_PointsRawOCR"
)

var (
	dailyProgressRe = regexp.MustCompile(`(\d{1,3})\s*/\s*180`)
	dailyNumberRe   = regexp.MustCompile(`\d+`)
)

type DailyNeedNightmareRecognition struct{}
type DailyNeedStaminaRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedNightmareRecognition{}
var _ maa.CustomRecognitionRunner = &DailyNeedStaminaRecognition{}

type dailyNeedNightmareParam struct {
	Mode        string `json:"mode"`
	StaminaType string `json:"stamina_type"`
}

type dailyOCRItem struct {
	text string
	box  maa.Rect
}

func (r *DailyNeedNightmareRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := dailyNeedNightmareParam{}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DailyNeedNightmare").Msg("failed to parse param")
		}
	}

	mode := strings.TrimSpace(strings.ToLower(param.Mode))
	if mode == "" {
		mode = "capture"
	}

	if mode != "capture" {
		return &maa.CustomRecognitionResult{
			Box:    maa.Rect{0, 0, 1, 1},
			Detail: fmt.Sprintf(`{"need_nightmare":true,"mode":%q,"reason":"explicit"}`, mode),
		}, true
	}

	progress, ok := readDailyProgress(ctx, arg)
	if !ok {
		return nil, false
	}
	points, ok := readDailyPoints(ctx, arg)
	if !ok {
		return nil, false
	}
	ready := points >= 100
	need := !ready && strings.TrimSpace(strings.ToLower(param.StaminaType)) != "tacet"
	if !need {
		log.Info().
			Str("component", "DailyNeedNightmare").
			Int("progress", progress).
			Int("points", points).
			Str("stamina_type", param.StaminaType).
			Msg("daily nightmare capture not needed")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"need_nightmare":true,"mode":"capture","progress":%d,"points":%d}`, progress, points),
	}, true
}

func (r *DailyNeedStaminaRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	progress, ok := readDailyProgress(ctx, arg)
	if !ok {
		return nil, false
	}
	points, ok := readDailyPoints(ctx, arg)
	if !ok {
		return nil, false
	}
	ready := points >= 100
	need := !ready && progress < 180
	if !need {
		log.Info().
			Str("component", "DailyNeedStamina").
			Int("progress", progress).
			Int("points", points).
			Msg("daily stamina farm not needed")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"need_stamina":true,"progress":%d,"points":%d}`, progress, points),
	}, true
}

func readDailyProgress(ctx *maa.Context, arg *maa.CustomRecognitionArg) (int, bool) {
	if arg == nil {
		return 0, false
	}
	detail, err := ctx.RunRecognition(dailyProgressOCRNode, arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Err(err).Str("component", "DailyProgress").Msg("daily progress OCR failed")
		return 0, false
	}

	for _, item := range dailyOCRItems(detail) {
		match := dailyProgressRe.FindStringSubmatch(strings.ReplaceAll(item.text, " ", ""))
		if len(match) != 2 {
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		return value, true
	}
	log.Warn().Str("component", "DailyProgress").Msg("daily progress OCR unparsed")
	return 0, false
}

func readDailyPoints(ctx *maa.Context, arg *maa.CustomRecognitionArg) (int, bool) {
	if arg == nil {
		return 0, false
	}
	detail, err := ctx.RunRecognition(dailyPointsOCRNode, arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Err(err).Str("component", "DailyPoints").Msg("daily points OCR failed")
		return 0, false
	}

	best := -1
	for _, item := range dailyOCRItems(detail) {
		for _, candidate := range dailyNumberRe.FindAllString(item.text, -1) {
			value, err := strconv.Atoi(candidate)
			if err != nil {
				continue
			}
			if value > best {
				best = value
			}
		}
	}
	if best < 0 {
		log.Warn().Str("component", "DailyPoints").Msg("daily points OCR unparsed")
		return 0, false
	}
	return best, true
}

func dailyOCRItems(detail *maa.RecognitionDetail) []dailyOCRItem {
	if detail == nil {
		return nil
	}
	if detail.Results == nil {
		return []dailyOCRItem{{text: detail.DetailJson, box: detail.Box}}
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	items := make([]dailyOCRItem, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		items = append(items, dailyOCRItem{text: ocr.Text, box: ocr.Box})
	}
	if len(items) == 0 {
		items = append(items, dailyOCRItem{text: detail.DetailJson, box: detail.Box})
	}
	return items
}
