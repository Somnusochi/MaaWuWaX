package counter

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("CounterReset", &CounterResetAction{})
	maa.AgentServerRegisterCustomAction("CounterIncrement", &CounterIncrementAction{})
	maa.AgentServerRegisterCustomRecognition("CounterCheckBelowLimit", &CounterCheckBelowLimitRecognition{})
	maa.AgentServerRegisterCustomRecognition("CounterCheckEquals", &CounterCheckEqualsRecognition{})
	log.Info().Str("component", "counter").Msg("registered counter components")
}
