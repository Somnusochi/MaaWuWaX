// Package echonhance implements echo enhancement and stat change Custom Actions for Wuthering Waves.
package echonhance

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// ---------------------------------------------------------------------------
// EchoStatReader — OCR reads echo sub-stats from the enhancement UI.
// ---------------------------------------------------------------------------

type EchoStatReader struct{}

var _ maa.CustomRecognitionRunner = &EchoStatReader{}

func (r *EchoStatReader) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition(
		"__EchoStat_OCR",
		arg.Img,
		`{
			"__EchoStat_OCR": {
				"recognition": "OCR",
				"roi": [100, 200, 400, 350]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}

	stats := parseEchoStats(detail.DetailJson)
	jsonStr, _ := sonic.Marshal(stats)

	log.Info().
		Str("component", "EchoStatReader").
		Str("stats", string(jsonStr)).
		Msg("echo stats read")

	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: string(jsonStr),
	}, true
}

// EchoStat represents a single echo sub-stat.
type EchoStat struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// parseEchoStats extracts stat name-value pairs from OCR text.
func parseEchoStats(ocrText string) []EchoStat {
	ocrText = strings.Trim(ocrText, `"`)
	lines := strings.Split(ocrText, "\n")
	var stats []EchoStat
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Try to separate name from value.
		name, value := splitStatLine(line)
		if name != "" {
			stats = append(stats, EchoStat{Name: name, Value: value})
		}
	}
	return stats
}

func splitStatLine(line string) (string, float64) {
	// Common patterns: "暴击 6.9%" or "攻击百分比 8.2%"
	line = strings.TrimSpace(line)

	// Find the numeric part.
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] >= '0' && line[i] <= '9' {
			// Found end of number, scan backwards.
			start := i
			for start > 0 && (line[start-1] >= '0' && line[start-1] <= '9' || line[start-1] == '.') {
				start--
			}
			numStr := strings.TrimRight(line[start:i+1], "%％")
			val, err := strconv.ParseFloat(numStr, 64)
			if err == nil {
				name := normalizeStatName(strings.TrimSpace(line[:start]))
				return name, val
			}
		}
	}
	return normalizeStatName(line), 0
}

func normalizeStatName(name string) string {
	name = strings.TrimSpace(name)
	if strings.Contains(name, "暴击伤害") {
		return "暴击伤害"
	}
	if strings.Contains(name, "暴击") {
		return "暴击"
	}
	if strings.Contains(name, "攻击") {
		if strings.Contains(name, "%") || strings.Contains(name, "百分比") {
			return "攻击百分比"
		}
		return "攻击"
	}
	if strings.Contains(name, "生命") {
		if strings.Contains(name, "%") || strings.Contains(name, "百分比") {
			return "生命百分比"
		}
		return "生命"
	}
	if strings.Contains(name, "防御") {
		if strings.Contains(name, "%") || strings.Contains(name, "百分比") {
			return "防御百分比"
		}
		return "防御"
	}
	if strings.Contains(name, "效率") {
		return "共鸣效率"
	}
	if strings.Contains(name, "普攻") {
		return "普攻伤害加成"
	}
	if strings.Contains(name, "重击") {
		return "重击伤害加成"
	}
	if strings.Contains(name, "解放") {
		return "共鸣解放伤害加成"
	}
	if strings.Contains(name, "技能") {
		return "共鸣技能伤害加成"
	}
	return name
}

// ---------------------------------------------------------------------------
// EchoEnhanceAction — evaluates echo stats and decides lock or discard.
// Returns true for lock (good echo), false for discard.
// ---------------------------------------------------------------------------

type EchoEnhanceAction struct {
	successCount int
	failedCount  int
}

var _ maa.CustomActionRunner = &EchoEnhanceAction{}

type echoEnhanceParam struct {
	NeedDoubleCrit     bool     `json:"need_double_crit"`
	DoubleCritMin      float64  `json:"double_crit_min"`
	FirstCritMin       float64  `json:"first_crit_min"`
	ValidStatsMin      int      `json:"valid_stats_min"`
	FirstMustBeValid   bool     `json:"first_must_be_valid"`
	AllValidBeforeCrit bool     `json:"all_valid_before_crit"`
	ValidStats         []string `json:"valid_stats"`
	PauseOnSuccess     bool     `json:"pause_on_success"`
}

func defaultEchoEnhanceParam() echoEnhanceParam {
	return echoEnhanceParam{
		NeedDoubleCrit:     true,
		DoubleCritMin:      13.8,
		FirstCritMin:       6.9,
		ValidStatsMin:      3,
		FirstMustBeValid:   true,
		AllValidBeforeCrit: true,
		PauseOnSuccess:     true,
		ValidStats: []string{
			"暴击", "暴击伤害", "攻击百分比",
		},
	}
}

func (a *EchoEnhanceAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := defaultEchoEnhanceParam()
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoEnhance").Msg("failed to parse param with direct unmarshal, trying tolerant parse")
			parseEchoEnhanceParamCompat(arg.CustomActionParam, &param)
		}
	}

	ctrl := ctx.GetTasker().GetController()

	// Read echo stats.
	detail, err := ctx.RunRecognition(
		"__EchoEnhance_ReadStats",
		nil,
		`{
			"__EchoEnhance_ReadStats": {
				"recognition": "OCR",
				"roi": [100, 200, 400, 350]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		a.failedCount++
		log.Warn().Str("component", "EchoEnhance").Msg("failed to read echo stats, discarding")
		a.captureEchoSnapshot(ctx, filepath.Join("failed", fmt.Sprintf("%03d_stats_read_failed.png", a.failedCount)))
		// Press Z (discard) directly — pipeline WaitDropped node handles the rest.
		ctrl.PostClickKey(keycode.MustCode("Z")).Wait()
		return true
	}

	stats := parseEchoStats(detail.DetailJson)
	keep, reason := evaluateEcho(stats, param)

	if keep {
		a.successCount++
		a.captureEchoSnapshot(ctx, filepath.Join("success", fmt.Sprintf("%03d.png", a.successCount)))
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Int("success_count", a.successCount).
			Int("failed_count", a.failedCount).
			Bool("pause_on_success", param.PauseOnSuccess).
			Str("result", "keep").
			Msg("echo KEEP — locking (C key)")
		// Press C (lock) — pipeline Esc node handles return.
		ctrl.PostClickKey(keycode.MustCode("C")).Wait()
		if param.PauseOnSuccess {
			time.Sleep(500 * time.Millisecond)
			ctx.GetTasker().PostStop().Wait()
		}
	} else {
		a.failedCount++
		a.captureEchoSnapshot(ctx, filepath.Join("failed", fmt.Sprintf("%03d_%s.png", a.failedCount, sanitizeEchoFilename(reason))))
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Int("success_count", a.successCount).
			Int("failed_count", a.failedCount).
			Str("reason", reason).
			Msg("echo DISCARD (Z key)")
		// Press Z (discard) — pipeline WaitDropped node handles confirmation.
		ctrl.PostClickKey(keycode.MustCode("Z")).Wait()
	}

	return true
}

func parseEchoEnhanceParamCompat(raw string, param *echoEnhanceParam) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return
	}
	if v, ok := readCompatFloat(m, "double_crit_min"); ok {
		param.DoubleCritMin = v
	}
	if v, ok := readCompatFloat(m, "first_crit_min"); ok {
		param.FirstCritMin = v
	}
	if v, ok := readCompatInt(m, "valid_stats_min"); ok {
		param.ValidStatsMin = v
	}
}

func readCompatFloat(m map[string]any, key string) (float64, bool) {
	raw, ok := m[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func readCompatInt(m map[string]any, key string) (int, bool) {
	raw, ok := m[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func evaluateEcho(stats []EchoStat, param echoEnhanceParam) (bool, string) {
	if len(stats) == 0 {
		return false, "empty_stats"
	}

	totalCount := len(stats)
	critRateVal := 0.0
	critDmgVal := 0.0
	hasCritRate := false
	hasCritDmg := false
	checkedFirstCrit := false
	hasEncounteredCrit := false
	invalidCount := 0

	validSet := make(map[string]bool)
	for _, s := range param.ValidStats {
		validSet[s] = true
	}

	for _, stat := range stats {
		name := stat.Name
		val := stat.Value
		isCritStat := name == "暴击" || name == "暴击伤害"

		// Check all-valid-before-crit rule.
		if param.AllValidBeforeCrit &&
			validSet["暴击"] && validSet["暴击伤害"] &&
			!hasEncounteredCrit {
			if !isCritStat && !validSet[name] {
				log.Debug().Str("stat", name).Msg("invalid stat before crit, discard")
				return false, "invalid_before_crit_" + sanitizeEchoFilename(name)
			}
			if isCritStat {
				hasEncounteredCrit = true
			}
		}

		isValid := validSet[name]
		if !isValid {
			invalidCount++
		}

		if name == "暴击" {
			hasCritRate = true
			critRateVal += val
			if validSet["暴击"] && !checkedFirstCrit {
				checkedFirstCrit = true
				if val < param.FirstCritMin {
					log.Debug().Float64("val", val).Msg("first crit rate too low")
					return false, "first_crit_rate_low"
				}
			}
		} else if name == "暴击伤害" {
			hasCritDmg = true
			critDmgVal += val
			if validSet["暴击伤害"] && !checkedFirstCrit {
				checkedFirstCrit = true
				if val/2 < param.FirstCritMin {
					log.Debug().Float64("val", val).Msg("first crit dmg too low")
					return false, "first_crit_dmg_low"
				}
			}
		}
	}

	// Must have double crit check.
	if param.NeedDoubleCrit {
		missing := 0
		if !hasCritRate {
			missing++
		}
		if !hasCritDmg {
			missing++
		}
		remaining := 5 - totalCount
		if remaining < missing {
			log.Debug().Int("missing", missing).Int("remaining", remaining).Msg("cannot get double crit")
			return false, "cannot_complete_double_crit"
		}
	}

	// Double crit total check.
	if hasCritRate && hasCritDmg {
		total := critRateVal + critDmgVal/2
		if total < param.DoubleCritMin {
			log.Debug().Float64("total", total).Msg("double crit total too low")
			return false, "double_crit_total_low"
		}
	}

	// First must be valid check.
	if totalCount == 1 && param.FirstMustBeValid && invalidCount == 1 {
		log.Debug().Msg("first stat is invalid")
		return false, "first_stat_invalid"
	}

	// Valid stats count check.
	validCount := totalCount - invalidCount
	remainingSlots := 5 - totalCount
	if (validCount + remainingSlots) < param.ValidStatsMin {
		log.Debug().Int("max_possible", validCount+remainingSlots).Msg("insufficient valid stats")
		return false, "insufficient_valid_stats"
	}

	return true, "keep"
}

func (a *EchoEnhanceAction) captureEchoSnapshot(ctx *maa.Context, relativeName string) {
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostScreencap().Wait()
	img, err := ctrl.CacheImage()
	if err != nil || img == nil {
		log.Warn().Err(err).Str("component", "EchoEnhance").Msg("failed to capture echo snapshot")
		return
	}

	cropped := cropEchoPanel(img)
	if cropped == nil {
		log.Warn().Str("component", "EchoEnhance").Msg("failed to crop echo snapshot")
		return
	}

	fullPath := filepath.Join("debug", "echo_enhance", relativeName)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		log.Warn().Err(err).Str("component", "EchoEnhance").Str("path", fullPath).Msg("failed to create snapshot directory")
		return
	}

	file, err := os.Create(fullPath)
	if err != nil {
		log.Warn().Err(err).Str("component", "EchoEnhance").Str("path", fullPath).Msg("failed to create snapshot file")
		return
	}
	defer file.Close()

	if err := png.Encode(file, cropped); err != nil {
		log.Warn().Err(err).Str("component", "EchoEnhance").Str("path", fullPath).Msg("failed to encode snapshot")
		return
	}

	log.Info().Str("component", "EchoEnhance").Str("path", fullPath).Msg("saved echo snapshot")
}

func cropEchoPanel(img image.Image) image.Image {
	b := img.Bounds()
	if b.Empty() {
		return nil
	}
	w := b.Dx()
	h := b.Dy()
	x0 := b.Min.X + int(float64(w)*0.09)
	y0 := b.Min.Y + int(float64(h)*0.09)
	x1 := b.Min.X + int(float64(w)*0.37)
	y1 := b.Min.Y + int(float64(h)*0.55)
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	rect := image.Rect(x0, y0, x1, y1).Intersect(b)
	if rect.Empty() {
		return nil
	}

	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			dst.Set(x, y, img.At(rect.Min.X+x, rect.Min.Y+y))
		}
	}
	return dst
}

func sanitizeEchoFilename(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "unknown"
	}
	return out
}

// ---------------------------------------------------------------------------
// EchoChangeSelectAction — selects the target main stat for echo change.
// ---------------------------------------------------------------------------

type EchoChangeSelectAction struct{}

var _ maa.CustomActionRunner = &EchoChangeSelectAction{}

var echoChangeState struct {
	successCount int
}

type echoChangeParam struct {
	TargetStat string `json:"target_stat"`
}

type EchoChangeGuardRecognition struct{}

var _ maa.CustomRecognitionRunner = &EchoChangeGuardRecognition{}

func (r *EchoChangeGuardRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := echoChangeParam{TargetStat: "攻击"}
	if arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoChangeGuard").Msg("failed to parse param")
		}
	}

	detail, err := ctx.RunRecognition(
		"__EchoChange_CurrentMain",
		arg.Img,
		`{
			"__EchoChange_CurrentMain": {
				"recognition": "OCR",
				"roi": [115, 144, 80, 44]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "EchoChangeGuard").Msg("current main stat not found")
		return nil, false
	}

	current := normalizeFiveToOneText(applyTextFix(detail.DetailJson))
	target := normalizeFiveToOneText(param.TargetStat)
	if target != "" && strings.Contains(current, target) {
		log.Warn().
			Str("component", "EchoChangeGuard").
			Str("current", current).
			Str("target", target).
			Msg("target stat is already current stat")
		return nil, false
	}

	payload, _ := sonic.Marshal(map[string]string{
		"current": current,
		"target":  target,
	})
	return &maa.CustomRecognitionResult{Box: detail.Box, Detail: string(payload)}, true
}

func (a *EchoChangeSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoChangeParam{TargetStat: "攻击"}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoChangeSelect").Msg("failed to parse param")
		}
	}

	box, ok := a.findTargetStat(ctx, param.TargetStat)
	if !ok {
		log.Warn().Str("component", "EchoChangeSelect").Str("target", param.TargetStat).Msg("target stat not found")
		return false
	}

	ctx.GetTasker().GetController().PostClick(
		int32(box[0]+box[2]/2),
		int32(box[1]+box[3]/2),
	).Wait()

	log.Info().
		Str("component", "EchoChangeSelect").
		Str("target", param.TargetStat).
		Msg("selected target stat")

	return true
}

func (a *EchoChangeSelectAction) findTargetStat(ctx *maa.Context, target string) (maa.Rect, bool) {
	detail, err := ctx.RunRecognition(
		"__EchoChange_FindStat",
		nil,
		`{
			"__EchoChange_FindStat": {
				"recognition": "OCR",
				"roi": [100, 300, 800, 400]
			}
		}`,
	)
	if err == nil && detail != nil && detail.Hit && detail.Results != nil {
		results := detail.Results.Filtered
		if len(results) == 0 {
			results = detail.Results.All
		}
		for _, result := range results {
			ocr, ok := result.AsOCR()
			if !ok || ocr == nil {
				continue
			}
			if echoChangeTextMatches(ocr.Text, target) {
				return ocr.Box, true
			}
		}
	}

	detail, err = ctx.RunRecognition(
		"__EchoChange_FindStatFallback",
		nil,
		fmt.Sprintf(`{
			"__EchoChange_FindStatFallback": {
				"recognition": "OCR",
				"expected": %q,
				"roi": [100, 300, 800, 400]
			}
		}`, target),
	)
	if err != nil || detail == nil || !detail.Hit {
		return maa.Rect{}, false
	}
	return detail.Box, true
}

func echoChangeTextMatches(text string, target string) bool {
	text = applyTextFix(text)
	text = normalizeFiveToOneText(text)
	target = normalizeFiveToOneText(target)
	if target == "暴击" && strings.Contains(text, "暴击伤害") {
		return false
	}
	return strings.Contains(text, target)
}

type EchoChangeResetAction struct{}

var _ maa.CustomActionRunner = &EchoChangeResetAction{}

func (a *EchoChangeResetAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	echoChangeState.successCount = 0
	log.Info().Str("component", "EchoChange").Msg("reset change-echo counters")
	return true
}

type EchoChangeRecordSuccessAction struct{}

var _ maa.CustomActionRunner = &EchoChangeRecordSuccessAction{}

func (a *EchoChangeRecordSuccessAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	echoChangeState.successCount++
	(&EchoEnhanceAction{}).captureEchoSnapshot(ctx, filepath.Join("change_success", fmt.Sprintf("%03d.png", echoChangeState.successCount)))
	log.Info().
		Str("component", "EchoChange").
		Int("success_count", echoChangeState.successCount).
		Msg("echo main stat changed")
	return true
}

type EchoChangeSummaryAction struct{}

var _ maa.CustomActionRunner = &EchoChangeSummaryAction{}

func (a *EchoChangeSummaryAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().
		Str("component", "EchoChange").
		Int("success_count", echoChangeState.successCount).
		Str("snapshot_dir", filepath.Join("debug", "echo_enhance", "change_success")).
		Msg("change-echo task finished")
	return true
}

// ---------------------------------------------------------------------------
// FiveToOneMergeAction — performs repeated batch fusion in the Data Dock screen.
// ---------------------------------------------------------------------------

type FiveToOneMergeAction struct{}

var _ maa.CustomActionRunner = &FiveToOneMergeAction{}

var fiveToOneState struct {
	setsProcessed int
	fusions       int
	shortages     int
}

type fiveToOneParam struct {
	MaxRounds        int                 `json:"max_rounds"`
	MaxRoundsPerSet  int                 `json:"max_rounds_per_set"`
	Keep             map[string][]string `json:"keep"`
	KeepConfig       string              `json:"keep_config"`
	ContinueOnOCRErr bool                `json:"continue_on_ocr_error"`
}

func (a *FiveToOneMergeAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := fiveToOneParam{MaxRounds: 20, MaxRoundsPerSet: 20}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "FiveToOneMerge").Msg("failed to parse param")
		}
	}
	if param.Keep == nil {
		param.Keep = map[string][]string{}
	}
	if param.KeepConfig != "" {
		var keep map[string][]string
		if err := sonic.Unmarshal([]byte(param.KeepConfig), &keep); err != nil {
			log.Warn().Err(err).Str("component", "FiveToOneMerge").Msg("failed to parse keep_config")
		} else {
			param.Keep = keep
		}
	}
	if param.MaxRoundsPerSet <= 0 {
		param.MaxRoundsPerSet = param.MaxRounds
	}

	fiveToOneState.setsProcessed = 0
	fiveToOneState.fusions = 0
	fiveToOneState.shortages = 0

	claimHandled := false
	for _, setName := range fiveToOneSets {
		if ctx.GetTasker().Stopping() {
			return true
		}
		if !a.mergeSet(ctx, setName, 1, &claimHandled, param) && !param.ContinueOnOCRErr {
			return true
		}
		if !a.mergeSet(ctx, setName, 2, &claimHandled, param) && !param.ContinueOnOCRErr {
			return true
		}
		fiveToOneState.setsProcessed++
	}

	log.Info().
		Str("component", "FiveToOneMerge").
		Int("sets_processed", fiveToOneState.setsProcessed).
		Int("fusions", fiveToOneState.fusions).
		Int("shortages", fiveToOneState.shortages).
		Msg("all configured sets processed")
	return true
}

var fiveToOneSets = []string{
	"凝夜白霜", "熔山裂谷", "彻空冥雷", "啸谷长风", "浮星祛暗", "沉日劫明", "隐世回光", "轻云出月", "不绝余音",
	"凌冽决断之心", "此间永驻之光", "幽夜隐匿之帷", "高天共奏之曲", "无惧浪涛之勇", "流云逝尽之空", "愿戴荣光之旅", "奔狼燎原之焰",
}

var fiveToOneMainStats = []string{
	"攻击力百分比", "生命值百分比", "防御力百分比", "暴击率", "暴击伤害", "共鸣效率",
	"冷凝伤害加成", "热熔伤害加成", "导电伤害加成", "气动伤害加成", "衍射伤害加成", "湮灭伤害加成", "治疗效果加成",
}

var fiveToOneFlatMainStats = []string{"主属性生命值", "主属性攻击力", "主属性防御力"}

var fiveToOneTextFix = map[string]string{
	"凝夜自霜":      "凝夜白霜",
	"主属性灭伤害加成":  "主属性湮灭伤害加成",
	"灭伤害加成":     "主属性湮灭伤害加成",
	"主属性行射伤害加成": "主属性衍射伤害加成",
}

func (a *FiveToOneMergeAction) mergeSet(ctx *maa.Context, setName string, step int, claimHandled *bool, param fiveToOneParam) bool {
	keeps := param.Keep[setName]
	log.Info().
		Str("component", "FiveToOneMerge").
		Str("set", setName).
		Int("step", step).
		Strs("keeps", keeps).
		Msg("processing set")

	if len(keeps) >= len(fiveToOneMainStats) {
		log.Info().Str("component", "FiveToOneMerge").Str("set", setName).Msg("all main stats kept, skipping")
		return true
	}
	if step == 2 && !containsAny(keeps, "攻击力百分比") {
		return true
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(51, 655).Wait()
	time.Sleep(300 * time.Millisecond)
	if step == 1 {
		ctrl.PostClick(794, 590).Wait()
		time.Sleep(80 * time.Millisecond)
	}
	ctrl.PostClick(256, 511).Wait()
	time.Sleep(80 * time.Millisecond)
	ctrl.PostClick(602, 511).Wait()
	time.Sleep(80 * time.Millisecond)
	if step == 1 {
		ctrl.PostClick(909, 511).Wait()
		time.Sleep(80 * time.Millisecond)
	}

	if step == 1 {
		ctrl.PostClick(1146, 396).Wait()
		time.Sleep(500 * time.Millisecond)
		if !a.clickOCR(ctx, setName, maa.Rect{141, 137, 972, 403}) {
			log.Warn().Str("component", "FiveToOneMerge").Str("set", setName).Msg("set filter not found")
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}

	ctrl.PostClick(1146, 533).Wait()
	time.Sleep(500 * time.Millisecond)
	choices := a.ocrStatChoices(ctx)
	if len(choices) == 0 {
		log.Warn().Str("component", "FiveToOneMerge").Str("set", setName).Int("step", step).Msg("stat OCR failed")
		return false
	}

	clicked := 0
	for _, choice := range choices {
		if step == 1 {
			if isFiveToOneFlatStat(choice.Text) {
				continue
			}
			if !isFiveToOneMainStat(choice.Text) {
				continue
			}
			if containsAny(keeps, normalizeFiveToOneText(choice.Text)) {
				continue
			}
		} else if !strings.Contains(normalizeFiveToOneText(choice.Text), "攻击力百分比") {
			continue
		}
		ctrl.PostClick(int32(choice.Box[0]+choice.Box[2]/2), int32(choice.Box[1]+choice.Box[3]/2)).Wait()
		clicked++
		time.Sleep(60 * time.Millisecond)
	}

	log.Info().
		Str("component", "FiveToOneMerge").
		Str("set", setName).
		Int("step", step).
		Int("clicked_filters", clicked).
		Msg("selected merge filters")

	ctrl.PostClick(1037, 605).Wait()
	time.Sleep(500 * time.Millisecond)
	a.runMergeLoop(ctx, claimHandled, param.MaxRoundsPerSet, setName, step)
	return true
}

func (a *FiveToOneMergeAction) runMergeLoop(ctx *maa.Context, claimHandled *bool, maxRounds int, setName string, step int) {
	if maxRounds <= 0 {
		maxRounds = 20
	}
	ctrl := ctx.GetTasker().GetController()
	for round := 0; round < maxRounds; round++ {
		if ctx.GetTasker().Stopping() {
			return
		}

		ctrl.PostClick(333, 655).Wait()
		time.Sleep(500 * time.Millisecond)
		ctrl.PostClick(998, 648).Wait()
		time.Sleep(1200 * time.Millisecond)

		if !*claimHandled && a.clickConfirm(ctx) {
			ctrl.PostClick(627, 396).Wait()
			time.Sleep(200 * time.Millisecond)
			a.clickConfirm(ctx)
			*claimHandled = true
			time.Sleep(800 * time.Millisecond)
		}

		if !a.hasResult(ctx) {
			if a.hasBatchFusion(ctx) {
				ctrl.PostClick(333, 655).Wait()
				fiveToOneState.shortages++
				log.Info().
					Str("component", "FiveToOneMerge").
					Str("set", setName).
					Int("step", step).
					Int("rounds", round).
					Int("fusions", fiveToOneState.fusions).
					Int("shortages", fiveToOneState.shortages).
					Msg("not enough echoes")
				return
			}
			time.Sleep(800 * time.Millisecond)
			continue
		}

		ctrl.PostClick(678, 36).Wait()
		time.Sleep(600 * time.Millisecond)
		ctrl.PostClick(870, 655).Wait()
		time.Sleep(800 * time.Millisecond)
		fiveToOneState.fusions++
	}

	log.Info().
		Str("component", "FiveToOneMerge").
		Str("set", setName).
		Int("step", step).
		Int("rounds", maxRounds).
		Msg("max rounds reached")
}

func (a *FiveToOneMergeAction) clickConfirm(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__FiveToOne_Confirm",
		nil,
		`{
			"__FiveToOne_Confirm": {
				"recognition": "OCR",
				"expected": "确认",
				"roi": [760, 520, 420, 160]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

func (a *FiveToOneMergeAction) hasResult(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__FiveToOne_Result",
		nil,
		`{
			"__FiveToOne_Result": {
				"recognition": "OCR",
				"expected": "获得声骸",
				"roi": [380, 0, 520, 120]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

func (a *FiveToOneMergeAction) hasBatchFusion(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__FiveToOne_BatchFusion",
		nil,
		`{
			"__FiveToOne_BatchFusion": {
				"recognition": "OCR",
				"expected": "批量融合",
				"roi": [900, 600, 380, 120]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

type fiveToOneChoice struct {
	Text string
	Box  maa.Rect
}

func (a *FiveToOneMergeAction) clickOCR(ctx *maa.Context, text string, roi maa.Rect) bool {
	detail, err := ctx.RunRecognition(
		"__FiveToOne_ClickOCR",
		nil,
		fmt.Sprintf(`{
			"__FiveToOne_ClickOCR": {
				"recognition": "OCR",
				"expected": %q,
				"roi": [%d, %d, %d, %d]
			}
		}`, text, roi[0], roi[1], roi[2], roi[3]),
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

func (a *FiveToOneMergeAction) ocrStatChoices(ctx *maa.Context) []fiveToOneChoice {
	detail, err := ctx.RunRecognition(
		"__FiveToOne_StatOCR",
		nil,
		`{
			"__FiveToOne_StatOCR": {
				"recognition": "OCR",
				"roi": [141, 137, 972, 403]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return nil
	}

	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	choices := make([]fiveToOneChoice, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		text := normalizeFiveToOneText(ocr.Text)
		if text == "" {
			continue
		}
		choices = append(choices, fiveToOneChoice{
			Text: text,
			Box:  ocr.Box,
		})
	}
	return choices
}

func normalizeFiveToOneText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, " ", ""))
	for from, to := range fiveToOneTextFix {
		text = strings.ReplaceAll(text, from, to)
	}
	return text
}

func containsAny(values []string, text string) bool {
	text = normalizeFiveToOneText(text)
	for _, value := range values {
		if strings.Contains(text, normalizeFiveToOneText(value)) {
			return true
		}
	}
	return false
}

func isFiveToOneMainStat(text string) bool {
	text = normalizeFiveToOneText(text)
	for _, stat := range fiveToOneMainStats {
		if strings.Contains(text, "主属性"+stat) || strings.Contains(text, stat) {
			return true
		}
	}
	return false
}

func isFiveToOneFlatStat(text string) bool {
	text = normalizeFiveToOneText(text)
	for _, stat := range fiveToOneFlatMainStats {
		if strings.Contains(text, stat) && !strings.Contains(text, "百分比") {
			return true
		}
	}
	return false
}

// ── Language guard (ok-ww: enhance requires zh_CN/zh_TW, change requires zh_CN) ──

func checkLanguageSupport(component string, allowed ...string) bool {
	log.Debug().Str("component", component).Strs("allowed", allowed).Msg("language guard: assuming zh_CN")
	return true
}

// ── OCR text correction mapping (ok-ww: text_fix for ChangeEcho main stat) ──

var textFixMap = map[string]string{
	"凝夜自霜": "凝夜白霜",
	"熔山裂合": "熔山裂谷",
	"彻空真雷": "彻空冥雷",
	"凌冽决断": "凌冽决断之心",
	"此间永驻": "此间永驻之光",
	"幽夜隐匿": "幽夜隐匿之帷",
	"高天共奏": "高天共奏之曲",
	"无惧浪涛": "无惧浪涛之勇",
	"流云逝尽": "流云逝尽之空",
	"愿戴荣光": "愿戴荣光之旅",
	"奔狼燎原": "奔狼燎原之焰",
}

func applyTextFix(text string) string {
	if fixed, ok := textFixMap[text]; ok {
		return fixed
	}
	for wrong, correct := range textFixMap {
		if strings.HasPrefix(text, wrong) || strings.HasPrefix(wrong, text) {
			return correct
		}
	}
	return text
}
