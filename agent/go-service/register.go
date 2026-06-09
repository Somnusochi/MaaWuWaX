package main

import (
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/combat"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/falseaction"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/pipelineoverride"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/daily"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/dialogskip"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/echofarm"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/echonhance"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/login"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/navigation"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pickup"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/rogue"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/stamina"
	"github.com/rs/zerolog/log"
)

func registerAll() {
	// General Custom
	falseaction.Register()
	pipelineoverride.Register()

	// Business Custom
	combat.Register()
	pickup.Register()
	login.Register()
	dialogskip.Register()
	daily.Register()
	stamina.Register()
	navigation.Register()
	rogue.Register()
	echonhance.Register()
	echofarm.Register()

	log.Info().Msg("All custom components registered successfully")
}
