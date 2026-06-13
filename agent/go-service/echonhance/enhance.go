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

	counterpkg "github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/counter"
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
		"EchoEnhance_StatsOCR",
		arg.Img,
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
	NeedDoubleCrit     bool                `json:"need_double_crit"`
	DoubleCritMin      float64             `json:"double_crit_min"`
	FirstCritMin       float64             `json:"first_crit_min"`
	ValidStatsMin      int                 `json:"valid_stats_min"`
	MaxStatSlots       int                 `json:"max_stat_slots"`
	FirstMustBeValid   bool                `json:"first_must_be_valid"`
	AllValidBeforeCrit bool                `json:"all_valid_before_crit"`
	ValidStats         []string            `json:"valid_stats"`
	CritRateStat       string              `json:"crit_rate_stat"`
	CritDmgStat        string              `json:"crit_dmg_stat"`
	StatAliases        map[string][]string `json:"stat_aliases"`
	TextFix            map[string]string   `json:"text_fix"`
	StatsOCRNode       string              `json:"stats_ocr_node"`
	KeepKey            string              `json:"keep_key"`
	DiscardKey         string              `json:"discard_key"`
	CaptureSuccess     bool                `json:"capture_success"`
	CaptureFailure     bool                `json:"capture_failure"`
	CaptureReadFail    bool                `json:"capture_read_fail"`
}

func defaultEchoEnhanceParam() echoEnhanceParam {
	return echoEnhanceParam{
		NeedDoubleCrit:     true,
		DoubleCritMin:      13.8,
		FirstCritMin:       6.9,
		ValidStatsMin:      3,
		MaxStatSlots:       5,
		FirstMustBeValid:   true,
		AllValidBeforeCrit: true,
		CritRateStat:       "暴击",
		CritDmgStat:        "暴击伤害",
		StatsOCRNode:       "EchoEnhance_StatsOCR",
		KeepKey:            "C",
		DiscardKey:         "Z",
		CaptureSuccess:     true,
		CaptureFailure:     true,
		CaptureReadFail:    true,
		ValidStats: []string{
			"暴击", "暴击伤害", "攻击百分比",
		},
		StatAliases: map[string][]string{
			"暴击":       {"暴击率"},
			"攻击百分比":    {"攻击力百分比", "攻击%"},
			"生命百分比":    {"生命值百分比", "生命%"},
			"防御百分比":    {"防御力百分比", "防御%"},
			"普攻伤害加成":   {"普攻加成"},
			"重击伤害加成":   {"重击加成"},
			"共鸣技能伤害加成": {"技能伤害加成"},
			"共鸣解放伤害加成": {"解放伤害加成"},
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
	detail, err := ctx.RunRecognition(param.StatsOCRNode, nil)
	if err != nil || detail == nil || !detail.Hit {
		a.failedCount++
		log.Warn().Str("component", "EchoEnhance").Msg("failed to read echo stats, discarding")
		if param.CaptureReadFail {
			a.captureEchoSnapshot(ctx, filepath.Join("failed", fmt.Sprintf("%03d_stats_read_failed.png", a.failedCount)))
		}
		// Press Z (discard) directly — pipeline WaitDropped node handles the rest.
		ctrl.PostClickKey(keycode.MustCode(param.DiscardKey)).Wait()
		return true
	}

	stats := normalizeEchoStats(parseEchoStats(detail.DetailJson), param)
	keep, reason := evaluateEcho(stats, param)

	if keep {
		a.successCount++
		if param.CaptureSuccess {
			a.captureEchoSnapshot(ctx, filepath.Join("success", fmt.Sprintf("%03d.png", a.successCount)))
		}
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Int("success_count", a.successCount).
			Int("failed_count", a.failedCount).
			Str("result", "keep").
			Msg("echo KEEP — locking (C key)")
		// Press C (lock) — pipeline Esc node handles return.
		ctrl.PostClickKey(keycode.MustCode(param.KeepKey)).Wait()
	} else {
		a.failedCount++
		if param.CaptureFailure {
			a.captureEchoSnapshot(ctx, filepath.Join("failed", fmt.Sprintf("%03d_%s.png", a.failedCount, sanitizeEchoFilename(reason))))
		}
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Int("success_count", a.successCount).
			Int("failed_count", a.failedCount).
			Str("reason", reason).
			Msg("echo DISCARD (Z key)")
		// Press Z (discard) — pipeline WaitDropped node handles confirmation.
		ctrl.PostClickKey(keycode.MustCode(param.DiscardKey)).Wait()
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
	if v, ok := readCompatInt(m, "max_stat_slots"); ok {
		param.MaxStatSlots = v
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
	if param.MaxStatSlots <= 0 {
		param.MaxStatSlots = 5
	}

	critRateStat := canonicalizeEchoEnhanceStatName(param.CritRateStat, param)
	if critRateStat == "" {
		critRateStat = "暴击"
	}
	critDmgStat := canonicalizeEchoEnhanceStatName(param.CritDmgStat, param)
	if critDmgStat == "" {
		critDmgStat = "暴击伤害"
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
		name := canonicalizeEchoEnhanceStatName(s, param)
		if name == "" {
			name = normalizeStatName(s)
		}
		validSet[name] = true
	}

	for _, stat := range stats {
		name := stat.Name
		val := stat.Value
		isCritStat := name == critRateStat || name == critDmgStat

		// Check all-valid-before-crit rule.
		if param.AllValidBeforeCrit &&
			validSet[critRateStat] && validSet[critDmgStat] &&
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

		if name == critRateStat {
			hasCritRate = true
			critRateVal += val
			if validSet[critRateStat] && !checkedFirstCrit {
				checkedFirstCrit = true
				if val < param.FirstCritMin {
					log.Debug().Float64("val", val).Msg("first crit rate too low")
					return false, "first_crit_rate_low"
				}
			}
		} else if name == critDmgStat {
			hasCritDmg = true
			critDmgVal += val
			if validSet[critDmgStat] && !checkedFirstCrit {
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
		remaining := param.MaxStatSlots - totalCount
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
	remainingSlots := param.MaxStatSlots - totalCount
	if (validCount + remainingSlots) < param.ValidStatsMin {
		log.Debug().Int("max_possible", validCount+remainingSlots).Msg("insufficient valid stats")
		return false, "insufficient_valid_stats"
	}

	return true, "keep"
}

func normalizeEchoStats(stats []EchoStat, param echoEnhanceParam) []EchoStat {
	for i := range stats {
		stats[i].Name = canonicalizeEchoEnhanceStatName(stats[i].Name, param)
	}
	return stats
}

func canonicalizeEchoEnhanceStatName(name string, param echoEnhanceParam) string {
	base := normalizeEchoEnhanceText(name, param.TextFix)
	if base == "" {
		return ""
	}
	for canonical, aliases := range param.StatAliases {
		if textMatchesEchoEnhanceAlias(base, canonical, param.TextFix) {
			return canonical
		}
		for _, alias := range aliases {
			if textMatchesEchoEnhanceAlias(base, alias, param.TextFix) {
				return canonical
			}
		}
	}

	fallback := normalizeStatName(base)
	if fallback == "" {
		return base
	}
	return fallback
}

func textMatchesEchoEnhanceAlias(text string, alias string, textFix map[string]string) bool {
	alias = normalizeEchoEnhanceText(alias, textFix)
	return alias != "" && strings.Contains(text, alias)
}

func normalizeEchoEnhanceText(text string, textFix map[string]string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, " ", ""))
	if len(textFix) == 0 {
		return text
	}
	for from, to := range textFix {
		text = strings.ReplaceAll(text, from, to)
	}
	return text
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

type EchoChangeRecordSuccessAction struct{}

var _ maa.CustomActionRunner = &EchoChangeRecordSuccessAction{}

func (a *EchoChangeRecordSuccessAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	filename := fmt.Sprintf("%d.png", time.Now().UnixMilli())
	(&EchoEnhanceAction{}).captureEchoSnapshot(ctx, filepath.Join("change_success", filename))
	log.Info().
		Str("component", "EchoChange").
		Str("snapshot", filename).
		Msg("echo main stat changed")
	return true
}

// ---------------------------------------------------------------------------
// FiveToOnePrepareStepAction prepares filters for one set/step batch-fusion
// lifecycle. The actual fuse/confirm/result loop is owned by Pipeline JSON.
// ---------------------------------------------------------------------------

type FiveToOnePrepareStepAction struct{}
type FiveToOneAdvanceStepAction struct{}
type FiveToOneRecordShortageAction struct{}

var _ maa.CustomActionRunner = &FiveToOnePrepareStepAction{}
var _ maa.CustomActionRunner = &FiveToOneAdvanceStepAction{}
var _ maa.CustomActionRunner = &FiveToOneRecordShortageAction{}

var fiveToOneState struct {
	setsProcessed int
	shortages     int
	initialized   bool
	param         fiveToOneParam
	setIndex      int
	step          int
	currentSet    string
	currentStep   int
}

type fiveToOneParam struct {
	Keep             map[string][]string `json:"keep"`
	KeepConfig       string              `json:"keep_config"`
	ContinueOnOCRErr bool                `json:"continue_on_ocr_error"`
	StepOneChoiceCount int               `json:"step_one_choice_count"`
	Sets             []string            `json:"sets"`
	MainStats        []string            `json:"main_stats"`
	FlatMainStats    []string            `json:"flat_main_stats"`
	StepTwoStats     []string            `json:"step_two_stats"`
	TextFix          map[string]string   `json:"text_fix"`
	FilterOCRNode    string              `json:"filter_ocr_node"`
	StatOCRNode      string              `json:"stat_ocr_node"`
	Layout           fiveToOneLayout     `json:"layout"`
	Timing           fiveToOneTiming     `json:"timing"`
	ParseOK          bool                `json:"-"`
}

type fiveToOneLayout struct {
	SelectRarity        [2]int32 `json:"select_rarity"`
	SelectCostStep1     [2]int32 `json:"select_cost_step1"`
	SelectMainStatLeft  [2]int32 `json:"select_main_stat_left"`
	SelectMainStatMid   [2]int32 `json:"select_main_stat_mid"`
	SelectMainStatRight [2]int32 `json:"select_main_stat_right"`
	OpenSetFilter       [2]int32 `json:"open_set_filter"`
	OpenStatFilter      [2]int32 `json:"open_stat_filter"`
	ConfirmFilter       [2]int32 `json:"confirm_filter"`
}

type fiveToOneTiming struct {
	SelectRarityDelayMs   int `json:"select_rarity_delay_ms"`
	SelectCostDelayMs     int `json:"select_cost_delay_ms"`
	SelectMainStatDelayMs int `json:"select_main_stat_delay_ms"`
	OpenSetFilterDelayMs  int `json:"open_set_filter_delay_ms"`
	AfterSetFilterDelayMs int `json:"after_set_filter_delay_ms"`
	OpenStatFilterDelayMs int `json:"open_stat_filter_delay_ms"`
	SelectStatDelayMs     int `json:"select_stat_delay_ms"`
	ConfirmFilterDelayMs  int `json:"confirm_filter_delay_ms"`
}

func (a *FiveToOnePrepareStepAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	if !fiveToOneState.initialized {
		fiveToOneState.param = parseFiveToOneParam(arg)
		if !fiveToOneState.param.ParseOK {
			fiveToOneState.initialized = false
			return false
		}
		fiveToOneState.initialized = true
		fiveToOneState.setIndex = 0
		fiveToOneState.step = 1
		fiveToOneState.currentSet = ""
		fiveToOneState.currentStep = 0
		fiveToOneState.setsProcessed = 0
		fiveToOneState.shortages = 0
	}

	param := fiveToOneState.param
	for fiveToOneState.setIndex < len(param.Sets) {
		if ctx.GetTasker().Stopping() {
			return false
		}

		setName := param.Sets[fiveToOneState.setIndex]
		step := fiveToOneState.step
		if !a.stepNeedsProcessing(setName, step, param) {
			advanceFiveToOneStep()
			continue
		}
		if !a.prepareFilters(ctx, setName, step, param) {
			if param.ContinueOnOCRErr {
				advanceFiveToOneStep()
				continue
			}
			return false
		}

		fiveToOneState.currentSet = setName
		fiveToOneState.currentStep = step
		return true
	}

	logFiveToOneSummary()
	fiveToOneState.initialized = false
	return false
}

func parseFiveToOneParam(arg *maa.CustomActionArg) fiveToOneParam {
	param := fiveToOneParam{ParseOK: true}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Error().Err(err).Str("component", "FiveToOnePrepareStep").Msg("failed to parse param")
			param.ParseOK = false
			return param
		}
	}
	if param.Keep == nil {
		param.Keep = map[string][]string{}
	}
	if param.KeepConfig != "" {
		var keep map[string][]string
		if err := sonic.Unmarshal([]byte(param.KeepConfig), &keep); err != nil {
			log.Error().Err(err).Str("component", "FiveToOnePrepareStep").Msg("failed to parse keep_config")
			param.ParseOK = false
			return param
		} else {
			param.Keep = keep
		}
	}
	param.Sets = normalizeFiveToOneStringList(param.Sets, defaultFiveToOneSets)
	param.MainStats = normalizeFiveToOneStringList(param.MainStats, defaultFiveToOneMainStats)
	param.FlatMainStats = normalizeFiveToOneStringList(param.FlatMainStats, defaultFiveToOneFlatMainStats)
	param.StepTwoStats = normalizeFiveToOneStringList(param.StepTwoStats, defaultFiveToOneStepTwoStats)
	param.TextFix = normalizeFiveToOneTextFix(param.TextFix)
	if param.FilterOCRNode == "" {
		param.FilterOCRNode = "FiveToOne_ClickOCR"
	}
	if param.StatOCRNode == "" {
		param.StatOCRNode = "FiveToOne_StatOCR"
	}
	if param.StepOneChoiceCount <= 0 {
		param.StepOneChoiceCount = 16
	}
	param.Layout = normalizeFiveToOneLayout(param.Layout)
	param.Timing = normalizeFiveToOneTiming(param.Timing)
	return param
}

func normalizeFiveToOneStringList(values []string, fallback []string) []string {
	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}

func normalizeFiveToOneTextFix(values map[string]string) map[string]string {
	out := make(map[string]string, len(defaultFiveToOneTextFix))
	for from, to := range defaultFiveToOneTextFix {
		out[from] = to
	}
	for from, to := range values {
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" || to == "" {
			continue
		}
		out[from] = to
	}
	return out
}

func normalizeFiveToOneLayout(layout fiveToOneLayout) fiveToOneLayout {
	if layout.SelectRarity == [2]int32{} {
		layout.SelectRarity = [2]int32{51, 655}
	}
	if layout.SelectCostStep1 == [2]int32{} {
		layout.SelectCostStep1 = [2]int32{794, 590}
	}
	if layout.SelectMainStatLeft == [2]int32{} {
		layout.SelectMainStatLeft = [2]int32{256, 511}
	}
	if layout.SelectMainStatMid == [2]int32{} {
		layout.SelectMainStatMid = [2]int32{602, 511}
	}
	if layout.SelectMainStatRight == [2]int32{} {
		layout.SelectMainStatRight = [2]int32{909, 511}
	}
	if layout.OpenSetFilter == [2]int32{} {
		layout.OpenSetFilter = [2]int32{1146, 396}
	}
	if layout.OpenStatFilter == [2]int32{} {
		layout.OpenStatFilter = [2]int32{1146, 533}
	}
	if layout.ConfirmFilter == [2]int32{} {
		layout.ConfirmFilter = [2]int32{1037, 605}
	}
	return layout
}

func normalizeFiveToOneTiming(timing fiveToOneTiming) fiveToOneTiming {
	if timing.SelectRarityDelayMs <= 0 {
		timing.SelectRarityDelayMs = 300
	}
	if timing.SelectCostDelayMs <= 0 {
		timing.SelectCostDelayMs = 80
	}
	if timing.SelectMainStatDelayMs <= 0 {
		timing.SelectMainStatDelayMs = 80
	}
	if timing.OpenSetFilterDelayMs <= 0 {
		timing.OpenSetFilterDelayMs = 500
	}
	if timing.AfterSetFilterDelayMs <= 0 {
		timing.AfterSetFilterDelayMs = 250
	}
	if timing.OpenStatFilterDelayMs <= 0 {
		timing.OpenStatFilterDelayMs = 500
	}
	if timing.SelectStatDelayMs <= 0 {
		timing.SelectStatDelayMs = 60
	}
	if timing.ConfirmFilterDelayMs <= 0 {
		timing.ConfirmFilterDelayMs = 500
	}
	return timing
}

func (a *FiveToOnePrepareStepAction) stepNeedsProcessing(setName string, step int, param fiveToOneParam) bool {
	keeps := param.Keep[setName]
	if len(keeps) >= len(param.MainStats) {
		log.Info().Str("component", "FiveToOnePrepareStep").Str("set", setName).Msg("all main stats kept, skipping")
		return false
	}
	return step != 2 || containsAnyValue(keeps, param.StepTwoStats)
}

func logFiveToOneSummary() {
	fiveToOneState.setsProcessed = 0
	if fiveToOneState.setIndex > 0 {
		fiveToOneState.setsProcessed = fiveToOneState.setIndex
	}
	log.Info().
		Str("component", "FiveToOneSummary").
		Int("sets_processed", fiveToOneState.setsProcessed).
		Int("shortages", fiveToOneState.shortages).
		Msg("all configured sets processed")
}

var defaultFiveToOneSets = []string{
	"凝夜白霜", "熔山裂谷", "彻空冥雷", "啸谷长风", "浮星祛暗", "沉日劫明", "隐世回光", "轻云出月", "不绝余音",
	"凌冽决断之心", "此间永驻之光", "幽夜隐匿之帷", "高天共奏之曲", "无惧浪涛之勇", "流云逝尽之空", "愿戴荣光之旅", "奔狼燎原之焰",
}

var defaultFiveToOneMainStats = []string{
	"攻击力百分比", "生命值百分比", "防御力百分比", "暴击率", "暴击伤害", "共鸣效率",
	"冷凝伤害加成", "热熔伤害加成", "导电伤害加成", "气动伤害加成", "衍射伤害加成", "湮灭伤害加成", "治疗效果加成",
}

var defaultFiveToOneFlatMainStats = []string{"主属性生命值", "主属性攻击力", "主属性防御力"}

var defaultFiveToOneStepTwoStats = []string{"攻击力百分比"}

var defaultFiveToOneTextFix = map[string]string{
	"凝夜自霜":      "凝夜白霜",
	"主属性灭伤害加成":  "主属性湮灭伤害加成",
	"灭伤害加成":     "主属性湮灭伤害加成",
	"主属性行射伤害加成": "主属性衍射伤害加成",
}

func (a *FiveToOnePrepareStepAction) prepareFilters(ctx *maa.Context, setName string, step int, param fiveToOneParam) bool {
	keeps := param.Keep[setName]
	layout := param.Layout
	timing := param.Timing
	log.Info().
		Str("component", "FiveToOnePrepareStep").
		Str("set", setName).
		Int("step", step).
		Strs("keeps", keeps).
		Msg("processing set")

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(layout.SelectRarity[0], layout.SelectRarity[1]).Wait()
	time.Sleep(time.Duration(timing.SelectRarityDelayMs) * time.Millisecond)
	if step == 1 {
		ctrl.PostClick(layout.SelectCostStep1[0], layout.SelectCostStep1[1]).Wait()
		time.Sleep(time.Duration(timing.SelectCostDelayMs) * time.Millisecond)
	}
	ctrl.PostClick(layout.SelectMainStatLeft[0], layout.SelectMainStatLeft[1]).Wait()
	time.Sleep(time.Duration(timing.SelectMainStatDelayMs) * time.Millisecond)
	ctrl.PostClick(layout.SelectMainStatMid[0], layout.SelectMainStatMid[1]).Wait()
	time.Sleep(time.Duration(timing.SelectMainStatDelayMs) * time.Millisecond)
	if step == 1 {
		ctrl.PostClick(layout.SelectMainStatRight[0], layout.SelectMainStatRight[1]).Wait()
		time.Sleep(time.Duration(timing.SelectMainStatDelayMs) * time.Millisecond)
	}

	if step == 1 {
		ctrl.PostClick(layout.OpenSetFilter[0], layout.OpenSetFilter[1]).Wait()
		time.Sleep(time.Duration(timing.OpenSetFilterDelayMs) * time.Millisecond)
		if !a.clickOCR(ctx, setName) {
			log.Warn().Str("component", "FiveToOnePrepareStep").Str("set", setName).Msg("set filter not found")
			return false
		}
		time.Sleep(time.Duration(timing.AfterSetFilterDelayMs) * time.Millisecond)
	}

	ctrl.PostClick(layout.OpenStatFilter[0], layout.OpenStatFilter[1]).Wait()
	time.Sleep(time.Duration(timing.OpenStatFilterDelayMs) * time.Millisecond)
	choices := a.ocrStatChoices(ctx)
	if len(choices) == 0 {
		log.Warn().Str("component", "FiveToOnePrepareStep").Str("set", setName).Int("step", step).Msg("stat OCR failed")
		return false
	}
	if step == 1 && len(choices) != param.StepOneChoiceCount {
		log.Warn().
			Str("component", "FiveToOnePrepareStep").
			Str("set", setName).
			Int("step", step).
			Int("expected_choices", param.StepOneChoiceCount).
			Int("actual_choices", len(choices)).
			Msg("stat OCR incomplete, aborting to avoid wrong fusion")
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
		} else if !containsAny(param.StepTwoStats, normalizeFiveToOneText(choice.Text)) {
			continue
		}
		ctrl.PostClick(int32(choice.Box[0]+choice.Box[2]/2), int32(choice.Box[1]+choice.Box[3]/2)).Wait()
		clicked++
		time.Sleep(time.Duration(timing.SelectStatDelayMs) * time.Millisecond)
	}

	log.Info().
		Str("component", "FiveToOnePrepareStep").
		Str("set", setName).
		Int("step", step).
		Int("clicked_filters", clicked).
		Msg("selected merge filters")

	ctrl.PostClick(layout.ConfirmFilter[0], layout.ConfirmFilter[1]).Wait()
	time.Sleep(time.Duration(timing.ConfirmFilterDelayMs) * time.Millisecond)
	return true
}

type fiveToOneChoice struct {
	Text string
	Box  maa.Rect
}

const fiveToOneRoundCounterKey = "five_to_one_round"

func (a *FiveToOneAdvanceStepAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().
		Str("component", "FiveToOneAdvanceStep").
		Str("set", fiveToOneState.currentSet).
		Int("step", fiveToOneState.currentStep).
		Int("round", counterpkg.Peek(fiveToOneRoundCounterKey)).
		Msg("advancing to next step after round limit")
	advanceFiveToOneStep()
	return true
}

func (a *FiveToOneRecordShortageAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	fiveToOneState.shortages++
	log.Info().
		Str("component", "FiveToOneRecordShortage").
		Str("set", fiveToOneState.currentSet).
		Int("step", fiveToOneState.currentStep).
		Int("round", counterpkg.Peek(fiveToOneRoundCounterKey)).
		Int("shortages", fiveToOneState.shortages).
		Msg("not enough echoes")
	advanceFiveToOneStep()
	return true
}

func advanceFiveToOneStep() {
	fiveToOneState.currentSet = ""
	fiveToOneState.currentStep = 0
	if fiveToOneState.step == 1 {
		fiveToOneState.step = 2
		return
	}
	fiveToOneState.step = 1
	fiveToOneState.setIndex++
	fiveToOneState.setsProcessed = fiveToOneState.setIndex
}

func (a *FiveToOnePrepareStepAction) clickOCR(ctx *maa.Context, text string) bool {
	detail, err := ctx.RunRecognition(fiveToOneState.param.FilterOCRNode, nil)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return false
	}
	box, ok := findFiveToOneOCRBox(detail, text)
	if !ok {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(box[0]+box[2]/2),
		int32(box[1]+box[3]/2),
	).Wait()
	return true
}

func (a *FiveToOnePrepareStepAction) ocrStatChoices(ctx *maa.Context) []fiveToOneChoice {
	detail, err := ctx.RunRecognition(fiveToOneState.param.StatOCRNode, nil)
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

func findFiveToOneOCRBox(detail *maa.RecognitionDetail, text string) (maa.Rect, bool) {
	target := normalizeFiveToOneText(text)
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		if strings.Contains(normalizeFiveToOneText(ocr.Text), target) {
			return ocr.Box, true
		}
	}
	return maa.Rect{}, false
}

func normalizeFiveToOneText(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, " ", ""))
	textFix := fiveToOneState.param.TextFix
	if len(textFix) == 0 {
		textFix = defaultFiveToOneTextFix
	}
	for from, to := range textFix {
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

func containsAnyValue(values []string, targets []string) bool {
	for _, value := range values {
		if containsAny(targets, value) {
			return true
		}
	}
	return false
}

func isFiveToOneMainStat(text string) bool {
	text = normalizeFiveToOneText(text)
	mainStats := fiveToOneState.param.MainStats
	if len(mainStats) == 0 {
		mainStats = defaultFiveToOneMainStats
	}
	for _, stat := range mainStats {
		if strings.Contains(text, "主属性"+stat) || strings.Contains(text, stat) {
			return true
		}
	}
	return false
}

func isFiveToOneFlatStat(text string) bool {
	text = normalizeFiveToOneText(text)
	flatMainStats := fiveToOneState.param.FlatMainStats
	if len(flatMainStats) == 0 {
		flatMainStats = defaultFiveToOneFlatMainStats
	}
	for _, stat := range flatMainStats {
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
