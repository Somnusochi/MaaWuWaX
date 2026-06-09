package daily

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("SpendStamina", &SpendStaminaAction{})
	maa.AgentServerRegisterCustomAction("ClaimMail", &ClaimMailAction{})
	maa.AgentServerRegisterCustomAction("ClaimBattlePass", &ClaimBattlePassAction{})
	maa.AgentServerRegisterCustomRecognition("DailyProgress", &DailyProgressReader{})
	log.Info().Str("component", "daily").Msg("registered SpendStamina, ClaimMail, ClaimBattlePass, DailyProgress")
}
