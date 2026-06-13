package daily

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("DailyNeedNightmare", &DailyNeedNightmareRecognition{})
	maa.AgentServerRegisterCustomRecognition("DailyNeedStamina", &DailyNeedStaminaRecognition{})
	log.Info().Str("component", "daily").Msg("registered daily components")
}
