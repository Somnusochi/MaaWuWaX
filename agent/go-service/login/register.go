package login

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("LoginScreenDetect", &LoginScreenDetect{})
	log.Info().Str("component", "login").Msg("registered LoginScreenDetect")
}
