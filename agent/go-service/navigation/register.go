package navigation

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("MinimapNavigate", &MinimapNavigateRecognition{})
	maa.AgentServerRegisterCustomRecognition("BossBookTargetVisible", &BossBookTargetVisibleRecognition{})
	maa.AgentServerRegisterCustomRecognition("ForwardApproachResult", &ForwardApproachResultRecognition{})
	maa.AgentServerRegisterCustomAction("DirectionWalk", &DirectionWalkAction{})
	maa.AgentServerRegisterCustomAction("ForwardApproachUntil", &ForwardApproachUntilAction{})
	maa.AgentServerRegisterCustomAction("BossBookRememberSelection", &BossBookRememberSelectionAction{})
	maa.AgentServerRegisterCustomAction("BossBookScrollPage", &BossBookScrollPageAction{})
	maa.AgentServerRegisterCustomAction("BossBookSelectByIndex", &BossBookSelectByIndexAction{})
	maa.AgentServerRegisterCustomAction("BossBookInputSearchText", &BossBookInputSearchTextAction{})
	log.Info().Str("component", "navigation").Msg("registered navigation components")
}
