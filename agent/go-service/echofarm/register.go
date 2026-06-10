package echofarm

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("EchoOrbDetect", &EchoOrbDetect{})
	maa.AgentServerRegisterCustomAction("EchoFarmResetLoop", &EchoFarmResetLoopAction{})
	maa.AgentServerRegisterCustomAction("EchoFarmNextRound", &EchoFarmNextRoundAction{})
	maa.AgentServerRegisterCustomAction("EchoFarmCollect", &EchoFarmCollectAction{})
	log.Info().Str("component", "echofarm").Msg("registered echo-farm components")
}
