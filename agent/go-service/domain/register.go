package domain

import (
	"github.com/rs/zerolog/log"
)

func Register() {
	log.Info().Str("component", "domain").Msg("no custom domain components to register")
}
