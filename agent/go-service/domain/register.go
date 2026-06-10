package domain

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("BookTargetSelect", &BookTargetSelectAction{})
	maa.AgentServerRegisterCustomAction("SimulationSelectMaterial", &SimulationSelectMaterialAction{})
	log.Info().Str("component", "domain").Msg("registered domain components")
}
