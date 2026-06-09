package rogue

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("RogueMain", &RogueMainAction{})
	maa.AgentServerRegisterCustomAction("RogueBuffSelect", &RogueBuffSelectAction{})
	log.Info().Str("component", "rogue").Msg("registered RogueMain, RogueBuffSelect")
}
