package pipelineoverride

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	runner := &PipelineOverrideAction{}
	maa.AgentServerRegisterCustomAction("PipelineOverride", runner)
	maa.AgentServerRegisterCustomAction("PipelineOverrideAction", runner)
	log.Info().Str("component", "PipelineOverride").Msg("registered")
}
