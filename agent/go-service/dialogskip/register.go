package dialogskip

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("SkipDialogAdvanced", &SkipDialogAdvancedAction{})
	maa.AgentServerRegisterCustomAction("SkipDialogMessageClick", &SkipDialogMessageClickAction{})
	log.Info().Str("component", "dialogskip").Msg("registered dialog skip components")
}
