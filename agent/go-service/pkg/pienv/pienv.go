// Package pienv parses PI V2 environment variables injected by the MXU client.
package pienv

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

const (
	EnvInterfaceVersion   = "PI_INTERFACE_VERSION"
	EnvClientName         = "PI_CLIENT_NAME"
	EnvClientVersion      = "PI_CLIENT_VERSION"
	EnvClientLanguage     = "PI_CLIENT_LANGUAGE"
	EnvClientMaaFWVersion = "PI_CLIENT_MAAFW_VERSION"
	EnvVersion            = "PI_VERSION"
	EnvController         = "PI_CONTROLLER"
	EnvResource           = "PI_RESOURCE"
)

// MacOSConfig holds macOS controller-specific fields.
type MacOSConfig struct {
	TitleRegex string `json:"title_regex,omitempty"`
	Screencap  string `json:"screencap,omitempty"`
	Input      string `json:"input,omitempty"`
}

// Controller is the parsed PI_CONTROLLER JSON.
type Controller struct {
	Name    string       `json:"name"`
	Label   string       `json:"label,omitempty"`
	Type    string       `json:"type"`
	MacOS   *MacOSConfig `json:"macos,omitempty"`
}

// Resource is the parsed PI_RESOURCE JSON.
type Resource struct {
	Name string   `json:"name"`
	Path []string `json:"path"`
}

// Env holds all parsed PI_* environment variables.
type Env struct {
	InterfaceVersion   string
	ClientName         string
	ClientVersion      string
	ClientLanguage     string
	ClientMaaFWVersion string
	Version            string
	Controller         *Controller
	Resource           *Resource
}

var (
	global *Env
	once   sync.Once
)

func doInit() {
	env := &Env{
		InterfaceVersion:   os.Getenv(EnvInterfaceVersion),
		ClientName:         os.Getenv(EnvClientName),
		ClientVersion:      os.Getenv(EnvClientVersion),
		ClientLanguage:     os.Getenv(EnvClientLanguage),
		ClientMaaFWVersion: os.Getenv(EnvClientMaaFWVersion),
		Version:            os.Getenv(EnvVersion),
	}

	if raw := os.Getenv(EnvController); raw != "" {
		var ctrl Controller
		if err := json.Unmarshal([]byte(raw), &ctrl); err != nil {
			log.Warn().Err(err).Str("env_key", EnvController).Msg("failed to parse env")
		} else {
			env.Controller = &ctrl
		}
	}

	if raw := os.Getenv(EnvResource); raw != "" {
		var res Resource
		if err := json.Unmarshal([]byte(raw), &res); err != nil {
			log.Warn().Err(err).Str("env_key", EnvResource).Msg("failed to parse env")
		} else {
			env.Resource = &res
		}
	}

	global = env

	log.Info().
		Str("component", "pienv").
		Str("interface_version", env.InterfaceVersion).
		Str("client_name", env.ClientName).
		Str("client_version", env.ClientVersion).
		Str("client_language", env.ClientLanguage).
		Bool("controller_ok", env.Controller != nil).
		Bool("resource_ok", env.Resource != nil).
		Msg("PI environment initialized")
}

// Init reads and parses all PI_* environment variables.
func Init() {
	once.Do(doInit)
}

// Get returns the global Env.
func Get() *Env {
	once.Do(doInit)
	return global
}

// ClientLanguage returns the PI_CLIENT_LANGUAGE value.
func ClientLanguage() string { return Get().ClientLanguage }

// ControllerType returns the controller type (e.g. "MacOS"), or empty.
func ControllerType() string {
	if c := Get().Controller; c != nil {
		return c.Type
	}
	return ""
}
