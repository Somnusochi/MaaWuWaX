# ok-ww 战斗系统完整实现

> 从 ok-wuthering-waves 项目中提取，作为 MaaWuWaX 移植参考。

## 1. 架构总览

```
BaseTask (ok框架)
  → BaseWWTask (鸣潮通用工具)
    → CombatCheck (战斗状态检测)
      → BaseCombatTask (队伍管理、切换)
        → AutoCombatTask (触发器入口)
```

## 2. 战斗进入/退出检测 (`CombatCheck.py`)

### 2.1 核心算法 `do_check_in_combat(target)`

**已处于战斗中 (`self._in_combat == True`):**
1. 大招动画中 → 直接返回 True
2. 检查 `scene.in_combat()` 缓存
3. `check_f_break()` 检测 QTE
4. 当前角色 `skip_combat_check()` 钩子
5. `on_combat_check()` 钩子（拾取、领域检测等）
6. `has_target()` — 目标锁定指示器
7. `combat_end_condition` 回调
8. `target_enemy(wait=True)` 重新锁定
9. `handle_monthly_card()` 月卡检查
10. 全部失败 → `reset_to_false()`

**未在战斗中 (`self._in_combat == False`):**

```python
# CombatCheck.py L133-176
def do_check_in_combat(self, target):
    if self._in_combat:
        # ...已在战斗中的逻辑
    else:
        from src.task.AutoCombatTask import AutoCombatTask
        has_target = self.has_target()
        if not has_target and target:
            self.log_debug('try target')
            self.middle_click(after_sleep=0.1)
        in_combat = has_target or (
            (self.config.get('Auto Target') or not isinstance(self, AutoCombatTask))
            and self.check_health_bar()
        )
        if in_combat:
            if not has_target and not self.target_enemy(wait=True):
                # 锁定失败
                return False
            self.has_lavitator = self.find_one('edge_levitator', threshold=0.65)
            self._in_combat = self.load_chars()
            return self._in_combat
```

关键逻辑：`in_combat = has_target or ((Auto Target 开启 或 非 AutoCombatTask) and check_health_bar())`

### 2.2 `has_target()` — 模板匹配置信度 0.6

```python
# CombatCheck.py L252-280
def has_target(self, double_check=False):
    threshold = 0.6
    has_name, no_name = self.get_target_names()
    scale = 1.2 if self.is_browser() else 1.1
    # 搜索 4 个区域
    best = self.find_best_match_in_box(
        self.get_box_by_name(has_name).scale(scale), [has_name, no_name], threshold=threshold)
    if not best:
        best = self.find_best_match_in_box(
            self.get_box_by_name('box_target_enemy_long'), [has_name, no_name], threshold=threshold)
    if not best:
        best = self.find_best_match_in_box(
            self.get_box_by_name('target_box_long2'), [has_name, no_name], threshold=threshold)
    if not best:
        best = self.find_best_match_in_box(
            self.get_box_by_name(has_name).scale(1.1, 2.0), [has_name, no_name], threshold=threshold)
    return best and best.name == has_name
```

### 2.3 `has_health_bar()` — 红色血条颜色检测

```python
# CombatCheck.py L297-322
def has_health_bar(self):
    if self._in_combat:
        min_width = self.width_of_screen(12 / 3840)   # 12px → 战中
    else:
        min_width = self.width_of_screen(100 / 3840)  # 100px → 战前
    min_height = self.height_of_screen(9 / 2160)
    max_height = min_height * 3

    # 红色血条: r:174-225, g:55-85, b:55-76
    boxes = find_color_rectangles(self.frame, enemy_health_color_red,
                                   min_width, min_height, max_height=max_height)
    if len(boxes) > 0:
        return True
    else:
        # Boss 血条: 屏幕顶部区域 (1269,58)-(2533,200) @3840x2160
        boxes = find_color_rectangles(self.frame, boss_health_color, min_width,
                                       min_height * 1.3,
                                       box=self.box_of_screen(1269/3840, 58/2160,
                                                               2533/3840, 200/2160))
        return len(boxes) == 1
```

### 2.4 `target_enemy()` — 鼠标中键锁定

```python
# CombatCheck.py L282-295
def target_enemy(self, wait=True):
    if not wait:
        self.middle_click()
    else:
        if self.has_target():
            return True
        else:
            start = time.time()
            while time.time() - start < self.target_enemy_time_out:  # 默认 3s
                self.middle_click(interval=0.2)   # 每 0.2s 中键一次
                if self.has_target():
                    return True
                self.next_frame()
```

### 2.5 `check_f_break()` — QTE F 破防检测

```python
# CombatCheck.py L81-92
def check_f_break(self):
    if not self.can_break and not self._in_liberation \
       and time.time() - self.last_break_check_time > 1:
        self.last_break_check_time = time.time()
        if self.find_one(Labels.f_break_full, threshold=0.9):
            return True
        if self.find_one('f_break',
                          box=self.box_of_screen(0.2, 0.2, 0.75, 0.8)):
            if not self.is_pick_f():  # 排除拾取 F
                self.can_break = True
                return True
```

### 2.6 `check_count_down()` — 倒计时检测

```python
# CombatCheck.py L99-124
def check_count_down(self):
    # 区域: (1820, 266, 2100, 340) @3840x2160
    count_down_area = self.box_of_screen_scaled(3840, 2160, 1820, 266, 2100, 340)
    count_down = self.calculate_color_percentage(text_white_color, count_down_area)

    if self.has_count_down:
        if count_down < 0.03:
            numbers = self.ocr(box=count_down_area, match=re.compile(r'\d\d'))
            if not numbers:
                self.has_count_down = False
                return False
            return True
        return True
    else:
        if count_down > 0.03:  # 白色像素 > 3%
            numbers = self.ocr(box=count_down_area, match=re.compile(r'\d\d'))
            if numbers:
                self.has_count_down = True
        return self.has_count_down
```

## 3. 颜色常量全集

### 3.1 战斗检测颜色 (`CombatCheck.py`)

```python
# 目标锁定黄色 L349-353
target_enemy_color_yellow = {
    'r': (0x84, 0xAD), 'g': (0x84, 0xAF), 'b': (0x20, 0x6F)
}

# 敌方红色血条 L355-359
enemy_health_color_red = {
    'r': (174, 225), 'g': (55, 85), 'b': (55, 76)
}

# Boss 血条 L385-389
boss_health_color = {
    'r': (245, 255), 'g': (30, 185), 'b': (4, 75)
}

# Boss 血条区域 L379-383
boss_red_text_color = {
    'r': (200, 230), 'g': (70, 90), 'b': (60, 80)
}
```

### 3.2 协奏环元素颜色 (`BaseCombatTask.py` L956-987)

```python
con_colors = [
    # SPECTRO (金)
    {'r': (205,235), 'g': (190,222), 'b': (90,130)},
    # ELECTRIC (紫)
    {'r': (150,190), 'g': (95,140), 'b': (210,249)},
    # FIRE (红)
    {'r': (200,230), 'g': (100,130), 'b': (75,105)},
    # ICE (蓝)
    {'r': (60,95), 'g': (150,180), 'b': (210,245)},
    # WIND (绿)
    {'r': (70,110), 'g': (215,250), 'b': (155,190)},
    # HAVOC (暗紫)
    {'r': (190,220), 'g': (65,105), 'b': (145,175)}
]
```

### 3.3 其他颜色 (`BaseChar.py`)

```python
# 强击满检测
forte_white_color = {'r': (244,255), 'g': (246,255), 'b': (250,255)}

# CD 点数检测
dot_color = {'r': (195,255), 'g': (195,255), 'b': (195,255)}

# 文字白色
text_white_color = {'r': (244,255), 'g': (244,255), 'b': (244,255)}
```

## 4. 角色系统 (`BaseChar.py`)

### 4.1 角色类型

```python
class CharType(StrEnum):
    MAIN_DPS = 'MainDps'
    SUB_DPS = 'SubDps'
    HEALER = 'Healer'

class SwitchPriority(StrEnum):
    NORMAL = 'normal'
    MUST = 'must'
    NO = 'no'

class Elements(IntEnum):
    SPECTRO = 0; ELECTRIC = 1; FIRE = 2; ICE = 3; WIND = 4; HAVOC = 5
```

### 4.2 角色模板 (`do_perform()`)

```python
# BaseChar.py L266 - 默认模板
def do_perform(self):
    self.wait_intro(1.2)
    self.click_echo(time_out=0)
    self.click_liberation()
    if not self.click_resonance()[0]:
        self.heavy_click_forte(self.is_mouse_forte_full)
    self.switch_next_char()
```

### 4.3 共鸣技能释放

```python
# BaseChar.py L352
def click_resonance(self):
    while self.resonance_available():  # 技能可用则持续按键
        self.send_key(self.key_config.get('Resonance Key'))  # 默认 E
        self.sleep(0.1)
    # 支持 has_animation 模式，检测角色离队再回来
    # 超时: SKILL_TIME_OUT = 15s
```

### 4.4 协奏环检测 (`count_rings()`)

```python
# BaseCombatTask.py L848
def count_rings(self, image, color_range, min_area):
    # 1. 创建环形蒙版（内径 0.35119h, 外径 0.42261h）
    # 2. 应用元素颜色范围蒙版
    # 3. 形态学闭运算 (3x3 kernel)
    # 4. cv2.connectedComponentsWithStats 找连通区域
    # 5. is_full_ring(): 轮廓 → 多边形近似 → 凸性检测(>=4顶点)
    # 返回 (area, is_full)
```

## 5. 关键时序参数

| 参数 | 默认值 | 用途 |
|------|--------|------|
| `target_enemy_time_out` | 3s | 目标锁定超时 |
| `switch_char_time_out` | 5s | 角色切换超时 |
| `sleep_check_interval` | 0.4s | 战斗检测间隔 |
| `SKILL_TIME_OUT` | 15s | 技能执行超时 |
| `intro_motion_freeze_duration` | 0.9s | 入场动画冻结时间 |
| `cycle_time_out` | 1.1s | 角色循环超时 |
| `cycle_intro_time` | 1.2s | 入场循环时间 |
| Liberation animation | 7s | 大招动画超时 |
| Heavy attack | 0.6s | 重击持续时间 |

## 6. 角色切换优先级

```python
# BaseCombatTask.py L405
def _choose_switch_target(self):
    # 4 级优先级:
    # 1. MUST 目标 (来自 get_switch_priority() 返回 MUST)
    # 2. NO 目标跳过
    # 3. NORMAL 目标 (基于角色类型选择)
    #    - 当前是 MAIN_DPS → 切换到 buff 剩余时间最短的非主C
    #    - 有非主C 缺少 buff → 切换到最早缺 buff 的
    #    - 当前是 SUB_DPS/HEALER → 切换到主C (最早的)
    #    - 回退: 缺buff的奶 > 缺buff的副C > 主C > 最早的
```

## 7. 数据流

```
AutoCombatTask.run()
  → warm_up_char_features()       # 预加载所有角色模板
  → scene.in_team()               # 检测是否在队伍界面
  → while self.in_combat():       # CombatCheck.in_combat()
      → has_target()              # 模板匹配
      → check_health_bar()        # 颜色检测
      → load_chars()              # 进入战斗时加载角色
      → get_current_char().perform()  # 角色连招
          → do_perform()          # 角色特化逻辑
          → switch_next_char()    # 优先级切换
              → _choose_switch_target()
              → 发送数字键 (1/2/3)
              → wait for index change
              → switch_out() on outgoing char
      → combat_end()              # 战斗结束清理
      → switch_healer()           # 切奶回血
```
