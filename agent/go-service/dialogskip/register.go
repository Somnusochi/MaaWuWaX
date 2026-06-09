package dialogskip

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("IsMailEnabled", &IsMailEnabledRecognition{})
	log.Info().Str("component", "dialogskip").Msg("registered IsMailEnabled")
}
