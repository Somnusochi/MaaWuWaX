package combat

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ── CombatStateRecognition — ok-ww CombatCheck port ───────────────────

type CombatStateRecognition struct{}

var _ maa.CustomRecognitionRunner = &CombatStateRecognition{}

func (r *CombatStateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	screenAnalyzer.Update(ctx, arg.Img)

	state := "fighting"
	if screenAnalyzer.HasDodge {
		state = "dodge"
	} else if !screenAnalyzer.InCombat() {
		if screenAnalyzer.PickupF {
			state = "loot"
		} else {
			state = "idle"
		}
	} else if screenAnalyzer.ConcertoPct >= 1.0 {
		state = "switch"
	}

	detail, _ := sonic.Marshal(map[string]any{
		"state":    state,
		"target":   screenAnalyzer.HasTarget,
		"hp":       screenAnalyzer.HasHPBar,
		"concerto": screenAnalyzer.ConcertoPct,
	})

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1280, 720},
		Detail: string(detail),
	}, true
}

// ── CombatMainAction — state machine driven action ────────────────────

type CombatMainAction struct {
	phase           string
	lastDodge       time.Time
	lastAction      time.Time
	lastSwitch      time.Time
	lastSwitchIn    [3]time.Time
	lastSwitchFrom  [3]charSlot
	combatStart     time.Time
	rotationIdx     int
	switchIdx       int
	charStates      map[string]*combatCharState
	param           combatMainParam
	currentTaskName string
}

type combatMainParam struct {
	UseLiberation bool `json:"use_liberation"`
	AutoTarget    bool `json:"auto_target"`
	SwitchHealer  bool `json:"switch_healer"`
	ChisaDPS      bool `json:"chisa_dps"`
	IunoC6        bool `json:"iuno_c6"`
}

type switchPriority int

const (
	switchPriorityNo   switchPriority = -1
	switchPriorityAuto switchPriority = 0
	switchPriorityMust switchPriority = 2
)

func (a *CombatMainAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := combatMainParam{UseLiberation: true, AutoTarget: true}
	a.currentTaskName = arg.CurrentTaskName
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "Combat").Msg("failed to parse param")
		}
	}
	a.param = param

	now := time.Now()
	roi := maa.Rect{0, 0, 1, 1}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostScreencap().Wait()

	// Track frame for freeze compensation (ok-ww: time_elapsed_accounting_for_freeze)
	screenAnalyzer.TrackFrame(now.UnixNano())

	img, err := ctrl.CacheImage()
	if err == nil && img != nil {
		screenAnalyzer.Update(ctx, img)
	}

	// ── Dodge ──
	if screenAnalyzer.HasDodge && time.Since(a.lastDodge) > 500*time.Millisecond {
		ctx.RunAction("Combat_Dodge", roi, "", nil)
		a.lastDodge = now
		time.Sleep(120 * time.Millisecond)
		return true
	}

	// ── Out of combat ──
	if !screenAnalyzer.InCombat() {
		a.combatStart = time.Time{}
		screenAnalyzer.FreezeDuration = 0
		screenAnalyzer.LastFrameTime = 0
		if param.SwitchHealer && time.Since(a.lastSwitch) > 2*time.Second {
			current := a.currentSlot()
			if a.effectiveRole(current) != roleHealer {
				for i, slot := range screenAnalyzer.CharSlots {
					if a.effectiveRole(slot) == roleHealer && slot.Alive && i != screenAnalyzer.CurrentIdx {
						ctx.RunAction(switchActionName(i), roi, "", nil)
						a.lastSwitchIn[i] = now
						a.lastSwitch = now
						time.Sleep(300 * time.Millisecond)
						return true
					}
				}
			}
		}
		if screenAnalyzer.PickupF {
			ctx.RunAction("Combat_LootAction", roi, "", nil)
			time.Sleep(250 * time.Millisecond)
			return true
		}
		if param.AutoTarget && !screenAnalyzer.HasTarget {
			// Auto-target: middle-click to lock on to nearest enemy (ok-ww: target_enemy)
			ctrl.PostClick(640, 360).Wait()
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(120 * time.Millisecond)
		return true
	}

	if a.combatStart.IsZero() {
		a.combatStart = now
	}

	// ── Health-based switch: if current char is low HP, switch to healer ──
	if screenAnalyzer.HasHealthLow() && time.Since(a.lastSwitch) > 3*time.Second {
		for i, slot := range screenAnalyzer.CharSlots {
			if a.effectiveRole(slot) == roleHealer && slot.Alive && i != screenAnalyzer.CurrentIdx {
				a.performCharacterStrategy(ctx, param) // quick perform then switch
				a.lastSwitch = now
				time.Sleep(120 * time.Millisecond)
				return true
			}
		}
	}

	// ── Concerto switch ──
	if time.Since(a.lastSwitch) > 8*time.Second && screenAnalyzer.ConcertoPct >= 1.0 {
		target := a.chooseSwitchTarget(now, true)
		if target >= 0 {
			ctx.RunAction(switchActionName(target), roi, "", nil)
			a.lastSwitchFrom[target] = a.currentSlot()
			a.lastSwitchIn[target] = now
			if targetName := screenAnalyzer.CharSlots[target].Name; targetName != "" {
				if state := a.charStates[targetName]; state != nil {
					state.lastIntroSwitchIn = now
				}
			}
			a.switchIdx++
			a.lastSwitch = now
			a.rotationIdx = 0
			time.Sleep(300 * time.Millisecond)
			return true
		}
	}

	if time.Since(a.lastAction) <= 120*time.Millisecond {
		return true
	}

	a.performCharacterStrategy(ctx, param)
	a.rotationIdx++
	a.lastAction = now

	if screenAnalyzer.PickupF {
		ctx.RunAction("Combat_LootAction", roi, "", nil)
		time.Sleep(250 * time.Millisecond)
	}

	return true
}

// ── CharacterDetectRecognition ────────────────────────────────────────

type CharacterDetectRecognition struct{}

var _ maa.CustomRecognitionRunner = &CharacterDetectRecognition{}

func (r *CharacterDetectRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	screenAnalyzer.Update(ctx, arg.Img)
	alive := []int{}
	for i, a := range screenAnalyzer.CharAlive {
		if a {
			alive = append(alive, i+1)
		}
	}
	d, _ := sonic.Marshal(map[string]any{
		"alive":       alive,
		"current":     screenAnalyzer.CurrentIdx + 1,
		"team_size":   screenAnalyzer.TeamSize,
		"char_slots":  screenAnalyzer.CharSlots,
		"concerto":    screenAnalyzer.ConcertoPct,
		"liberation":  screenAnalyzer.Liberation,
		"in_combat":   screenAnalyzer.InCombat(),
		"has_target":  screenAnalyzer.HasTarget,
		"has_hp_bar":  screenAnalyzer.HasHPBar,
		"has_boss_hp": screenAnalyzer.HasBossHP,
	})
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1280, 720},
		Detail: string(d),
	}, true
}

func (a *CombatMainAction) chooseSwitchTarget(now time.Time, hasIntro bool) int {
	current := screenAnalyzer.CurrentIdx
	if current < 0 || current >= 3 {
		return a.fallbackSwitchTarget()
	}

	candidates := make([]int, 0, 2)
	for i, slot := range screenAnalyzer.CharSlots {
		if i == current || !slot.Alive {
			continue
		}
		candidates = append(candidates, i)
	}
	if len(candidates) == 0 {
		return -1
	}

	must := make([]int, 0, len(candidates))
	filtered := make([]int, 0, len(candidates))
	for _, idx := range candidates {
		switch a.characterSwitchPriority(idx, current, now) {
		case switchPriorityMust:
			must = append(must, idx)
		case switchPriorityNo:
			continue
		default:
			filtered = append(filtered, idx)
		}
	}
	if len(must) > 0 {
		return a.oldest(must)
	}
	if hasIntro {
		if len(filtered) == 0 {
			return -1
		}
		if target := a.oldestByRole(filtered, roleMain, now, false); target >= 0 {
			return target
		}
		if target := a.oldestByRole(filtered, roleSub, now, false); target >= 0 {
			return target
		}
		if target := a.oldestByRole(filtered, roleHealer, now, false); target >= 0 {
			return target
		}
		return a.oldest(filtered)
	}
	if len(filtered) == 0 {
		return -1
	}
	candidates = filtered
	if len(candidates) == 0 {
		return -1
	}
	if available := a.filterTargetsWithoutSwitchCD(candidates, now); len(available) > 0 {
		candidates = available
	}

	currentRole := a.effectiveRole(screenAnalyzer.CharSlots[current])
	if currentRole == roleMain {
		if target := a.oldestByRole(candidates, roleSub, now, true); target >= 0 {
			return target
		}
		if target := a.oldestByRole(candidates, roleHealer, now, true); target >= 0 {
			return target
		}
	}

	if currentRole == roleSub || currentRole == roleHealer {
		if target := a.oldestByRole(candidates, roleMain, now, false); target >= 0 {
			return target
		}
	}

	return a.oldest(candidates)
}

func (a *CombatMainAction) filterTargetsWithoutSwitchCD(candidates []int, now time.Time) []int {
	filtered := make([]int, 0, len(candidates))
	for _, idx := range candidates {
		if !a.targetHasSwitchCD(idx, now) {
			filtered = append(filtered, idx)
		}
	}
	return filtered
}

func (a *CombatMainAction) targetHasSwitchCD(idx int, now time.Time) bool {
	if a == nil || idx < 0 || idx >= len(screenAnalyzer.CharSlots) {
		return false
	}
	slot := screenAnalyzer.CharSlots[idx]
	state := a.charStates[slot.Name]
	if state == nil || state.lastSwitchOut.IsZero() {
		return false
	}
	return now.Sub(state.lastSwitchOut) <= time.Second
}

func (a *CombatMainAction) fallbackSwitchTarget() int {
	for step := 0; step < 3; step++ {
		idx := (a.switchIdx + step + 1) % 3
		if screenAnalyzer.CharAlive[idx] {
			return idx
		}
	}
	return -1
}

func (a *CombatMainAction) oldestByRole(candidates []int, role charRole, now time.Time, preferExpiredBuff bool) int {
	filtered := make([]int, 0, len(candidates))
	for _, idx := range candidates {
		slot := screenAnalyzer.CharSlots[idx]
		if a.effectiveRole(slot) == role {
			filtered = append(filtered, idx)
		}
	}
	if len(filtered) == 0 {
		return -1
	}
	if preferExpiredBuff {
		expired := make([]int, 0, len(filtered))
		for _, idx := range filtered {
			meta := screenAnalyzer.CharSlots[idx]
			if a.effectiveRole(meta) == roleUnknown || !a.hasActiveBuff(idx, now) {
				expired = append(expired, idx)
			}
		}
		if len(expired) > 0 {
			filtered = expired
		}
	}
	return a.oldest(filtered)
}

func (a *CombatMainAction) hasActiveBuff(idx int, now time.Time) bool {
	if a == nil || idx < 0 || idx >= len(screenAnalyzer.CharSlots) {
		return false
	}
	slot := screenAnalyzer.CharSlots[idx]
	buffTime := a.effectiveBuffTime(slot.Name)
	if buffTime <= 0 {
		return false
	}
	state := a.charStates[slot.Name]
	if state == nil || state.lastBuff.IsZero() {
		return false
	}
	return combatActor{action: a, state: state}.freezeElapsed(state.lastBuff, state.lastBuffFreeze).Seconds() < buffTime
}

func (a *CombatMainAction) oldest(candidates []int) int {
	best := candidates[0]
	bestTime := a.lastSwitchIn[best]
	for _, idx := range candidates[1:] {
		if a.lastSwitchIn[idx].Before(bestTime) {
			best = idx
			bestTime = a.lastSwitchIn[idx]
		}
	}
	return best
}

func charBuffTime(name string) float64 {
	for _, meta := range charTemplates {
		if meta.Name == name {
			if meta.BuffTime <= 0 {
				return 0
			}
			return meta.BuffTime
		}
	}
	return 0
}

func (a *CombatMainAction) effectiveRole(slot charSlot) charRole {
	if a != nil && a.param.ChisaDPS && slot.Name == "chisa" {
		return roleMain
	}
	if a != nil && a.param.IunoC6 && slot.Name == "iuno" {
		return roleMain
	}
	return slot.Role
}

func (a *CombatMainAction) effectiveBuffTime(name string) float64 {
	if a != nil && a.param.ChisaDPS && name == "chisa" {
		return 0
	}
	if a != nil && a.param.IunoC6 && name == "iuno" {
		return 0
	}
	return charBuffTime(name)
}

func ciacconaCartethyiaTransformed(a *CombatMainAction) bool {
	if a == nil || a.charStates == nil {
		return false
	}
	state := a.charStates["cartethyia"]
	return state != nil && state.transformed
}

func (a *CombatMainAction) characterSwitchPriority(targetIdx, currentIdx int, now time.Time) switchPriority {
	target := screenAnalyzer.CharSlots[targetIdx]
	current := screenAnalyzer.CharSlots[currentIdx]
	if target.Name == "" || !target.Alive {
		return switchPriorityNo
	}
	state := a.charStates[target.Name]
	if state == nil {
		return switchPriorityAuto
	}

	sinceLib := combatActor{action: a, state: state}.freezeElapsed(state.lastLiberation, state.lastLiberationFreeze)
	sinceSwitchIn := now.Sub(a.lastSwitchIn[targetIdx])
	hasIntro := screenAnalyzer.ConcertoPct >= 1.0

	switch target.Name {
	case "carlotta", "carlotta2":
		if hasIntro && current.Name == "zhezhi" {
			return switchPriorityMust
		}
	case "phoebe":
		if !hasIntro &&
			!state.phoebeLastOutroAt.IsZero() &&
			now.Sub(state.phoebeLastOutroAt) < 4500*time.Millisecond {
			return switchPriorityNo
		}
	case "aemeath":
		libElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastLiberation, state.aemeathLiberationFreeze)
		if state.pendingLiberation2 || (libElapsed > 17*time.Second && libElapsed < 25*time.Second) {
			return switchPriorityMust
		}
	case "cartethyia":
		if state.transformed {
			return switchPriorityMust
		}
	case "brant":
		anchorElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastAnchor, state.brantAnchorFreeze)
		libElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastLiberation, state.brantLiberationFreeze)
		if !state.lastAnchor.IsZero() && anchorElapsed < 4*time.Second {
			return switchPriorityNo
		}
		if libElapsed < 12*time.Second || (hasIntro && current.Name == "lupa") {
			return switchPriorityMust
		}
	case "encore":
		heavyElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastHeavy, state.encoreHeavyFreeze)
		libElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastLiberation, state.encoreLiberationFreeze)
		resElapsed := combatActor{action: a, state: state}.freezeElapsed(state.lastResonance, state.encoreResonanceFreeze)
		if heavyElapsed < 4600*time.Millisecond {
			return switchPriorityNo
		}
		if libElapsed < 9500*time.Millisecond || resElapsed < 2*time.Second {
			return switchPriorityMust
		}
	case "camellya":
		if hasIntro {
			return switchPriorityMust
		}
	case "cantarella":
		if screenAnalyzer.ConcertoPct >= 1.0 &&
			(current.Name == "roccia" || current.Name == "sanhua" || current.Name == "sanhua2") {
			return switchPriorityMust
		}
	case "changli", "changli2", "chang_changli":
		if hasIntro && current.Name == "brant" {
			return switchPriorityMust
		}
	case "lupa":
		if sinceLib < 12*time.Second ||
			(hasIntro && (current.Name == "changli" || current.Name == "changli2" || current.Name == "chang_changli")) {
			return switchPriorityMust
		}
	case "linnai":
		if hasIntro &&
			(current.Name == "mornye" || current.Name == "mornye_new" || current.Name == "moning" || current.Name == "moning_new") {
			return switchPriorityMust
		}
	case "luhesi":
		if hasIntro && (state.lastIntro.IsZero() || now.Sub(state.lastIntro) > 24*time.Second) {
			return switchPriorityMust
		}
	case "phrolova":
		if sinceLib < 24*time.Second {
			if hasIntro && current.Name == "cantarella" && sinceLib > 14*time.Second {
				return switchPriorityMust
			}
			return switchPriorityNo
		}
	case "hiyuki":
		if hasIntro && current.Name == "linnai" {
			return switchPriorityMust
		}
	case "shorekeeper":
		if hasIntro && current.Name == "augusta" {
			return switchPriorityMust
		}
	case "jinhsi", "jinhsi2":
		if screenAnalyzer.ConcertoPct >= 1.0 || state.incarnationActive || state.incarnationCD {
			return switchPriorityMust
		}
		return switchPriorityNo
	case "zani", "zani2":
		actor := combatActor{action: a, state: state}
		if state.zaniInLiberation {
			return switchPriorityMust
		}
		if hasIntro && zaniCrisisTimeLeft(actor) > 0 {
			return switchPriorityNo
		}
	case "chisa":
		if target.Role == roleHealer && sinceSwitchIn > 12*time.Second {
			return switchPriorityMust
		}
	case "denia":
		if sinceSwitchIn > 14*time.Second {
			return switchPriorityMust
		}
	case "mornye", "mornye_new", "moning", "moning_new":
		if hasIntro {
			if current.Name == "aemeath" {
				return switchPriorityMust
			}
			if a.hasTeamName("linnai") && current.Name != "linnai" {
				return switchPriorityMust
			}
		}
	case "zhezhi":
		actor := combatActor{action: a, state: state}
		if zhezhiCarlottaReady(actor) {
			return switchPriorityMust
		}
		if !a.hasActiveBuff(targetIdx, now) {
			return switchPriorityMust
		}
	case "ciaccona":
		actor := combatActor{action: a, state: state}
		attr := actor.ciacconaAttribute()
		if state.ciacconaInLiberation {
			if attr == 2 && sinceLib < 20*time.Second {
				return switchPriorityNo
			}
			if attr == 3 && (sinceLib < 8*time.Second || ciacconaCartethyiaTransformed(a)) {
				return switchPriorityNo
			}
		}
		if !a.hasActiveBuff(targetIdx, now) {
			return switchPriorityMust
		}
	case "sanhua", "sanhua2", "yinlin", "mortefi", "yuanwu", "danjin", "qiuyuan", "chouyuan", "roccia", "iuno":
		if !a.hasActiveBuff(targetIdx, now) {
			return switchPriorityMust
		}
	}
	return switchPriorityAuto
}

func switchActionName(index int) string {
	return []string{"Combat_SwitchChar1", "Combat_SwitchChar2", "Combat_SwitchChar3"}[index]
}

func (a *CombatMainAction) hasTeamName(name string) bool {
	for _, slot := range screenAnalyzer.CharSlots {
		if slot.Alive && slot.Name == name {
			return true
		}
	}
	return false
}
