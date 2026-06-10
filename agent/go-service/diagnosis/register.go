package diagnosis

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("DiagnosisSnapshot", &SnapshotAction{})
	log.Info().Str("component", "diagnosis").Msg("registered diagnosis components")
}
