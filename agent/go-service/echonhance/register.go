package echonhance

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("EchoStatReader", &EchoStatReader{})
	maa.AgentServerRegisterCustomAction("EchoEnhance", &EchoEnhanceAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeRecordSuccess", &EchoChangeRecordSuccessAction{})
	maa.AgentServerRegisterCustomAction("FiveToOnePrepareStep", &FiveToOnePrepareStepAction{})
	maa.AgentServerRegisterCustomAction("FiveToOneAdvanceStep", &FiveToOneAdvanceStepAction{})
	maa.AgentServerRegisterCustomAction("FiveToOneRecordShortage", &FiveToOneRecordShortageAction{})
	log.Info().Str("component", "echonhance").Msg("registered echonhance components")
}
