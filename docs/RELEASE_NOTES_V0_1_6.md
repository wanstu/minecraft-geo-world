# GeoWorld v0.1.6 — 发布候选说明

状态：**v0.1.6 已于 2026-11-16 正式发布。Tag `v0.1.6` 指向验收提交 `8e0daf0`；GitHub Actions Core + 9/9 release-build + Publish 全部成功，Release 包含 9 个 JAR 与 `SHA256SUMS.txt`。**

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
- 地形破碎修复阶段、缓存 v27 修复前的 1.21.1 NeoForge 候选 JAR：`29,164,469` 字节，SHA256 `6b5f56afd95fe605e65eafd4428d2f2593ba9c4f47cb4d2a62aac52f94881117`；该 SHA 仅作为阶段历史证据，不是当前发布矩阵产物。

2026-11-15 视觉失败的旧候选 SHA256 `eae4a66ab738504505f8e69bc38b2f71b5ade4c8846c56501f40e19a527f72c4` 已废弃，不能继续用于验收。

## 发布状态

2026-11-16 使用种子 `-1818694639628404530` 的实际新存档已读取确认：`generationRules=geoworld-rules-v1.1.23`、`RiverGeometryVersion=8`；存档 `world.geoworld` SHA256 为 `5b8eb7c9aa823d9a6f15a6ee510d6c387e5eadb1deee01de4bc6803bee280469`，并与本轮新缓存完全一致。

人工视觉复测通过 `(-1608,-1082)` 以及四个回归区域 `(-1047,-1076)`、`(-1015,-988)`、`(-1100,-1070)`、`(1653,10405)`。此前的放射状沙墙、薄墙、尖柱和单格断层未再出现。

随后对当前 v8/cache-v27 代码完整重跑 9 目标发布矩阵，**9/9 全部 `clean check build` 成功**；每个候选 JAR 均验证版本为 0.1.6，并包含 Windows/Linux/macOS × x64/arm64 六套 native engine。

当前发布状态：**已发布**。`v0.1.6` tag 指向 `8e0daf0`，GitHub Actions 发布工作流成功，正式 Release 已生成。
## 2026-11-16 缓存复用事故

当天第二次视觉截图仍出现严重放射状沙墙，但该截图后来被证明**不是 RiverGeometryVersion 8 的结果**。新建存档中的 `world.geoworld` 为 `RiverGeometryVersion=7`，SHA256 为 `eb96a9168259d9a8f1ce6f9583e12834904bf63da2ea3bc578a0accab17b70b8`；eadm 的 `geoworld-cache/446c3c5c195674ed8b8d565eceba8e10.geoworld` 与之字节级 SHA 相同，且缓存文件创建于前一天。

根因是旧缓存 key 虽包含 rules/profile/config，却没有绑定实际 native engine 内容，也没有在加载时校验 `RiverGeometryVersion`。由于 pre-v8 与 v8 都仍使用 `geoworld-rules-v1.1.23`，新 JAR 可以命中旧 v7 模型。

现已修复：缓存格式升级到 `v27`，cache key 加入 `expected RiverGeometryVersion=8` 与嵌入 native engine 的 SHA256；缓存加载再次硬校验模型内部 `riverGeometryVersion()`。`GeoWorldEngineBridgeVerifier` 已实际验证“当前 engine 首次生成 geomV8 → 第二次命中当前缓存仍为 geomV8”，随后 1.21.1 NeoForge 完整 `clean check build` 通过。

当时用于实际视觉复测的 1.21.1 NeoForge 候选 JAR SHA256：`f186dabaae5a20fd70a3791937e89f603555e41c4f1428afec7ab22610b94b24`。后续 post-v8 发布矩阵从同一代码状态重新打包，JAR SHA 另行记录在 `docs/acceptance/v016_release_readiness.json`；最终发布产物仍由 CI 从 tag commit 重建。

因此 2026-11-16 这张异常截图只能作为**旧 v7 缓存污染的证据**，不能作为 v8 视觉验收结果。下一次验收开始前，必须先从实际新存档中读取并确认 `RiverGeometryVersion=8`。
