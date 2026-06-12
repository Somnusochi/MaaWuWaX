package navigation

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("MinimapNavigate", &MinimapNavigateRecognition{})
	maa.AgentServerRegisterCustomAction("DirectionWalk", &DirectionWalkAction{})
	maa.AgentServerRegisterCustomAction("ClickFastTravel", &ClickFastTravelAction{})
	maa.AgentServerRegisterCustomAction("TeleportBoss", &TeleportBossAction{})
	maa.AgentServerRegisterCustomAction("BossBookPrepareProfile", &BossBookPrepareProfileAction{})
	maa.AgentServerRegisterCustomAction("BossBookTargetSelect", &BossBookTargetSelectAction{})
	log.Info().Str("component", "navigation").Msg("registered navigation components")
}
