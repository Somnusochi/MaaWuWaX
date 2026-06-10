// Package schedule provides a time-based CustomRecognition that gates
// pipeline execution based on configured time windows.
// Implements ok-ww's support_schedule_task pattern.
package schedule

import (
	"encoding/json"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type scheduleParam struct {
	// AfterHour/Minute: task allowed after this time of day (local).
	AfterHour   int `json:"after_hour"`
	AfterMinute int `json:"after_minute"`
	// BeforeHour/Minute: task must finish before this time.
	BeforeHour   int `json:"before_hour"`
	BeforeMinute int `json:"before_minute"`
	// Weekdays: empty = all days, otherwise e.g. [1,3,5] = Mon/Wed/Fri.
	Weekdays []int `json:"weekdays"`
}

// ScheduleRecognition checks if the current local time falls within
// the configured time window. Returns hit if so, else no-hit.
type ScheduleRecognition struct{}

var _ maa.CustomRecognitionRunner = &ScheduleRecognition{}

func (r *ScheduleRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := scheduleParam{}
	if arg.CustomRecognitionParam != "" {
		if err := json.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "Schedule").Msg("failed to parse param, allowing")
			return &maa.CustomRecognitionResult{Box: maa.Rect{0, 0, 1, 1}}, true
		}
	}

	now := time.Now()

	// Check weekday filter.
	if len(param.Weekdays) > 0 {
		today := int(now.Weekday())
		found := false
		for _, d := range param.Weekdays {
			if d == today {
				found = true
				break
			}
		}
		if !found {
			log.Debug().Int("weekday", today).Strs("allowed", intsToStrs(param.Weekdays)).Str("component", "Schedule").Msg("weekday not allowed")
			return nil, false
		}
	}

	// Check time window.
	todayMinutes := now.Hour()*60 + now.Minute()
	afterMinutes := param.AfterHour*60 + param.AfterMinute
	beforeMinutes := param.BeforeHour*60 + param.BeforeMinute

	if afterMinutes > 0 && todayMinutes < afterMinutes {
		log.Debug().Int("now_min", todayMinutes).Int("after_min", afterMinutes).Str("component", "Schedule").Msg("before allowed window")
		return nil, false
	}
	if beforeMinutes > 0 && todayMinutes >= beforeMinutes {
		log.Debug().Int("now_min", todayMinutes).Int("before_min", beforeMinutes).Str("component", "Schedule").Msg("after allowed window")
		return nil, false
	}

	log.Debug().Int("now_min", todayMinutes).Str("component", "Schedule").Msg("within schedule window")
	return &maa.CustomRecognitionResult{Box: maa.Rect{0, 0, 1, 1}}, true
}

func intsToStrs(ints []int) []string {
	s := make([]string, len(ints))
	for i, v := range ints {
		s[i] = string(rune('0' + v))
	}
	return s
}
