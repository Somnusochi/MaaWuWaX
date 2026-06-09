package navigation

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("MinimapNavigate", &MinimapNavigateRecognition{})
	maa.AgentServerRegisterCustomAction("DirectionWalk", &DirectionWalkAction{})
	maa.AgentServerRegisterCustomAction("TeleportBoss", &TeleportBossAction{})
	log.Info().Str("component", "navigation").Msg("registered MinimapNavigate, DirectionWalk, TeleportBoss")
}
