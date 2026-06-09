# MaaWuWaX 编码规范

## 核心原则

1. **Pipeline JSON 低代码优先** — 能用 Pipeline JSON 实现的逻辑，绝不写 Go 代码
2. **Go 只写 Custom 组件** — Go Service 仅实现 `CustomRecognition` 和 `CustomAction`
3. **禁止在 Agent 中写业务流程** — 业务流程由 Pipeline JSON 编排，Go 只提供原子操作

## Pipeline JSON 规范

- 节点命名: `模块名_动作名`（如 `Combat_ScanEnemy`, `Daily_ClaimMail`）
- 每个 JSON 文件控制在 200 行以内
- `roi` 使用 720p 基准坐标 `[x, y, w, h]`
- `threshold` 默认 0.7，特殊场景可放宽到 0.5
- 所有 `next` 列表必须覆盖所有可能的画面状态

## Go Service 规范

- 模块目录: `agent/go-service/<模块名>/`
- 每个模块含 `register.go`，在根 `register.go` 统一注册
- Custom Recognition/Action 类型名与注册名一致
- 使用 `zerolog` 记录日志，禁止 `fmt.Println`
- JSON 编解码使用 `sonic`（`sonic.Marshal` / `sonic.Unmarshal`）
- 从 Pipeline `attach` / `custom_recognition_param` / `custom_action_param` 读取参数
- 按键使用 `pkg/keycode` 包的 CGKeyCode 常量

## 图片资源规范

- 模板图片基准分辨率: 720p (1280×720)
- 文件命名: `<功能>_<描述>.png`（如 `char_jinhsi.png`, `btn_confirm.png`）
- 存放路径: `assets/resource/image/`
- 不使用游戏解包原始纹理，使用实际游戏截图裁切

## 项目结构

```
MaaWuWaX/
├── assets/
│   ├── interface.json          # PI V2 项目接口
│   ├── resource/
│   │   ├── image/              # 模板图片
│   │   └── pipeline/           # Pipeline JSON
│   ├── tasks/                  # Task 定义 JSON
│   └── locales/                # 国际化（后续添加）
├── agent/go-service/           # Go Service
├── maafw/                      # MaaFramework 动态库
└── debug/                      # 日志输出
```
