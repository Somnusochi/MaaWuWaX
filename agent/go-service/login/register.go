package login

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

var multiAccountSwitcher = &MultiAccountSwitchAction{}

func Register() {
	maa.AgentServerRegisterCustomRecognition("LoginScreenDetect", &LoginScreenDetect{})
	maa.AgentServerRegisterCustomAction("MultiAccountSwitch", multiAccountSwitcher)
	maa.AgentServerRegisterCustomAction("MultiAccountMarkFailed", &MultiAccountMarkFailedAction{})
	log.Info().Str("component", "login").Msg("registered login components")
}
