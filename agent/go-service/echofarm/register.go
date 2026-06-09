package echofarm

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("EchoOrbDetect", &EchoOrbDetect{})
	maa.AgentServerRegisterCustomAction("FiveToOneMerge", &FiveToOneMergeAction{})
	log.Info().Str("component", "echofarm").Msg("registered EchoOrbDetect, FiveToOneMerge")
}
