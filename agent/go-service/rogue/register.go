package rogue

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("RogueBuffSelect", &RogueBuffSelectAction{})
	maa.AgentServerRegisterCustomAction("RogueTreasureReward", &RogueTreasureRewardAction{})
	maa.AgentServerRegisterCustomAction("RogueWalkToTarget", &RogueWalkToTargetAction{})
	maa.AgentServerRegisterCustomAction("RogueWalkGate", &RogueWalkGateAction{})
	log.Info().Str("component", "rogue").Msg("registered rogue components")
}
