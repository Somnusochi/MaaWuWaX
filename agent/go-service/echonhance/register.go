package echonhance

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomRecognition("EchoStatReader", &EchoStatReader{})
	maa.AgentServerRegisterCustomAction("EchoEnhance", &EchoEnhanceAction{})
	maa.AgentServerRegisterCustomAction("EchoChangeSelect", &EchoChangeSelectAction{})
	log.Info().Str("component", "echonhance").Msg("registered EchoStatReader, EchoEnhance, EchoChangeSelect")
}
