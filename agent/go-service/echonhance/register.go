package echonhance

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("EchoStatReader", &EchoStatReader{})
	maa.AgentServerRegisterCustomRecognition("EchoChangeGuard", &EchoChangeGuardRecognition{})
	maa.AgentServerRegisterCustomAction("EchoEnhance", &EchoEnhanceAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeReset", &EchoChangeResetAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeSelect", &EchoChangeSelectAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeRecordSuccess", &EchoChangeRecordSuccessAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeSummary", &EchoChangeSummaryAction{})
	maa.AgentServerRegisterCustomAction("FiveToOneMerge", &FiveToOneMergeAction{})
	log.Info().Str("component", "echonhance").Msg("registered echonhance components")
}
