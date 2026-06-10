// Package domain implements Custom atoms shared by stamina-spending domains.
package domain

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type BookTargetSelectAction struct{}

var _ maa.CustomActionRunner = &BookTargetSelectAction{}

type bookTargetParam struct {
	SerialNumber int `json:"serial_number"`
	TotalNumber  int `json:"total_number"`
}

func (a *BookTargetSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := bookTargetParam{SerialNumber: 1, TotalNumber: 1}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "BookTargetSelect").Msg("failed to parse param")
		}
	}
	if param.SerialNumber < 1 {
		param.SerialNumber = 1
	}
	if param.TotalNumber < param.SerialNumber {
		param.TotalNumber = param.SerialNumber
	}

	ctrl := ctx.GetTasker().GetController()
	row := param.SerialNumber
	if row > 4 {
		barTop := 112
		barBottom := 634
		y := barTop + (barBottom-barTop)*param.SerialNumber/param.TotalNumber
		ctrl.PostClick(1245, int32(y)).Wait()
		time.Sleep(900 * time.Millisecond)
		row = 4
	}

	y := int32(174 + (row-1)*118)
	ctrl.PostClick(1195, y).Wait()
	time.Sleep(1000 * time.Millisecond)

	log.Info().
		Str("component", "BookTargetSelect").
		Int("serial_number", param.SerialNumber).
		Int("total_number", param.TotalNumber).
		Msg("selected book target")
	return true
}

type SimulationSelectMaterialAction struct{}

var _ maa.CustomActionRunner = &SimulationSelectMaterialAction{}

type simulationMaterialParam struct {
	Material string `json:"material"`
}

func (a *SimulationSelectMaterialAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := simulationMaterialParam{Material: "shell_credit"}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "SimulationSelectMaterial").Msg("failed to parse param")
		}
	}

	index := 2
	switch param.Material {
	case "resonator_exp":
		index = 0
	case "weapon_exp":
		index = 1
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(282, int32(122+index*58)).Wait()
	time.Sleep(800 * time.Millisecond)
	ctrl.PostClick(1190, 648).Wait()
	time.Sleep(900 * time.Millisecond)
	ctrl.PostClick(1190, 648).Wait()
	time.Sleep(1200 * time.Millisecond)

	log.Info().
		Str("component", "SimulationSelectMaterial").
		Str("material", param.Material).
		Int("index", index).
		Msg("selected simulation material")
	return true
}
