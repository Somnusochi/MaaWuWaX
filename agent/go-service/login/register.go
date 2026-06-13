package login

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

var multiAccountSwitcher = &MultiAccountState{}

func Register() {
	maa.AgentServerRegisterCustomRecognition("MultiAccountCanSwitch", &MultiAccountCanSwitchRecognition{})
	maa.AgentServerRegisterCustomRecognition("MultiAccountDropdownExpanded", &MultiAccountDropdownExpandedRecognition{})
	maa.AgentServerRegisterCustomRecognition("MultiAccountCurrentVisible", &MultiAccountCurrentVisibleRecognition{})
	maa.AgentServerRegisterCustomAction("MultiAccountResetState", &MultiAccountResetStateAction{})
	maa.AgentServerRegisterCustomAction("MultiAccountMarkCurrent", &MultiAccountMarkCurrentAction{})
	maa.AgentServerRegisterCustomAction("MultiAccountSelectNext", &MultiAccountSelectNextAction{})
	maa.AgentServerRegisterCustomRecognition("MultiAccountSelectedMatches", &MultiAccountSelectedMatchesRecognition{})
	maa.AgentServerRegisterCustomAction("MultiAccountMarkFailed", &MultiAccountMarkFailedAction{})
	log.Info().Str("component", "login").Msg("registered login components")
}
