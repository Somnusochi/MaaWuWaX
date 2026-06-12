package nightmare

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("NightmareFindNest", &FindNestRecognition{})
	maa.AgentServerRegisterCustomRecognition("NestScroll", &NestScrollRecognition{})
	maa.AgentServerRegisterCustomAction("NightmareApproachNest", &ApproachNestAction{})
	log.Info().Str("component", "nightmare").Msg("registered nightmare components")
}
