# ok-ww 任务模块完整实现参考

> 全部 16 个任务的算法、坐标、颜色范围、OCR 正则、时序参数。坐标均为归一化 (0.0-1.0)。

## 1. BaseWWTask — 所有任务的基类 (1189 行)

### 颜色常量

```python
f_white_color = {'r': (235,255), 'g': (235,255), 'b': (235,255)}
echo_color = {'r': (200,255), 'g': (150,220), 'b': (130,170)}
book_bar_color = {'r': (190,255), 'g': (190,255), 'b': (190,255)}
```

### 场景检测

```python
# 大世界: world_earth_icon(0.55) + in_team() + NOT illusive_realm_exit
def in_world(self):
    return self.find_one('world_earth_icon', threshold=0.55) \
       and self.in_team() \
       and not self.find_one('illusive_realm_exit')

# 领域: illusive_realm_exit(0.7) + in_team() + NOT world_earth_icon
def in_realm(self):
    return self.find_one('illusive_realm_exit', threshold=0.7) \
       and self.in_team() \
       and not self.find_one('world_earth_icon')
```

### 队伍检测

```python
def in_team(self):
    # 模板匹配 char_1_text, char_2_text, char_3_text @ threshold 0.8
    # 3人队: 恰好缺1个 → 缺失位=当前角色
    # 2人队: 0缺失+2存在
    # 返回 (in_team: bool, current_index: int, count: int)
```

### F2 书本操作

```python
# 打开 F2 书
def openF2Book(self, feature="gray_book_all_monsters"):
    # Alt+click (0.77, 0.05), 回退 F2 键
    # 等待 feature 出现 @ box_gray_book, threshold 0.3

# Boss 分类标签页 Y 坐标:
# ningsu=0.28, moni=0.39, qiangdi=0.49, wuyin=0.6, zhange=0.7, mengyan=0.81

# 列表中点击第 N 个目标
def click_on_book_target(self, serial_number, total_number):
    # 前 4 个直接点击
    # 5+ 个: _find_book_scroll_top() + 计算滚动位置
    # 在 box(0.9113, 0.229, 0.9613, 0.861) 中找 boss_proceed(0.8)
```

### 体力系统

```python
def get_stamina(self):
    # OCR (0.49, 0.0, 0.92, 0.10) 匹配 number_re 和 stamina_re r'(\d+)/(\d+)'
    # 返回 (current, back_up, total)

def use_stamina(self, once, must_use=0):
    # current >= once*2: 双倍 (0.67, 0.62)
    # 否则单倍 (0.32, 0.62)
    # 检查 gem_add_stamina 弹窗, 点击确认 (0.70, 0.71)
```

### Echo 检测

```python
def yolo_find_echo(self, use_color=False, turn=True, time_out=8, threshold=0.5):
    # 4 方向旋转搜索: 居中 → YOLO detect → 找到则 walk_to_yolo_echo()
    # 阈值: realm 0.25 / world 0.65; 超时: realm 12s / world 4s

def walk_to_yolo_echo(self, time_out=8, echo_threshold=0.5):
    # 走向 YOLO 检测到的 echo, 丢失后维持方向走 3s
```

### 登录

```python
def wait_login(self):
    # 1. in_team → _logged_in = True
    # 2. login_close → 关闭公告
    # 3. OCR "登录"/"Log"/"登入" → 点击
    # 4. OCR "同意"+"隐私" → 点击同意
    # 5. OCR "开始游戏"/"进入游戏" → 点击
    # 6. switch_account → 选择账号
```

## 2. DailyTask — 一键日常 (190 行)

### 完整流程

```
1. ensure_main(180s) → 等待进入游戏世界
2. open_daily() → 打开 F2 书→任务页→OCR "/180" 读进度
3. need_nightmare? → NightmareNestTask
4. need_stamina? → TacetTask/ForgeryTask/SimulationTask
5. claim_daily() → 点击领取→100活跃度宝箱(0.93,0.88)
6. claim_mail() → 邮件图标(0.64,0.95)→全部领取(0.14,0.9)
7. claim_battle_pass() → Alt+click(0.86,0.05)→领取(0.68,0.91)×2
```

### 关键坐标

```python
# 日常进度 OCR: (0.1, 0.1, 0.5, 0.75) 匹配 r'^(\d+)/180$'
# 总活跃点 OCR: (0.19, 0.80, 0.30, 0.93) 匹配 number_re
# boss_proceed 按钮: box(0.803, 0.189, 0.960, 0.312)
# 领取按钮: (0.881, 0.237)
# 战令等级 OCR: (0.2, 0.13, 0.32, 0.22)
```

## 3. FarmEchoTask — 声骸刷取 (674 行)

### Boss 特殊处理

```python
# Lorelei: 每 660s 切换夜晚
# Sentry Construct/Lioness/Fallacy: combat_wait_time=5s
# Hyvatia: combat_wait_time=7s
# Fenrico/Fallacy/Lady of the Sea/Nameless: bypass_end_wait=True
# Nightmare Hecate: treat_as_not_in_realm=True
```

### 传送流程

```python
def teleport_to_configured_boss_and_prepare(self):
    # F2 → boss tab → Weekly(zhange, 9个) or Boss(qiangdi, 20个)
    # → click_on_book_target → 传送 → in_team_and_world
    # → walk_after_boss_teleport(): 向前走到 combat 或 F 交互
    # → enter_configured_boss_realm_from_f(): F→选等级→挑战(0.880,0.911)→确认(0.908,0.919)
```

### 主循环

```python
for count in range(Repeat Farm Count):
    # 1. in_realm_check(60s) → 重新检测领域状态
    # 2. manage_boss_interactions() → Boss 特化逻辑
    # 3. 可选: walk_to_boss → sleep combat_wait_time
    # 4. combat_once() → 战斗
    # 5. pick_echo() + yolo_find_echo / run_in_circle / walk_find_echo
    # 6. 宝箱交互/重启挑战
```

### 小地图导航

```python
def get_mini_map_turn_angle(self, feature, threshold=0.72):
    # box_minimap 内找 feature → rotate_arrow_and_find() (0-359°)
    # → _navigate_based_on_angle(angle): -45~45→W, 45~135→D, -135~-45→A, else→S

def go_to_boss_minimap(self):
    # 居中小地图 → find_boss_check_mark() → 计算角度 → WASD 方向 → walk
```

## 4. DomainTask / ForgeryTask / TacetTask / SimulationTask

### DomainTask 体力循环

```python
def farm_domain_with_recovery_loop(self, must_use, teleport_into_domain_once, max_recovery_retries=3):
    while True:
        # 1. open_F2_book_and_get_stamina() → 检查体力
        # 2. total < stamina_once or total < must_use → 回退返回
        # 3. teleport_into_domain_once()
        # 4. farm_in_domain(must_use)
        # 5. 死亡 → 重试 (最多3次) → revive_action()
```

### farm_in_domain

```python
def farm_in_domain(self, must_use=0):
    # 1. walk_until_f(4s) → pick_f
    # 2. combat_once()
    # 3. sleep(3s) → walk_to_treasure → pick_f
    # 4. use_stamina(once=stamina_once, must_use=must_use)
    # 5. 单倍 (0.32, 0.62) / 双倍 (0.67, 0.62)
    # 6. 继续 → "再次挑战" (0.68, 0.84)
    # 7. 确认弹窗 → "不再提醒" checkbox (0.49, 0.55)
```

### TacetTask 走路方法

```python
door_walk_method = {
    7: [["a", 0.3]],
    8: [["d", 0.6]],
    9: [["a", 1.5], ["w", 3], ["a", 2.5]],
}
# stamina_once = 60
```

### SimulationTask 材料选择

```python
# 'Resonator EXP' → index=0, click (0.22, 0.17)
# 'Weapon EXP'     → index=1, click (0.22, 0.25)
# 'Shell Credit'   → index=2, click (0.22, 0.33)
```

## 5. NightmareNestTask (164 行)

```python
count_re = re.compile(r"(\d{1,2})/(\d{1,2})")

def find_nest(self):
    # OCR (0.36, 0.13, 0.98, 0.91) 匹配 count_re
    # 分子 != 分母 且 分母 in ['24','36','48']

def combat_nest(self, nest):
    # 点击巢穴 → 传送 → in_team_and_world
    # F 进入 → 跑向战斗 → combat_once(0)
    # walk_find_echo(5s) 或 yolo_find_echo(30s)
```

## 6. AutoPickTask (75 行)

```python
default_config = {
    'Pick Up White List': ['吸收', 'Absorb'],
    'Pick Up Black List': ['开始合成', '领取奖励', 'Claim', '合成台']
}

def run(self):
    # 1. in_team() 检查
    # 2. find pick_up_f_hcenter_vcenter @ f_search_box (threshold 0.8)
    # 3. white_color_percentage > 0.5 才处理
    # 4. 检测 dialog_3_dots:
    #    有 → OCR 白名单 → 匹配则 send_fs() (按F×3)
    #    无 → OCR 黑名单 → 不匹配则 send_fs()
```

## 7. SkipDialogTask (30+100 行)

```python
def check_skip(self):
    # 1. try_click_skip() → 找 skip_dialog/skip_dialog_new (convert_dialog_icon, 0.75)
    # 2. skip_confirm(): skip_dialog_confirm + skip_dialog_check → 点复选框+确认
    # 3. btn_dialog_eye 检测 → 切换 auto_play → 点箭头或三点菜单
    # 4. skip_message(): message + message_dialog → 点下方关闭
```

## 8. FastTravelTask (27 行)

```python
def run(self):
    # gray_teleport 模板检测
    # OCR (0.7, 0.89, 1, 1) 匹配 "Travel"/"快速旅行"/"前往"/"Proceed"
    # → click_traval_button()
```

## 9. AutoRogueTask (436 行)

```python
white_list_buff = ["心流","悲鸣纪","余音贝","齿轮之心","全知之眼","指南针","医疗箱","妄语的残谱","激越的残谱"]
black_list_buff = ["雷暴","旋风","矛盾晶体"]

# 状态机:
# - 交易界面: OCR "交易" → ESC
# - 入口标题: OCR "千道门扉的异想" → 点击进入
# - 隐喻选择: OCR "隐喻获得" → buff_selector()
# - 战斗: in_combat() → combat_once()
# - 门: OCR "的记忆"/"梦乡的" → walk_to_gate()
# - 体力: OCR "补充结晶波片" → use_stamina
# - 宝箱: OCR "领取奖励" → 选体力档位点击
```

## 10. EnhanceEchoTask (346 行)

```python
default_config = {
    '必须有双爆': True,
    '双爆出现之前必须全有效词条': True,
    '双爆总计>=': 13.8,
    '首条双爆>=': 6.9,
    '有效词条>=': 3,
    '第一条必须为有效词条': True,
    '有效词条': ['暴击', '暴击伤害', '攻击百分比'],
}

# 算法:
# 1. OCR (0.82,0.86,0.97,0.96) "培养"
# 2. is_0_level(): OCR (0.65,0.35,1,0.57) "声骸技能"
# 3. 点击培养 → "阶段放入" → "强化并调谐"
# 4. OCR 副词条 (0.09,0.3,0.40,0.53) → 分离属性+数值
# 5. check_echo_stats(): 双爆检测 + 有效词条计数 + 首条双爆阈值
# 6. 成功: lock_and_esc() (C键上锁) / 失败: trash_and_esc() (Z键弃置)
```

## 11. ChangeEchoTask (100 行)

```python
# 仅支持 zh_CN
def run(self):
    # 1. find_echo_enhance() → "培养" → is_0_level()
    # 2. 点击培养 → 等"声骸强化"
    # 3. OCR 当前主属性 (0.09,0.20,0.15,0.26)
    # 4. 点击 (0.04,0.41) → "主音属性" → 选目标属性
    # 5. "确认" → "数据重构" (0.37,0.82,0.64,0.99)
    # 6. 等"获得声骸" → ESC 返回
```

## 12. FiveToOneTask (140 行)

```python
sets = ['凝夜白霜','熔山裂谷','彻空冥雷','啸谷长风','浮星祛暗','沉日劫明',
        '隐世回光','轻云出月','不绝余音','凌冽决断之心','此间永驻之光',
        '幽夜隐匿之帷','高天共奏之曲','无惧浪涛之勇','流云逝尽之空',
        '愿戴荣光之旅','奔狼燎原之焰']

def run(self):
    # 1. ensure_main → Alt+click(0.95,0.04) → "数据坞"
    # 2. (0.04,0.56) → "批量融合"(0.81,0.84)
    # 3. 17个套装逐个 merge_set(): 过滤→全选→融合→"获得声骸"→重复
```

## 13. 通用模式和约定

### 坐标系统
- 归一化坐标 (0.0-1.0)
- 部分标注 @3840x2160 基准分辨率自动缩放
- `box_of_screen(x1,y1,x2,y2)` 从归一化创建边界框
- `hcenter=True` 水平居中

### OCR 使用
- `ocr(box=..., match=regex_or_string)` 返回 Box 列表
- `wait_ocr()` 等待文字出现
- region: `(left, top, right, bottom)` 归一化

### 时序约定
- `after_sleep=X`: 操作后等待 X 秒
- `down_time=X`: 鼠标按下持续 X 秒
- `time_out=X`: 最大等待超时
- `interval=X`: 循环操作间隔
