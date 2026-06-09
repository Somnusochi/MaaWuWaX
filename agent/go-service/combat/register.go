package combat

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("CombatState", &CombatStateRecognition{})
	maa.AgentServerRegisterCustomAction("CombatMain", &CombatMainAction{})
	maa.AgentServerRegisterCustomRecognition("CharacterDetect", &CharacterDetectRecognition{})
	log.Info().Str("component", "combat").Msg("registered CombatState, CombatMain, CharacterDetect")
}
