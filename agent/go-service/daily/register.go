package daily

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

func Register() {
	maa.AgentServerRegisterCustomAction("SpendStamina", &SpendStaminaAction{})
	maa.AgentServerRegisterCustomAction("ClaimDailyRewards", &ClaimDailyRewardsAction{})
	maa.AgentServerRegisterCustomAction("ClaimMail", &ClaimMailAction{})
	maa.AgentServerRegisterCustomAction("ClaimBattlePass", &ClaimBattlePassAction{})
	maa.AgentServerRegisterCustomAction("MultiAccountSwitch", &MultiAccountSwitchAction{})
	maa.AgentServerRegisterCustomAction("MultiAccountMarkFailed", &MultiAccountMarkFailedAction{})
	maa.AgentServerRegisterCustomRecognition("DailyProgress", &DailyProgressReader{})
	maa.AgentServerRegisterCustomRecognition("DailyNeedsStamina", &DailyNeedsStaminaRecognition{})
	log.Info().Str("component", "daily").Msg("registered daily components")
}
