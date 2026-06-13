package echofarm

import (
	"github.com/rs/zerolog/log"
)

func Register() {
	log.Info().Str("component", "echofarm").Msg("registered echo-farm components")
}
