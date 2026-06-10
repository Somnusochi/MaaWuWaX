package combat

type charRole string

const (
	roleUnknown charRole = "unknown"
	roleMain    charRole = "main_dps"
	roleSub     charRole = "sub_dps"
	roleHealer  charRole = "healer"
)

type charMeta struct {
	Name     string
	Template string
	Role     charRole
	BuffTime float64
}

type charSlot struct {
	Index    int      `json:"index"`
	Name     string   `json:"name"`
	Role     charRole `json:"role"`
	Detected bool     `json:"detected"`
	Current  bool     `json:"current"`
	Alive    bool     `json:"alive"`
}

var charTemplates = []charMeta{
	{Name: "yinlin", Template: "char_yinlin.png", Role: roleSub, BuffTime: 15},
	{Name: "verina", Template: "char_verina.png", Role: roleHealer, BuffTime: 15},
	{Name: "shorekeeper", Template: "char_shorekeeper.png", Role: roleHealer, BuffTime: 15},
	{Name: "taoqi", Template: "char_taoqi.png", Role: roleHealer, BuffTime: 15},
	{Name: "rover", Template: "char_rover.png", Role: roleMain},
	{Name: "rover_male", Template: "char_rover_male.png", Role: roleMain},
	{Name: "encore", Template: "char_encore.png", Role: roleMain},
	{Name: "jianxin", Template: "char_jianxin.png", Role: roleHealer, BuffTime: 15},
	{Name: "sanhua", Template: "char_sanhua.png", Role: roleSub, BuffTime: 15},
	{Name: "sanhua2", Template: "char_sanhua2.png", Role: roleSub, BuffTime: 15},
	{Name: "jinhsi", Template: "char_jinhsi.png", Role: roleMain},
	{Name: "jinhsi2", Template: "char_jinhsi2.png", Role: roleMain},
	{Name: "yuanwu", Template: "char_yuanwu.png", Role: roleSub, BuffTime: 15},
	{Name: "changli", Template: "chang_changli.png", Role: roleMain},
	{Name: "chang_changli", Template: "chang_changli.png", Role: roleMain},
	{Name: "changli2", Template: "char_changli2.png", Role: roleMain},
	{Name: "chixia", Template: "char_chixia.png", Role: roleMain},
	{Name: "danjin", Template: "char_danjin.png", Role: roleSub, BuffTime: 15},
	{Name: "baizhi", Template: "char_baizhi.png", Role: roleHealer, BuffTime: 15},
	{Name: "calcharo", Template: "char_calcharo.png", Role: roleMain},
	{Name: "jiyan", Template: "char_jiyan.png", Role: roleMain},
	{Name: "mortefi", Template: "char_mortefi.png", Role: roleSub, BuffTime: 15},
	{Name: "zhezhi", Template: "char_zhezhi.png", Role: roleSub, BuffTime: 15},
	{Name: "xiangliyao", Template: "char_xiangliyao.png", Role: roleMain},
	{Name: "camellya", Template: "char_camellya.png", Role: roleMain},
	{Name: "youhu", Template: "char_youhu.png", Role: roleHealer, BuffTime: 15},
	{Name: "carlotta", Template: "char_carlotta.png", Role: roleMain},
	{Name: "carlotta2", Template: "char_carlotta2.png", Role: roleMain},
	{Name: "roccia", Template: "char_roccia.png", Role: roleSub, BuffTime: 15},
	{Name: "phoebe", Template: "char_phoebe.png", Role: roleSub, BuffTime: 15},
	{Name: "brant", Template: "char_brant.png", Role: roleHealer, BuffTime: 15},
	{Name: "cantarella", Template: "char_cantarella.png", Role: roleHealer, BuffTime: 15},
	{Name: "zani", Template: "char_zani.png", Role: roleMain},
	{Name: "zani2", Template: "char_zani2.png", Role: roleMain},
	{Name: "ciaccona", Template: "char_ciaccona.png", Role: roleSub, BuffTime: 15},
	{Name: "cartethyia", Template: "char_cartethyia.png", Role: roleMain},
	{Name: "lupa", Template: "char_lupa.png", Role: roleSub, BuffTime: 15},
	{Name: "phrolova", Template: "char_phrolova.png", Role: roleMain},
	{Name: "augusta", Template: "Augusta.png", Role: roleMain},
	{Name: "iuno", Template: "char_iuno.png", Role: roleSub, BuffTime: 15},
	{Name: "galbrena", Template: "char_galbrena.png", Role: roleMain},
	{Name: "qiuyuan", Template: "char_chouyuan.png", Role: roleSub, BuffTime: 15},
	{Name: "chouyuan", Template: "char_chouyuan.png", Role: roleSub, BuffTime: 15},
	{Name: "chisa", Template: "char_chisa.png", Role: roleHealer, BuffTime: 12},
	{Name: "denia", Template: "char_denia.png", Role: roleSub, BuffTime: 14},
	{Name: "douling", Template: "char_douling.png", Role: roleHealer, BuffTime: 15},
	{Name: "linnai", Template: "char_linnai.png", Role: roleSub, BuffTime: 15},
	{Name: "mornye", Template: "char_moning.png", Role: roleHealer, BuffTime: 15},
	{Name: "mornye_new", Template: "char_moning_new.png", Role: roleHealer, BuffTime: 15},
	{Name: "moning", Template: "char_moning.png", Role: roleHealer, BuffTime: 15},
	{Name: "moning_new", Template: "char_moning_new.png", Role: roleHealer, BuffTime: 15},
	{Name: "aemeath", Template: "char_aemeath.png", Role: roleMain},
	{Name: "xigelika", Template: "char_xigelika.png", Role: roleMain},
	{Name: "luhesi", Template: "char_luhesi.png", Role: roleMain},
	{Name: "hiyuki", Template: "char_hiyuki.png", Role: roleMain},
	{Name: "lucy", Template: "char_lucy.png", Role: roleMain},
	{Name: "rebecca", Template: "char_rebecca.png", Role: roleSub, BuffTime: 15},
}
