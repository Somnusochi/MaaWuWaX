package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type combatCharState struct {
	turnStart                  time.Time
	turnStartFreeze            int64
	lastPerform                time.Time
	lastSwitchOut              time.Time
	lastIntroSwitchIn          time.Time
	lastIntroSwitchFreeze      int64
	lastEcho                   time.Time
	lastHeavy                  time.Time
	lastLiberation             time.Time
	lastLiberationFreeze       int64
	lastResonance              time.Time
	lastResonanceFreeze        int64
	lastCrisis                 time.Time // Zani crisis-response window
	lastNightfall              time.Time // Zani nightfall follow-up window
	zaniLastChair              time.Time // Zani chair wait carry-over after liberation2
	zaniLastLiberation2        time.Time // Zani liberation2 exit timestamp
	zaniLastDodge              time.Time // Zani right-click recovery timestamp
	zaniLastChairFreeze        int64     // Zani chair wait freeze snapshot
	zaniLastDodgeFreeze        int64     // Zani dodge recovery freeze snapshot
	zaniLastAttackBreakthrough time.Time // Zani breakthrough combo completion time
	zaniLiberationFreeze       int64     // Zani liberation timestamp freeze snapshot
	zaniCrisisFreeze           int64     // Zani crisis timestamp freeze snapshot
	zaniNightfallFreeze        int64     // Zani nightfall timestamp freeze snapshot
	zaniLiberation2Freeze      int64     // Zani liberation2 timestamp freeze snapshot
	zaniAttackBreakFreeze      int64     // Zani breakthrough timestamp freeze snapshot
	zaniResonanceFreeze        int64     // Zani resonance timestamp freeze snapshot
	shorekeeperOutroAt         time.Time // Shorekeeper concerto outro timing
	shorekeeperOutroFreeze     int64     // Shorekeeper concerto outro freeze snapshot
	shorekeeperDodgeCount      int       // Shorekeeper remaining auto-dodge count
	lastEnhanceE               time.Time // Aemeath enhance-E window
	aemeathEnhanceEFreeze      int64     // Aemeath enhance-E timestamp freeze snapshot
	lastBuff                   time.Time // support buff tracking (Chisa)
	lastBuffFreeze             int64     // support buff timestamp freeze snapshot
	lastAnchor                 time.Time // Brant perform-anchor lock window
	brantAnchorFreeze          int64     // Brant perform-anchor freeze snapshot
	brantLiberationFreeze      int64     // Brant liberation timestamp freeze snapshot
	lastIntro                  time.Time // intro-window tracking (Luhesi/Aemeath intro-liberation window)
	aemeathIntroFreeze         int64     // Aemeath intro-liberation timestamp freeze snapshot
	aemeathIntroTime           int       // Aemeath per-outro intro_time: 14(Linnai/Lupa), 10(Changli), -1 disabled
	lastForte                  float64   // Camellyya heavy-attack forte sampling
	wolfReady                  bool      // Lupa drained-forte wolf follow-up
	liberationReady            bool      // Carlotta heavy-confirmed liberation setup
	libPermission              bool      // Hiyuki liberation hold permission
	incarnationActive          bool      // Jinhsi incarnation state
	incarnationCD              bool      // Jinhsi free intro follow-up entry
	pendingLiberation2         bool      // Aemeath heavy-prepared second liberation
	aemeathLiberationFreeze    int64     // Aemeath liberation timestamp freeze snapshot
	transformed                bool      // Cartethyia transformed state
	transformUntil             time.Time // Cartethyia transformed attack window
	cartethyiaTryMidAirOnce    bool      // Cartethyia one-shot mid-air attack fallback
	cartethyiaResAt            time.Time // Cartethyia big-form resonance timestamp
	cartethyiaN4At             time.Time // Cartethyia last big-form N4 completion
	cartethyiaResFreeze        int64     // Cartethyia big-form resonance freeze snapshot
	cartethyiaN4Freeze         int64     // Cartethyia N4 completion freeze snapshot
	zhezhiBlueReady            bool      // Zhezhi blue resonance follow-up state
	ciacconaInLiberation       bool      // Ciaccona liberation follow-up window
	zaniInLiberation           bool      // Zani liberation stance active
	lucyAlgorithmUntil         time.Time // Lucy algorithm compaction window
	lucyAlgorithmFreeze        int64     // Lucy algorithm compaction freeze snapshot
	waitingForForteDrop        bool      // Camellyya waiting for forte to visibly drop
	forteDropAt                time.Time // Camellyya forte-drop wait start
	forteDropFreeze            int64     // Camellyya forte-drop freeze snapshot
	carlottaContinueLiberation bool      // Carlotta interlock liberation continuation gate
	phrolovaResReady           bool      // Phrolova resonance-ready carry state
	changliEnhancedNormal      bool      // Changli enhanced-normal follow-up state
	encoreWarmupDone           bool      // Encore one-time warmup attack completed
	encoreHeavyFreeze          int64     // Encore heavy timestamp freeze snapshot
	encoreLiberationFreeze     int64     // Encore liberation timestamp freeze snapshot
	encoreResonanceFreeze      int64     // Encore resonance timestamp freeze snapshot
	camellyaInBudding          bool      // Camellya budding stance active
	forteDropStartForte        float64   // Camellyya forte value snapshot when waiting started
	havocRoverComboUntil       time.Time // Havoc Rover extended combo follow-up window
	jinhsiIncarnationUntil     time.Time // Jinhsi incarnation follow-up window
	galbrenaOpenerDone         bool      // Galbrena intro-style opener completed
	lupaLiberationFreeze       int64     // Lupa liberation timestamp freeze snapshot
	iunoHeavyFreeze            int64     // Iuno heavy timestamp freeze snapshot
	iunoLiberationFreeze       int64     // Iuno liberation timestamp freeze snapshot
	verinaHeavyFreeze          int64     // Verina heavy timestamp freeze snapshot
	phoebeEnterStatusCount     int       // Phoebe status-entry count in current support action chain
	phoebeLiberationCount      int       // Phoebe liberation casts in current action chain
	phoebeStarflashCount       int       // Phoebe starflash combo count in current action chain
	phoebeOutroCount           int       // Phoebe support outro count in current action chain
	phoebeStarLatched          bool      // Phoebe middle-star availability carried across action checks
	phoebeLastOutroAt          time.Time // Phoebe last successful support outro switch timestamp
	cantarellaHeavyFreeze      int64     // Cantarella heavy timestamp freeze snapshot
	mornyeHeavyFreeze          int64     // Mornye heavy timestamp freeze snapshot
	xiangliyaoLiberationFreeze int64     // Xiangliyao liberation timestamp freeze snapshot
}

type combatActor struct {
	action *CombatMainAction
	ctx    *maa.Context
	param  combatMainParam
	state  *combatCharState
	slot   charSlot
	roi    maa.Rect
}

func (a *CombatMainAction) performCharacterStrategy(ctx *maa.Context, param combatMainParam) {
	if a.charStates == nil {
		a.charStates = map[string]*combatCharState{}
	}
	slot := a.currentSlot()
	key := slot.Name
	if key == "" {
		key = "unknown"
	}
	state := a.charStates[key]
	if state == nil {
		state = &combatCharState{}
		a.charStates[key] = state
	}

	actor := combatActor{
		action: a,
		ctx:    ctx,
		param:  param,
		state:  state,
		slot:   slot,
		roi:    maa.Rect{0, 0, 1, 1},
	}

	log.Debug().Str("component", "Combat").Str("char", key).Str("role", string(slot.Role)).Msg("perform character strategy")

	state.turnStart = time.Now()
	state.turnStartFreeze = screenAnalyzer.FreezeDuration

	// Dispatch to per-character strategy file in char/ package.
	dispatchCharStrategy(actor)

	state.lastPerform = time.Now()
}

func (a *CombatMainAction) currentSlot() charSlot {
	if screenAnalyzer.CurrentIdx >= 0 && screenAnalyzer.CurrentIdx < len(screenAnalyzer.CharSlots) {
		return screenAnalyzer.CharSlots[screenAnalyzer.CurrentIdx]
	}
	for _, slot := range screenAnalyzer.CharSlots {
		if slot.Current {
			return slot
		}
	}
	return charSlot{Name: "unknown", Role: roleUnknown, Alive: true}
}

// combat primitives shared across character files.

func (c combatActor) attack() bool {
	c.run("Combat_RotationCombo")
	return true
}

func (c combatActor) attackFor(duration time.Duration) {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		c.attack()
		c.sleep(90 * time.Millisecond)
	}
}

func (c combatActor) skill() bool {
	if c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) < 2*time.Second {
		return false
	}
	c.run("Combat_RotationSkill1")
	c.state.lastResonance = time.Now()
	c.state.lastResonanceFreeze = screenAnalyzer.FreezeDuration
	return true
}

func (c combatActor) forceSkill() bool {
	c.run("Combat_RotationSkill1")
	c.state.lastResonance = time.Now()
	c.state.lastResonanceFreeze = screenAnalyzer.FreezeDuration
	return true
}

func (c combatActor) liberation() bool {
	if !c.param.UseLiberation {
		return false
	}
	if c.freezeElapsed(c.state.lastLiberation, c.state.lastLiberationFreeze) < 12*time.Second {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && c.isCurrentChar() {
		c.run("Combat_RotationLiberation")
		clicked = true
		if !screenAnalyzer.Liberation && c.currentLiberation() <= 0.05 {
			break
		}
		c.sleep(100 * time.Millisecond)
	}
	return confirmLiberationCast(c, clicked, 3*time.Second)
}

func (c combatActor) echo() bool {
	if c.currentEcho() <= 0.05 {
		return false
	}
	// Keep only a short debounce so repeated perform ticks do not double-send
	// the same echo before the UI has time to gray out. ok-ww's click_echo()
	// is driven by real-time availability rather than a long local cooldown.
	if !c.state.lastEcho.IsZero() && time.Since(c.state.lastEcho) < 300*time.Millisecond {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

func (c combatActor) echoImmediate() bool {
	if c.currentEcho() <= 0.05 {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

func (c combatActor) echoWait(wait time.Duration) bool {
	if wait <= 0 {
		return c.echo()
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if c.echo() {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return c.echo()
}

func (c combatActor) heavy(duration time.Duration) bool {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	c.sleep(duration)
	ctrl.PostTouchUp(0).Wait()
	return true
}

func (c combatActor) holdHeavyUntil(maxDuration, poll time.Duration, stop func() bool) bool {
	if maxDuration <= 0 {
		return false
	}
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	deadline := time.Now().Add(maxDuration)
	for time.Now().Before(deadline) {
		if stop != nil && stop() {
			break
		}
		c.sleep(poll)
	}
	ctrl.PostTouchUp(0).Wait()
	return true
}

func (c combatActor) holdSkillUntil(maxDuration, poll time.Duration, stop func() bool) bool {
	if maxDuration <= 0 {
		return false
	}
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("E")).Wait()
	deadline := time.Now().Add(maxDuration)
	for time.Now().Before(deadline) {
		if stop != nil && stop() {
			break
		}
		c.sleep(poll)
	}
	ctrl.PostKeyUp(keycode.MustCode("E")).Wait()
	c.state.lastResonance = time.Now()
	return true
}

func (c combatActor) forceLiberation() bool {
	c.run("Combat_RotationLiberation")
	return true
}

func confirmLiberationCast(c combatActor, clicked bool, backTimeout time.Duration) bool {
	if !clicked {
		return false
	}
	if backTimeout <= 0 {
		backTimeout = 3 * time.Second
	}
	leaveDeadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(leaveDeadline) && c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if c.isCurrentChar() || screenAnalyzer.Liberation || c.currentLiberation() > 0.05 {
		return false
	}
	freezeStart := time.Now()
	backDeadline := time.Now().Add(backTimeout)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	return true
}

func finishLiberationCast(c combatActor, clicked bool, backTimeout time.Duration) bool {
	if !confirmLiberationCast(c, clicked, backTimeout) {
		return false
	}
	c.state.lastLiberation = time.Now()
	c.state.lastLiberationFreeze = screenAnalyzer.FreezeDuration
	return true
}

func (c combatActor) currentResonance() float64 {
	return screenAnalyzer.ResonancePct
}

func (c combatActor) currentLiberation() float64 {
	return screenAnalyzer.LiberationPct
}

func (c combatActor) currentEcho() float64 {
	return screenAnalyzer.EchoPct
}

func (c combatActor) currentForte() float64 {
	return screenAnalyzer.FortePct
}

func (c combatActor) isCurrentChar() bool {
	if c.slot.Index < 0 || c.slot.Index >= len(screenAnalyzer.CharSlots) {
		return false
	}
	return screenAnalyzer.CurrentIdx == c.slot.Index
}

func (c combatActor) flying() bool {
	return screenAnalyzer.Flying
}

func (c combatActor) forteFull() bool {
	return screenAnalyzer.ForteFull
}

func (c combatActor) mouseForteFull() bool {
	return screenAnalyzer.MouseForteFull
}

func (c combatActor) eForteFull() bool {
	return screenAnalyzer.EForteFull
}

func (c combatActor) chisaDPS() bool {
	return c.param.ChisaDPS
}

func (c combatActor) iunoC6() bool {
	return c.param.IunoC6
}

func (c combatActor) hasLongAction() bool {
	return screenAnalyzer.HasLongAction
}

func (c combatActor) hasLongAction2() bool {
	return screenAnalyzer.HasLongAction2
}

func (c combatActor) linnaiColorFull() bool {
	return screenAnalyzer.LinnaiColorPct > 0.06
}

func (c combatActor) hiyukiLibHeavyReady() bool {
	return screenAnalyzer.HiyukiLibForte
}

func (c combatActor) hiyukiLeftPrompt() bool {
	return screenAnalyzer.HiyukiLeft
}

func (c combatActor) hiyukiRightPrompt() bool {
	return screenAnalyzer.HiyukiRight
}

func (c combatActor) ringElement() int {
	return screenAnalyzer.RingElement
}

func (c combatActor) lupaWolfReady() bool {
	return screenAnalyzer.LupaWolfReady
}

func (c combatActor) cartethyiaSwordBuffs() (bool, bool, bool) {
	return screenAnalyzer.CartethyiaSword1, screenAnalyzer.CartethyiaSword2, screenAnalyzer.CartethyiaSword3
}

func (c combatActor) cartethyiaBigLiberAvailable() bool {
	return screenAnalyzer.CartethyiaBigLib
}

func (c combatActor) cartethyiaIsSmall() bool {
	return screenAnalyzer.CartethyiaSmall
}

func (c combatActor) cartethyiaMidAirAttackAvailable() bool {
	return screenAnalyzer.CartethyiaMidAir
}

func (c combatActor) luhesiKickReady() bool {
	return screenAnalyzer.LuhesiKickReady
}

func (c combatActor) luhesiLibReady() bool {
	return screenAnalyzer.LuhesiLibReady
}

func (c combatActor) iunoHeavyReady() bool {
	return screenAnalyzer.IunoHeavyReady
}

func (c combatActor) iunoJumpReady() bool {
	return screenAnalyzer.IunoJumpReady
}

func (c combatActor) augustaLibReady() bool {
	return c.currentLiberation() > 0.05 && screenAnalyzer.AugustaLibReady
}

func (c combatActor) augustaMajestyReady() bool {
	return c.currentLiberation() > 0.05 && screenAnalyzer.AugustaMajesty
}

func (c combatActor) augustaProwessReady() bool {
	return screenAnalyzer.AugustaProwess
}

func (c combatActor) zhezhiBlueReady() bool {
	return screenAnalyzer.ZhezhiBluePct > 0.3
}

func (c combatActor) aemeathEnhanceEReady() bool {
	return screenAnalyzer.AemeathEnhanceE
}

func (c combatActor) aemeathLib2Ready() bool {
	return screenAnalyzer.AemeathLib2
}

func (c combatActor) zhezhiForteTier() int {
	switch forte := c.currentForte(); {
	case forte > 0.66:
		return 3
	case forte > 0.33:
		return 2
	case forte > 0.08:
		return 1
	default:
		return 0
	}
}

func (c combatActor) phoebeStarVisible() bool {
	return screenAnalyzer.PhoebeStarLight > 0.25 || screenAnalyzer.PhoebeStarBlue > 0.25
}

func (c combatActor) phoebeStarAvailable() bool {
	return c.phoebeStarVisible() || c.state.phoebeStarLatched
}

func (c combatActor) phoebeSupportMode() bool {
	return screenAnalyzer.PhoebeStarBlue >= screenAnalyzer.PhoebeStarLight
}

func (c combatActor) phoebePreferredSupport() bool {
	return c.teamHasAny("zani", "zani2") || (c.teamHas("cartethyia") && c.teamHasAny("havocrover", "rover", "rover_male"))
}

func (c combatActor) phoebeConfessionReady() bool {
	return screenAnalyzer.PhoebeRingBlue > 0.15
}

func (c combatActor) camellyaEphemeralReady() bool {
	return screenAnalyzer.CamellyaRedPct > 0.1
}

func (c combatActor) changliForteTier() int {
	if c.mouseForteFull() {
		return 4
	}
	pct := screenAnalyzer.ChangliFortePct
	switch {
	case pct >= 0.70:
		return 3
	case pct >= 0.45:
		return 2
	case pct >= 0.18:
		return 1
	default:
		return 0
	}
}

func (c combatActor) camellyaBuddingActive() bool {
	return screenAnalyzer.CamellyaBudding
}

func (c combatActor) camellyaForteValue(budding bool) float64 {
	if budding {
		return screenAnalyzer.CamellyaBudPct
	}
	return screenAnalyzer.CamellyaFortePct
}

func (c combatActor) zaniNightfallReady() bool {
	return screenAnalyzer.ZaniNightfallPct > 0.15
}

func (c combatActor) zaniForteValue() float64 {
	return screenAnalyzer.ZaniFortePct
}

func (c combatActor) zaniPrepared() bool {
	threshold := 0.4
	if c.teamHas("phoebe") {
		threshold = 0.6
	}
	if screenAnalyzer.ZaniBlazesPct >= threshold {
		return true
	}
	if !c.teamHas("phoebe") || screenAnalyzer.ZaniBlazesPct < 0.4 {
		return false
	}
	phoebeState := c.action.charStates["phoebe"]
	if phoebeState == nil || phoebeState.phoebeOutroCount < 1 {
		return false
	}
	return true
}

func (c combatActor) ciacconaAttribute() int {
	if c.teamHas("cartethyia") {
		return 3
	}
	if c.teamHas("phoebe") || c.teamHas("zani") || c.teamHas("zani2") {
		return 2
	}
	return 1
}

func (c combatActor) teamHas(name string) bool {
	for _, slot := range screenAnalyzer.CharSlots {
		if slot.Name == name && slot.Alive {
			return true
		}
	}
	return false
}

func (c combatActor) teamHasAny(names ...string) bool {
	for _, name := range names {
		if c.teamHas(name) {
			return true
		}
	}
	return false
}

func (c combatActor) forwardAttackFor(duration time.Duration) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("W")).Wait()
	c.attackFor(duration)
	ctrl.PostKeyUp(keycode.MustCode("W")).Wait()
}

func (c combatActor) jumpAttackFor(duration time.Duration) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostClickKey(keycode.MustCode("SPACE")).Wait()
	c.sleep(120 * time.Millisecond)
	c.attackFor(duration)
}

func (c combatActor) jump() {
	c.ctx.GetTasker().GetController().PostClickKey(keycode.MustCode("SPACE")).Wait()
}

func (c combatActor) rightClick() {
	c.ctx.GetTasker().GetController().PostClickV2(640, 360, 1, 1).Wait()
}

func (c combatActor) rightClickFor(duration time.Duration) {
	if duration <= 0 {
		return
	}
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		c.rightClick()
		c.sleep(50 * time.Millisecond)
	}
}

func (c combatActor) fBreak() {
	c.ctx.GetTasker().GetController().PostClickKey(keycode.MustCode("F")).Wait()
}

func (c combatActor) requestSwitch() {
	hasIntro := screenAnalyzer.ConcertoPct >= 1.0
	if (c.slot.Name == "zani" || c.slot.Name == "zani2") &&
		hasIntro &&
		c.state.zaniInLiberation &&
		zaniNightfallTimeLeft(c) > 0 &&
		zaniLiberationTimeLeft(c) >= 2*time.Second {
		c.sleep(200 * time.Millisecond)
		return
	}
	target := c.action.chooseSwitchTarget(time.Now(), hasIntro)
	if target < 0 {
		c.attackFor(200 * time.Millisecond)
		return
	}
	if c.slot.Name == "phoebe" && c.phoebePreferredSupport() && hasIntro {
		c.echoWait(1 * time.Second)
		c.state.phoebeOutroCount++
		c.state.phoebeLastOutroAt = time.Now()
	}
	c.run(switchActionName(target))
	c.state.lastSwitchOut = time.Now()
	if c.action != nil {
		if buffTime := c.action.effectiveBuffTime(c.slot.Name); buffTime > 0 && screenAnalyzer.ConcertoPct >= 1.0 {
			// Mirror ok-ww BaseChar.switch_out(): full-concerto switch-out refreshes
			// the active support/sub buff timestamp. Chisa support already records
			// the real grant time earlier, so keep that earlier timestamp instead
			// of replacing it with the later switch time.
			if c.slot.Name != "chisa" || c.state.lastBuff.IsZero() {
				c.state.lastBuff = time.Now()
				c.state.lastBuffFreeze = screenAnalyzer.FreezeDuration
			}
		}
	}
	switch c.slot.Name {
	case "encore":
		// ok-ww resets Encore's resonance-step state on switch_out so an old
		// step-1 window does not survive after leaving the field.
		c.state.lastResonance = time.Time{}
		c.state.encoreResonanceFreeze = 0
	}
	c.action.lastSwitchFrom[target] = c.slot
	c.action.lastSwitchIn[target] = time.Now()
	if hasIntro && target >= 0 && target < len(screenAnalyzer.CharSlots) {
		if targetName := screenAnalyzer.CharSlots[target].Name; targetName != "" {
			if targetState := c.action.charStates[targetName]; targetState != nil {
				targetState.lastIntroSwitchIn = time.Now()
				targetState.lastIntroSwitchFreeze = screenAnalyzer.FreezeDuration
			}
		}
	}
	c.action.lastSwitch = time.Now()
}

func (c combatActor) run(action string) {
	c.ctx.RunAction(action, c.roi, "", nil)
}

func (c combatActor) sleep(duration time.Duration) {
	if duration <= 0 {
		return
	}
	time.Sleep(duration)
}

func (c combatActor) addFreezeDuration(duration time.Duration) {
	if duration <= 0 {
		return
	}
	screenAnalyzer.FreezeDuration += duration.Nanoseconds()
}

func (c combatActor) freezeElapsed(since time.Time, freezeAt int64) time.Duration {
	if since.IsZero() {
		return time.Duration(1<<63 - 1)
	}
	elapsed := time.Since(since)
	freezeSince := screenAnalyzer.FreezeDuration - freezeAt
	if freezeSince > 0 {
		elapsed -= time.Duration(freezeSince)
	}
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func (c combatActor) performElapsed() time.Duration {
	if c.state == nil || c.state.turnStart.IsZero() {
		return 0
	}
	elapsed := time.Since(c.state.turnStart)
	freezeSinceStart := screenAnalyzer.FreezeDuration - c.state.turnStartFreeze
	if freezeSinceStart > 0 {
		elapsed -= time.Duration(freezeSinceStart)
	}
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

func (c combatActor) isFirstEngage() bool {
	if c.action == nil || c.state == nil || c.action.combatStart.IsZero() || c.state.turnStart.IsZero() {
		return false
	}
	delta := c.state.turnStart.Sub(c.action.combatStart)
	return delta >= 0 && delta < 100*time.Millisecond
}

func (c combatActor) recentlySwitchedIn(window time.Duration) bool {
	if window <= 0 {
		window = 1500 * time.Millisecond
	}
	idx := c.slot.Index
	if idx < 0 || idx >= len(c.action.lastSwitchIn) {
		return c.state.lastPerform.IsZero()
	}
	ts := c.action.lastSwitchIn[idx]
	if ts.IsZero() {
		return c.state.lastPerform.IsZero()
	}
	return c.freezeElapsed(ts, c.state.lastIntroSwitchFreeze) <= window
}

func (c combatActor) recentlyIntroSwitchedIn(window time.Duration) bool {
	if c.state == nil {
		return false
	}
	if window <= 0 {
		window = 1500 * time.Millisecond
	}
	if c.state.lastIntroSwitchIn.IsZero() {
		return false
	}
	return c.freezeElapsed(c.state.lastIntroSwitchIn, c.state.lastIntroSwitchFreeze) <= window
}

func (c combatActor) switchedFromRole(role charRole, window time.Duration) bool {
	if !c.recentlySwitchedIn(window) {
		return false
	}
	idx := c.slot.Index
	if idx < 0 || idx >= len(c.action.lastSwitchFrom) {
		return false
	}
	return c.action.lastSwitchFrom[idx].Role == role
}

func (c combatActor) switchedFromName(name string, window time.Duration) bool {
	if !c.recentlySwitchedIn(window) {
		return false
	}
	idx := c.slot.Index
	if idx < 0 || idx >= len(c.action.lastSwitchFrom) {
		return false
	}
	return c.action.lastSwitchFrom[idx].Name == name
}

func (c combatActor) switchedFromAny(window time.Duration, names ...string) bool {
	if !c.recentlySwitchedIn(window) {
		return false
	}
	idx := c.slot.Index
	if idx < 0 || idx >= len(c.action.lastSwitchFrom) {
		return false
	}
	from := c.action.lastSwitchFrom[idx].Name
	for _, name := range names {
		if from == name {
			return true
		}
	}
	return false
}

func (c combatActor) waitDown(timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	if !c.flying() {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !c.flying() {
			return true
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return !c.flying()
}

func (c combatActor) introReady() bool {
	return (c.currentResonance() > 0.05 || c.currentLiberation() > 0.05) && !c.flying()
}

func (c combatActor) waitIntro(timeout time.Duration, click bool) bool {
	if timeout <= 0 {
		timeout = 1200 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.introReady() {
			return true
		}
		if click {
			c.attack()
		}
		c.sleep(100 * time.Millisecond)
	}
	return c.introReady()
}

func (c combatActor) needFastPerform() bool {
	current := screenAnalyzer.CurrentIdx
	if current < 0 || current >= len(screenAnalyzer.CharSlots) {
		return false
	}
	now := time.Now()
	for idx, slot := range screenAnalyzer.CharSlots {
		if idx == current || !slot.Alive {
			continue
		}
		if c.action.characterSwitchPriority(idx, current, now) == switchPriorityMust {
			return true
		}
	}
	return false
}

// isOpenWorldAutoCombat mirrors ok-ww BaseChar.is_open_world_auto_combat():
// true when running the Farm 4C Echo dungeon/world task.
func (c combatActor) isOpenWorldAutoCombat() bool {
	return c.action != nil && c.action.currentTaskName == "Farm 4C Echo in Dungeon/World"
}

func shorekeeperPrepareOutro(c combatActor) {
	if screenAnalyzer.ConcertoPct < 1.0 {
		return
	}
	c.state.shorekeeperOutroAt = time.Now()
	c.state.shorekeeperOutroFreeze = screenAnalyzer.FreezeDuration
	c.state.shorekeeperDodgeCount = 5
}

func shorekeeperAutoDodge(c combatActor, condition func() bool) bool {
	if c.action == nil || condition == nil {
		return false
	}
	var state *combatCharState
	for _, name := range []string{"shorekeeper", "shorekeeper2"} {
		if s := c.action.charStates[name]; s != nil {
			state = s
			break
		}
	}
	if state == nil || state.shorekeeperDodgeCount <= 0 || state.shorekeeperOutroAt.IsZero() {
		return false
	}
	if (combatActor{action: c.action, state: state}).freezeElapsed(state.shorekeeperOutroAt, state.shorekeeperOutroFreeze) >= 30*time.Second {
		return false
	}
	clicked := false
	start := time.Now()
	for time.Since(start) < 1500*time.Millisecond {
		if !condition() {
			break
		}
		c.rightClickFor(50 * time.Millisecond)
		clicked = true
	}
	if clicked {
		state.shorekeeperDodgeCount--
	}
	return clicked
}
