package main

import (
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/combat"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/falseaction"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/pipelineoverride"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/schedule"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/common/subtask"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/daily"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/diagnosis"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/dialogskip"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/domain"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/echofarm"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/echonhance"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/farmmap"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/login"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/navigation"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/nightmare"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pickup"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/resource"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/rogue"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/stamina"
	"github.com/rs/zerolog/log"
)

func registerAll() {
	// ── Tier 1: Resource Sink ─────────────────────────────────────────
	resource.EnsureResourcePathSink()

	// ── Tier 2: Common Primitives ─────────────────────────────────────
	falseaction.Register()
	pipelineoverride.Register()
	schedule.Register()
	subtask.Register()

	// ── Tier 3: Business Modules ──────────────────────────────────────
	combat.Register()
	diagnosis.Register()
	pickup.Register()
	login.Register()
	dialogskip.Register()
	domain.Register()
	daily.Register()
	stamina.Register()
	navigation.Register()
	nightmare.Register()
	rogue.Register()
	echonhance.Register()
	echofarm.Register()
	farmmap.Register()

	log.Info().Msg("All custom components registered successfully")
}
