package dialogskip

import "github.com/rs/zerolog/log"

func Register() {
	log.Info().Str("component", "dialogskip").Msg("no custom dialog skip components to register")
}
