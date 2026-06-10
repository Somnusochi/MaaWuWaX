package dialogskip

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("IsMailEnabled", &IsMailEnabledRecognition{})
	maa.AgentServerRegisterCustomAction("SkipDialogAdvanced", &SkipDialogAdvancedAction{})
	log.Info().Str("component", "dialogskip").Msg("registered IsMailEnabled, SkipDialogAdvanced")
}
