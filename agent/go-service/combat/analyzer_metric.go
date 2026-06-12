package combat

import (
	"image"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type CombatAnalyzerMetricRecognition struct{}

var _ maa.CustomRecognitionRunner = &CombatAnalyzerMetricRecognition{}

type analyzerMetricParam struct {
	Mode           string   `json:"mode"`
	ROI            []int    `json:"roi"`
	Lower          []uint32 `json:"lower"`
	Upper          []uint32 `json:"upper"`
	MinValue       *float64 `json:"min_value"`
	MaxValue       *float64 `json:"max_value"`
	WhiteThreshold float64  `json:"white_threshold"`
	MeanMin        float64  `json:"mean_min"`
	StdMax         float64  `json:"std_max"`
	StepWidth      float64  `json:"step_width"`
	Segments       int      `json:"segments"`
	MinFreq        int      `json:"min_freq"`
	MaxFreq        int      `json:"max_freq"`
	MinAmp         float64  `json:"min_amp"`
	ScanFromRight  bool     `json:"scan_from_right"`
}

type analyzerMetricDetail struct {
	Value float64 `json:"value"`
	Int   int     `json:"int"`
	Bool  bool    `json:"bool"`
}

func (r *CombatAnalyzerMetricRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := analyzerMetricParam{}
	if arg == nil || arg.CustomRecognitionParam == "" {
		return nil, false
	}
	if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
		log.Warn().Err(err).Str("component", "CombatAnalyzerMetric").Msg("failed to parse param")
		return nil, false
	}
	img := image.Image(nil)
	if arg != nil {
		img = arg.Img
	}
	roi, ok := metricROI(param.ROI)
	if !ok {
		return nil, false
	}

	detail := analyzerMetricDetail{}
	switch param.Mode {
	case "near_white_pct":
		detail.Value = sampleNearWhitePct(img, roi)
		detail.Bool = withinMetricRange(detail.Value, param.MinValue, param.MaxValue)
	case "color_pct":
		lower, upper, ok := metricColorBounds(param.Lower, param.Upper)
		if !ok {
			return nil, false
		}
		detail.Value = sampleColorPct(img, roi, lower[0], upper[0], lower[1], upper[1], lower[2], upper[2])
		detail.Bool = withinMetricRange(detail.Value, param.MinValue, param.MaxValue)
	case "stripe_fill_pct":
		lower, upper, ok := metricColorBounds(param.Lower, param.Upper)
		if !ok {
			return nil, false
		}
		detail.Value = sampleStripeFillPct(img, roi, lower[0], upper[0], lower[1], upper[1], lower[2], upper[2])
		detail.Bool = withinMetricRange(detail.Value, param.MinValue, param.MaxValue)
	case "forte_num_fft":
		lower, upper, ok := metricColorBounds(param.Lower, param.Upper)
		if !ok {
			return nil, false
		}
		detail.Int = sampleForteNumByFFT(
			img,
			roi,
			param.StepWidth,
			colorRange{lower[0], upper[0], lower[1], upper[1], lower[2], upper[2]},
			param.Segments,
			param.MinFreq,
			param.MaxFreq,
			param.MinAmp,
			param.ScanFromRight,
		)
		detail.Value = float64(detail.Int)
		detail.Bool = withinMetricRange(detail.Value, param.MinValue, param.MaxValue)
	case "white_contrast_bool":
		detail.Bool = sampleForteFullByWhiteContrast(img, roi, param.WhiteThreshold)
		if detail.Bool {
			detail.Value = 1
		}
	case "gray_window_bool":
		if sampleNearWhitePct(img, roi) > param.WhiteThreshold {
			mean, std := sampleGrayMeanStd(img, roi)
			detail.Bool = mean > param.MeanMin && std < param.StdMax
		}
		if detail.Bool {
			detail.Value = 1
		}
	default:
		log.Warn().Str("component", "CombatAnalyzerMetric").Str("mode", param.Mode).Msg("unsupported metric mode")
		return nil, false
	}

	payload, err := sonic.Marshal(detail)
	if err != nil {
		return nil, false
	}
	return &maa.CustomRecognitionResult{
		Box:    roi,
		Detail: string(payload),
	}, true
}

func metricROI(roi []int) (maa.Rect, bool) {
	if len(roi) != 4 {
		return maa.Rect{}, false
	}
	return maa.Rect{roi[0], roi[1], roi[2], roi[3]}, true
}

func metricColorBounds(lower, upper []uint32) ([3]uint32, [3]uint32, bool) {
	if len(lower) != 3 || len(upper) != 3 {
		return [3]uint32{}, [3]uint32{}, false
	}
	return [3]uint32{lower[0], lower[1], lower[2]}, [3]uint32{upper[0], upper[1], upper[2]}, true
}

func withinMetricRange(value float64, minValue, maxValue *float64) bool {
	if minValue == nil && maxValue == nil {
		return true
	}
	if minValue != nil && value < *minValue {
		return false
	}
	if maxValue != nil && value > *maxValue {
		return false
	}
	return true
}
