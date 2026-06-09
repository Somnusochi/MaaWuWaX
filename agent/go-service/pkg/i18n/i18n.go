// Package i18n provides simple internationalization support.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/pienv"
	"github.com/rs/zerolog/log"
)

const DefaultLang = "zh_cn"

var (
	currentLang string
	messages    map[string]string
	mu          sync.RWMutex
)

// Init loads the locale file based on PI_CLIENT_LANGUAGE.
func Init() {
	raw := pienv.ClientLanguage()
	lang := strings.ToLower(strings.TrimSpace(raw))
	if lang == "" {
		lang = DefaultLang
	}

	loaded := loadMessages(lang)
	mu.Lock()
	currentLang = lang
	messages = loaded
	mu.Unlock()

	log.Info().
		Str("lang", lang).
		Int("message_count", len(loaded)).
		Msg("i18n initialized")
}

func loadMessages(lang string) map[string]string {
	msgs := make(map[string]string)

	// Try lang-specific first, then default.
	for _, l := range []string{lang, DefaultLang} {
		path := filepath.Join("assets", "locales", "go-service", l+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var loaded map[string]string
		if err := json.Unmarshal(data, &loaded); err != nil {
			log.Warn().Err(err).Str("lang", l).Msg("failed to parse i18n messages")
			continue
		}
		for k, v := range loaded {
			msgs[k] = v
		}
		if len(msgs) > 0 {
			return msgs
		}
	}

	return msgs
}

// T returns a localized string, applying fmt.Sprintf when args are provided.
func T(key string, args ...any) string {
	mu.RLock()
	val, ok := messages[key]
	mu.RUnlock()
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(val, args...)
	}
	return val
}

// Lang returns the current UI language code.
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}
