package stamina

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("StaminaReader", &StaminaReader{})
	maa.AgentServerRegisterCustomRecognition("StaminaCheck", &StaminaCheckRecognition{})
	log.Info().Str("component", "stamina").Msg("registered StaminaReader, StaminaCheck")
}
