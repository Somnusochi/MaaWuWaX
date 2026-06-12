package daily

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("DailyProgress", &DailyProgressReader{})
	maa.AgentServerRegisterCustomRecognition("DailyNeedsStamina", &DailyNeedsStaminaRecognition{})
	maa.AgentServerRegisterCustomRecognition("DailyNeedsNightmare", &DailyNeedsNightmareRecognition{})
	log.Info().Str("component", "daily").Msg("registered daily components")
}
