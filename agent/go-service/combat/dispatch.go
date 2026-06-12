package combat

// dispatchCharStrategy routes to the per-character strategy function.
// Each character has its own file: char_<name>.go implementing perform<Name>().
func dispatchCharStrategy(c combatActor) {
	switch c.slot.Name {
	// ── Healers ──────────────────────────────────────────────────────
	case "verina":
		performVerina(c)
	case "shorekeeper":
		performShorekeeper(c)
	case "baizhi":
		performBaizhi(c)
	case "douling":
		performDouling(c)
	case "jianxin":
		performJianxin(c)
	case "taoqi":
		performTaoqi(c)
	case "denia":
		performDenia(c)
	case "youhu":
		performYouhu(c)
	case "mornye", "mornye_new", "moning", "moning_new":
		performMornye(c)

	// ── SubDPS ───────────────────────────────────────────────────────
	case "sanhua", "sanhua2":
		performSanhua(c)
	case "chixia":
		performChixia(c)
	case "yinlin":
		performYinlin(c)
	case "mortefi":
		performMortefi(c)
	case "zhezhi":
		performZhezhi(c)
	case "yuanwu":
		performYuanwu(c)
	case "danjin":
		performDanjin(c)
	case "qiuyuan", "chouyuan":
		performQiuyuan(c)
	case "galbrena":
		performGalbrena(c)
	case "roccia":
		performRoccia(c)
	case "ciaccona":
		performCiaccona(c)
	case "iuno":
		performIuno(c)
	case "cantarella":
		performCantarella(c)

	// ── MainDPS ──────────────────────────────────────────────────────
	case "jinhsi", "jinhsi2":
		performJinhsi(c)
	case "camellya":
		performCamellya(c)
	case "changli", "changli2", "chang_changli":
		performChangli(c)
	case "jiyan":
		performJiyan(c)
	case "encore":
		performEncore(c)
	case "calcharo":
		performCalcharo(c)
	case "xiangliyao":
		performXiangliyao(c)
	case "carlotta", "carlotta2":
		performCarlotta(c)
	case "phrolova":
		performPhrolova(c)
	case "cartethyia":
		performCartethyia(c)
	case "zani", "zani2":
		performZani(c)
	case "lupa":
		performLupa(c)

	// ── CUSTOM: ok-ww do_perform() overrides ─────────────────────────
	case "havocrover", "rover", "rover_male":
		performHavocRover(c)
	case "phoebe":
		performPhoebe(c)
	case "aemeath":
		performAemeath(c)
	case "augusta":
		performAugusta(c)
	case "brant":
		performBrant(c)
	case "chisa":
		performChisa(c)
	case "hiyuki":
		performHiyuki(c)
	case "linnai":
		performLinnai(c)
	case "lucy":
		performLucy(c)
	case "luhesi":
		performLuhesi(c)
	case "rebecca":
		performRebecca(c)
	case "xigelika":
		performXigelika(c)

	default:
		performDefault(c)
	}
}
