# MaaWuWaX 架构回正与重构总结

本次重构根据“Pipeline JSON 低代码优先”的核心原则，对 MaaWuWaX 的遗留模块进行了全面的架构回正。我们深度梳理了 Go Service 中的硬编码调用逻辑，将其迁移到 Pipeline 原生的状态流转中。

## 1. Combat (战斗) 模块重构

- **移除 Go Action 中的流转控制**: `agent/go-service/combat/combat.go` 中原有的 `ctx.RunAction("Combat_Dodge")` 等硬编码调用已被全面移除。
- **引入状态驱动机制**: 引入了 `CombatCheckPending` Custom Recognition。现在通过 Go 设置全局状态变量 (`pendingAction`)，Pipeline JSON (`Combat/Main.json`) 会主动轮询该状态，进而决定是否进入闪避、换人、拾取等流程。这种“状态驱动”的方式保证了逻辑完全由 JSON 编排，Go 仅提供数据支撑。

## 2. Navigation (导航) 与 EchoFarm (声骸) 模块重构

- **清理冗余 Go 逻辑**: 彻底删除了不再使用的巨大单体行为 `TeleportBossAction` 及其繁杂的辅助函数（超过 120 行的僵尸代码被移除）。
- **优化 Realm Enter 逻辑**: 在 `EchoFarm` 模块中，原有的 `EchoFarmSelectRealmLevel` Go Action 中存在冗余的 `ctx.RunRecognition` 调用。现已将其彻底废弃，并在 `assets/resource/pipeline/EchoFarm/RealmEnter.json` 中重构为基于原生 `OCR` 的识别节点，实现了副本等级选择纯 JSON 表达。
- **AutoPick 简化**: 移除了针对拾取流程的冗余前置状态校验依赖，并修复了合并过程中引发的代码结构冲突。

## 3. Pipeline JSON 超大文件拆分

经扫描，项目内所有 Pipeline JSON 现已成功控制在 **200 行以内**（目前最长为 `NightmareNest/Combat.json`，共计 199 行）。原先过于臃肿的模块已被合理拆解（例如 `NightmareNest` 拆分为了 `Main.json`, `DailyModes.json`, `Combat.json` 等独立文件），极大地提升了项目的可读性和可维护性。

## 4. 验证与测试

已执行 `go test ./...` 完成全局构建校验，确保所有调整后的 Go 代码编译通过，无类型缺失及引入未使用的包错误，各自定义行动/识别节点均能正确注册。
