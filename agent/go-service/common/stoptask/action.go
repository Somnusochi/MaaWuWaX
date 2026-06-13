package stoptask

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

// StopTask posts a task stop request and succeeds immediately.
type StopTask struct{}

var _ maa.CustomActionRunner = &StopTask{}

func (a *StopTask) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	ctx.GetTasker().PostStop().Wait()
	return true
}
