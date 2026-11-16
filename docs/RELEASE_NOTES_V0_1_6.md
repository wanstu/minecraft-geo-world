# GeoWorld v0.1.6 — 发布候选说明

状态：**2026-11-15 首轮游戏内视觉验收失败；R8.1c 地形破碎根因已修复并完成自动复验，新的 1.21.1 NeoForge 候选包待人工视觉复测。当前禁止正式发布。**

## 当前修复

v0.1.6 仍使用 `geoworld-rules-v1.1.23`。首轮候选虽然通过河口水位与水墙自动验收，但实际游戏截图暴露了严重的薄墙、尖柱和一格垂直断层。根因不是水面本身，而是曲线河段最近点 Newton 求解在相邻方块间跳到错误端点，并叠加了河流交汇处高水位保护的硬开关。

新建 v1.1.23 世界现在使用 **RiverGeometryVersion 8**：

- 曲线河段最近点改为全段粗采样选取正确极小值区域，再用只允许距离下降的 safeguarded Newton 精化。
- 河流湿边缘的最小可见水深与几何权重统一，避免“几何上属于河道、WorldSample 又把它判成干地”。
- 多河交汇时，相邻较高河段的岸线保护按其实际 bank/delta 影响权重衰减，不再刚擦到保护范围就 100% 抬升另一条河岸。
- 已持久化的 RiverGeometryVersion 7 继续走旧算法；v1.1.22 仍固定为 v7，不热改旧世界。

固定种子 `-1818694639628404530` 对用户截图四个问题区域做 v7→v8 同参数模型复扫：`618→0`、`625→0`、`714→0`、`342→0`，合计 **2299→0**。同时旧 v7 基线计数完整恢复，证明不是靠改变旧模型语义做掉指标。

## 自动复验

- `go test ./...`：通过。
- Minecraft 1.21.1 NeoForge `clean check build -Pmod_version=0.1.6`：通过，`BUILD SUCCESSFUL`。
- Model Verifier：34/34。
- WaterOwnership：`unownedSurfaceWaterSources=0`、`unsafeExposedWaterSources=0`、`freeFallWaterFaces=0`、`runtimeWaterLost=0`、`riverWaterBoundaryModelMismatches=0`、`riverLocalWaterPeaks=0`、`wetlandPoolLeaks=0`。
- Java 适配器的 `test` 任务为 `NO-SOURCE`，这里只声明构建/验证链通过，不声明存在 Java 单元测试。
- 新 1.21.1 NeoForge JAR：`29,164,469` 字节，SHA256 `6b5f56afd95fe605e65eafd4428d2f2593ba9c4f47cb4d2a62aac52f94881117`；Loader 元数据为 0.1.6，并包含六套 native engine。

2026-11-15 视觉失败的旧候选 SHA256 `eae4a66ab738504505f8e69bc38b2f71b5ade4c8846c56501f40e19a527f72c4` 已废弃，不能继续用于验收。

## 发布状态

v8 修改发生在此前 9 目标候选矩阵验收之后，因此此前的 9/9 结果只保留为 **pre-v8 历史证据**，不能代表当前代码。当前只重新验证了 1.21.1 NeoForge。

下一步先用新的候选 JAR、新建世界重新做游戏内视觉验收。视觉通过后，再对当前 v8 代码完整重跑 9 目标发布矩阵并生成最终候选包；在这两项都完成前：

**不创建 `v0.1.6` tag，不发布 GitHub Release。**
