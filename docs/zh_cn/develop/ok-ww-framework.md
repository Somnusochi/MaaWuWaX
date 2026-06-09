# ok-ww 框架层完整实现

> Labels(246)、config、WWScene、YOLO、模板匹配、坐标系统

## 1. Labels.py — 246 个模板标签

所有模板名称的字符串枚举，按类别：

**角色头像 (~60):** `char_verina`, `char_jinhsi`, `char_camellya`, `char_shorekeeper`, `char_carlotta`, `char_phoebe`, `char_zani`, `char_brant`, `char_cantarella`, `char_roccia`, `char_lupa`, `char_cartethyia`, `char_changli2`, ...

**战斗 UI (~22):** `has_target`, `no_target`, `has_target_169`, `no_target_169`, `has_target_cloud`, `no_target_cloud`, `box_char_1/2/3`, `char_1/2/3_text`, `box_resonance`, `box_echo`, `box_liberation`, `box_extra_action`, `e_forte`, `mouse_forte`, `edge_levitator`, `f_break`, `f_break_full`

**协奏/元素 (~24):** `con_spectro/electric/fire/ice/wind/havoc`, `con_full_*`, `lib_ready_*`, `lib_mark_char_1/2/3`

**书本 UI (~13):** `book_canxiang/mengyan/moni/qiangdi/wuyin/zhange`, `gray_book_boss/quest/all_monsters/weekly/boss_highlight`

**地图/导航 (~22):** `map_way_point/big/pop_up_box`, `big_map_diamond/skull/star`, `world_earth_icon`, `arrow`, `boss_check_mark/minimap`, `boss_no_check_mark`, `treasure_icon`, `purple_target_distance_icon`, `on_the_wall`

**对话框 (~19):** `skip_dialog/new`, `skip_dialog_confirm`, `skip_dialog_check`, `skip_quest_confirm/new`, `btn_auto_play_dialog`, `btn_dialog_3dots/arrow/close/eye`, `message`, `message_dialog`

**Echo (~8):** `echo_dropped/not_dropped`, `echo_locked/not_locked`, `echo_enhance_btn/confirm/success/to`, `echo_merge`, `merge_echo_check`, `absorb_en_US/zh_CN`

**按钮 (~14):** `cancel_button_hcenter_vcenter`, `confirm_btn_hcenter_vcenter`, `confirm_btn_highlight_hcenter_vcenter`, `claim_cancel_button_hcenter_vcenter`, `gray_button_challenge`, `gray_confirm_exit_button`, `gray_start_battle`, `gray_crownless_battle`, `gray_teleport`, `custom_teleport_hcenter_vcenter`, `fast_travel_custom`, `remove_custom`, `pick_up_f_hcenter_vcenter`, `revive_confirm_hcenter_vcenter`

## 2. config.py — 项目配置

### 模板匹配配置

```python
'template_matching': {
    'coco_feature_json': 'assets/coco_annotations.json',  # COCO 标注文件
    'default_horizontal_variance': 0.002,   # 水平容忍度
    'default_vertical_variance': 0.002,     # 垂直容忍度
    'default_threshold': 0.8,               # 默认置信度阈值
    'feature_processor': process_feature,   # 自定义预处理回调
    'vcenter_features': ['monthly_card'],
    'hcenter_features': ['monthly_card'],
}
```

### OCR 配置

```python
'ocr': {
    'lib': 'onnxocr',
    'auto_simplify': True,
    'params': {
        'use_openvino': True,  # 也控制 YOLO 后端
        'use_npu': True,       # OpenVINO NPU 加速
    }
}
```

### 窗口配置 (Windows)

```python
'windows': {
    'exe': 'Client-Win64-Shipping.exe',
    'hwnd_class': 'UnrealWindow',
    'interaction': 'PostMessage',
    'capture_method': ['WGC', 'BitBlt_RenderFull'],
}
```

### 分辨率

```python
'supported_resolution': {
    'ratio': '16:9',
    'resize_to': [(2560,1440), (1920,1080), (1600,900), (1280,720)],
    'min_size': (1280, 720),
}
```

### 按键配置

```python
key_config_option = ConfigOption('Game Hotkey', default={
    'Echo Key': 'q',
    'Liberation Key': 'r',
    'Resonance Key': 'e',
    'Tool Key': 't',
    'Jump Key': 'space',
    'Dodge Key': 'lshift',
    'Wheel Key': 'tab',
})
```

### 任务注册

```python
# 一次性任务 (12个)
'onetime_tasks': [
    DailyTask, MultiAccountDailyTask, FarmEchoTask, AutoRogueTask,
    ForgeryTask, NightmareNestTask, SimulationTask, TacetTask,
    EnhanceEchoTask, ChangeEchoTask,
]

# 触发器任务 (6个, 后台持续运行)
'trigger_tasks': [
    AutoCombatTask, AutoPickTask, SkipDialogTask,
    AutoLoginTask, MouseResetTask, FastTravelTask,
]
```

## 3. WWScene.py — 场景状态缓存

```python
class WWScene(BaseScene):
    def __init__(self):
        self._in_team = None           # 惰性缓存
        self._echo_enhance_btn = None  # 惰性缓存
        self._in_combat = None         # 非惰性,实时跟踪
        self.cd_refreshed = False

    def reset(self):
        # 重置所有缓存, 在状态可能变化时调用
        self._in_team = None
        self._echo_enhance_btn = None
        self._in_combat = None
        self.cd_refreshed = False

    def in_team(self, fun):
        # 惰性求值: 仅在 None 时调用 fun(), 然后缓存
        if self._in_team is None:
            self._in_team = fun()
        return self._in_team

    def set_in_combat(self):   # 设置 _in_combat = True, 返回 True
    def set_not_in_combat(self):  # 设置 _in_combat = False, 返回 False
```

## 4. globals.py — 全局单例

```python
class Globals(QObject):
    _yolo_model = None  # 惰性加载

    @property
    def yolo_model(self):
        # 首次访问时加载 assets/echo_model/echo.onnx
        # use_openvino 控制后端: OpenVINO(NPU优先) 或 ONNX Runtime(DirectML优先)
        # 标签: {0: 'echo'}
```

## 5. OnnxYolo8Detect / OpenVinoYolo8Detect

```python
class OnnxYolo8Detect:
    def __init__(self, weights='echo.onnx', model_h=640, model_w=640, iou_thres=0.45):
        self.dic_labels = {0: 'echo'}  # 唯一检测类别

    def detect(self, image, threshold=0.5, label=-1):
        # letterbox → normalize → ONNX推理 → NMS → Box列表

class OpenVinoYolo8Detect:
    # 同上, 但使用 OpenVINO 推理
    # 设备选择: NPU (PERFORMANCE_HINT=LATENCY) → CPU 回退
```

## 6. process_feature.py — 模板预处理

```python
def process_feature(feature_name, feature):
    # illusive_realm_exit: convert_bw (244-255 → 白, 其他 → 黑)
    # purple_target_distance_icon: binarize_for_matching (threshold 244)
    # world_earth_icon: convert_bw
    # skip_dialog: convert_dialog_icon (210-244 → 白)
    # mouse_forte: binarize_for_matching
```

## 7. 坐标系统

```python
# 所有操作使用归一化坐标 (0.0-1.0)
def box_of_screen(self, x1, y1, x2, y2, hcenter=False):
    # 归一化 → 像素

def box_of_screen_scaled(self, ref_w, ref_h, x1, y1, x2, y2):
    # 从基准分辨率 (通常 3840x2160) 缩放到实际分辨率

def width_of_screen(self, ratio):   # ratio × 屏幕宽度 = 像素
def height_of_screen(self, ratio):  # ratio × 屏幕高度 = 像素
```

## 8. 图像预处理函数

```python
def convert_bw(cv_image):
    # cv2.inRange(image, (244,244,244), (255,255,255)) → 白, 其他 → 黑

def convert_dialog_icon(cv_image):
    # cv2.inRange(image, (210,210,210), (244,244,244)) → 白, 其他 → 黑

def isolate_white_text_to_black(cv_image):
    # 近白像素 → 黑, 其他 → 白 (OCR 准备)

def binarize_for_matching(image):
    # 灰度 → cv2.threshold(gray, 244, 255, THRESH_BINARY)
```

## 9. CharFactory — 角色工厂

```python
_char_dict_raw = {
    Labels.char_verina: {
        'cls': Verina, 'char_type': CharType.HEALER, 'ring_index': Elements.SPECTRO
    },
    Labels.char_jinhsi: {
        'cls': Jinhsi, 'char_type': CharType.MAIN_DPS, 'ring_index': Elements.SPECTRO
    },
    Labels.char_camellya: {
        'cls': Camellya, 'char_type': CharType.MAIN_DPS, 'ring_index': Elements.HAVOC
    },
    # ...47 个角色
}

def get_char_by_pos(self, box_char, index, old_char=None):
    # 1. old_char 置信度 >92% → 快速检查 0.6
    # 2. 类型变更 → 新建实例
    # 3. find_best_match_in_box(char_names, threshold=0.6)
    # 4. 回退: 保留 old_char 或 BaseChar
```
