// Package echonhance implements echo enhancement and stat change Custom Actions for Wuthering Waves.
package echonhance

import (
	"fmt"
	"strconv"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
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

type EchoEnhanceAction struct{}

var _ maa.CustomActionRunner = &EchoEnhanceAction{}

type echoEnhanceParam struct {
	NeedDoubleCrit      bool     `json:"need_double_crit"`
	DoubleCritMin       float64  `json:"double_crit_min"`
	FirstCritMin        float64  `json:"first_crit_min"`
	ValidStatsMin       int      `json:"valid_stats_min"`
	FirstMustBeValid    bool     `json:"first_must_be_valid"`
	AllValidBeforeCrit  bool     `json:"all_valid_before_crit"`
	ValidStats          []string `json:"valid_stats"`
}

func defaultEchoEnhanceParam() echoEnhanceParam {
	return echoEnhanceParam{
		NeedDoubleCrit:     true,
		DoubleCritMin:      13.8,
		FirstCritMin:       6.9,
		ValidStatsMin:      3,
		FirstMustBeValid:   true,
		AllValidBeforeCrit: true,
		ValidStats: []string{
			"暴击", "暴击伤害", "攻击百分比",
		},
	}
}

func (a *EchoEnhanceAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := defaultEchoEnhanceParam()
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoEnhance").Msg("failed to parse param")
		}
	}

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
		log.Warn().Str("component", "EchoEnhance").Msg("failed to read echo stats, discarding")
		return false
	}

	stats := parseEchoStats(detail.DetailJson)
	keep := evaluateEcho(stats, param)

	if keep {
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Msg("echo KEEP — locking")
	} else {
		log.Info().
			Str("component", "EchoEnhance").
			Int("stats", len(stats)).
			Msg("echo DISCARD")
	}

	return keep
}

func evaluateEcho(stats []EchoStat, param echoEnhanceParam) bool {
	if len(stats) == 0 {
		return false
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
				return false
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
					return false
				}
			}
		} else if name == "暴击伤害" {
			hasCritDmg = true
			critDmgVal += val
			if validSet["暴击伤害"] && !checkedFirstCrit {
				checkedFirstCrit = true
				if val/2 < param.FirstCritMin {
					log.Debug().Float64("val", val).Msg("first crit dmg too low")
					return false
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
			return false
		}
	}

	// Double crit total check.
	if hasCritRate && hasCritDmg {
		total := critRateVal + critDmgVal/2
		if total < param.DoubleCritMin {
			log.Debug().Float64("total", total).Msg("double crit total too low")
			return false
		}
	}

	// First must be valid check.
	if totalCount == 1 && param.FirstMustBeValid && invalidCount == 1 {
		log.Debug().Msg("first stat is invalid")
		return false
	}

	// Valid stats count check.
	validCount := totalCount - invalidCount
	remainingSlots := 5 - totalCount
	if (validCount + remainingSlots) < param.ValidStatsMin {
		log.Debug().Int("max_possible", validCount+remainingSlots).Msg("insufficient valid stats")
		return false
	}

	return true
}

// ---------------------------------------------------------------------------
// EchoChangeSelectAction — selects the target main stat for echo change.
// ---------------------------------------------------------------------------

type EchoChangeSelectAction struct{}

var _ maa.CustomActionRunner = &EchoChangeSelectAction{}

type echoChangeParam struct {
	TargetStat string `json:"target_stat"`
}

func (a *EchoChangeSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoChangeParam{TargetStat: "攻击"}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoChangeSelect").Msg("failed to parse param")
		}
	}

	ctrl := ctx.GetTasker().GetController()

	// OCR to find the target stat in the dropdown.
	detail, err := ctx.RunRecognition(
		"__EchoChange_FindStat",
		nil,
		fmt.Sprintf(`{
			"__EchoChange_FindStat": {
				"recognition": "OCR",
				"expected": %q,
				"roi": [100, 300, 800, 400]
			}
		}`, param.TargetStat),
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "EchoChangeSelect").Str("target", param.TargetStat).Msg("target stat not found")
		return false
	}

	box := detail.Box
	ctrl.PostClick(
		int32(box[0]+box[2]/2),
		int32(box[1]+box[3]/2),
	).Wait()

	log.Info().
		Str("component", "EchoChangeSelect").
		Str("target", param.TargetStat).
		Msg("selected target stat")

	return true
}
