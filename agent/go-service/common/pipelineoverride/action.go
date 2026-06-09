package pipelineoverride

import (
	"maps"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type pipelineOverrideParam struct {
	Patch     map[string]interface{} `json:"patch"`
	AllowNext *bool                  `json:"allow_next,omitempty"`
}

// PipelineOverrideAction applies ctx.OverridePipeline from JSON param.
type PipelineOverrideAction struct{}

var _ maa.CustomActionRunner = &PipelineOverrideAction{}

func (a *PipelineOverrideAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	if arg == nil {
		log.Error().Str("component", "PipelineOverride").Msg("got nil custom action arg")
		return false
	}

	var params pipelineOverrideParam
	if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &params); err != nil {
		log.Error().Err(err).Str("component", "PipelineOverride").Msg("failed to parse custom_action_param")
		return false
	}

	if len(params.Patch) == 0 {
		log.Error().Str("component", "PipelineOverride").Msg("requires non-empty patch")
		return false
	}

	allowNext := params.AllowNext != nil && *params.AllowNext

	cleanPatch := make(map[string]interface{}, len(params.Patch))
	for nodeName, raw := range params.Patch {
		if strings.TrimSpace(nodeName) == "" {
			continue
		}
		nodeObj, ok := raw.(map[string]interface{})
		if !ok {
			log.Error().Str("node", nodeName).Msg("patch entry must be a JSON object")
			return false
		}
		cloned := maps.Clone(nodeObj)
		if !allowNext {
			delete(cloned, "next")
		}
		cleanPatch[nodeName] = cloned
	}

	if err := ctx.OverridePipeline(cleanPatch); err != nil {
		log.Error().Err(err).Str("component", "PipelineOverride").Msg("OverridePipeline failed")
		return false
	}

	log.Debug().
		Str("component", "PipelineOverride").
		Int("node_count", len(cleanPatch)).
		Msg("OverridePipeline applied")
	return true
}
