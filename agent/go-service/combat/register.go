package combat

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("CombatState", &CombatStateRecognition{})
	maa.AgentServerRegisterCustomRecognition("CombatCheckPending", &CombatCheckPendingRecognition{})
	maa.AgentServerRegisterCustomAction("CombatMain", &CombatMainAction{})
	maa.AgentServerRegisterCustomRecognition("CharacterDetect", &CharacterDetectRecognition{})
	maa.AgentServerRegisterCustomRecognition("CombatAnalyzerMetric", &CombatAnalyzerMetricRecognition{})
	log.Info().Str("component", "combat").Msg("registered CombatState, CombatCheckPending, CombatMain, CharacterDetect, CombatAnalyzerMetric")
}
