package pickup

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("PickTextFilter", &PickTextFilterRecognition{})
	maa.AgentServerRegisterCustomAction("PickEnhanced", &PickEnhancedAction{})
	log.Info().Str("component", "pickup").Msg("registered PickTextFilter, PickEnhanced")
}
