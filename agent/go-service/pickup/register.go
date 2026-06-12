package pickup

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("CheckWhitePct", &CheckWhitePctRecognition{})
	log.Info().Str("component", "pickup").Msg("registered CheckWhitePct")
}
