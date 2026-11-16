# GeoWorld R8.1c — 河口接收水位与低岸统一（测试候选）

日期：2026-11-14。状态更新至 2026-11-16：规则 v1.1.23 已实现；2026-11-15 首轮游戏内视觉验收因严重地形破碎失败。现已以 RiverGeometryVersion 8 修复并完成 1.21.1 NeoForge 自动复验，**新的游戏内视觉复测与 post-v8 全 9 目标发布矩阵仍待完成，当前禁止发布**。

## 原始缺陷

R8.1b 以种子 -1818694639628404530 的实际 NeoForge 1.21.1 方块确认了河口 #509 的一格逆坡；正式 river #535/#851 各一格模型逆坡、#915 为四格。证据文件：docs/acceptance/r81b_rc4_water_and_sinkhole_evidence.json。

Waterbed 的 riverBankSurfaceCeiling 用实际低岸限制河面，但 WorldSample 一旦切换 OceanEligibility 便把海洋水面设置为统一海平面，未同时协调河海交接。仅提升水面会产生岸边水墙，不能作为解决方案。

## 修复设计与历史隔离

- 新世界规则 v1.1.23 启用持久化 WorldSample.CoastalBackwaterVersion=1；老世界配置缺省为 0，保存的 v1.1.22 及更早规则不变。
- 初始 R8.1c 水位修复阶段 RiverGeometryVersion 为 7；2026-11-15 视觉验收发现 v7 曲线河道最近点求解会造成地形破碎，因此新建 v1.1.23 世界现升级为 RiverGeometryVersion 8。已持久化 v7 与 v1.1.22 仍保留旧语义。
- Waterbed.Sample 只增加附近河道原始接收水位的元数据，保留已存在的低岸保护计算。
- 仅当河道有匹配海平面的正式 receiver，最终地面低于海平面、且河道实际湿区或沿岸已有宏观海水影响时，以海平面作为相连水体水位。
- 保留海水资格阈值边界：河口上游湿区继续标 River，真正海洋仍标 Ocean；仅对与接收河道相邻、海洋影响大于 0.02 的低岸补充自然海水。没有 receiver 的内陆低地不被强制淹没。
- Go/Java 使用同一持久化版本开关。v1.1.23 的其他地貌代码兼容 v1.1.22。

## 固定实测与阶段验收

原始基线模型：R72VoxelAudit/data/geoworld/world.geoworld；SHA256：
eb1ffcbc19ee43e55495f2007716b1ca0a907dcedf7b493303358d689e20512f。

模型级候选阶段记录的模型：.tmp/r8-hydrology/r81c-v23.geoworld；当时压缩文件 SHA256：
9024476e6ce6f59dea5196e600048575c63b792d7926050e1dfbb1d62b9d89b2。真实 NeoForge 方块验收使用的持久化 world.geoworld 压缩 SHA256 为 f3b7f470e585aec4950d52f5aec779061d521b1853359e364075c8a88aeeec1a；旧/新 Anvil 对照世界的该文件字节完全一致。

对 88 条正式入海河段、每段 129 点共 11,352 点的 Go 模型检查：
- v1.1.22 的四条逆升河段为 #509(+1)、#535(+1)、#851(+1)、#915(+4)。
- v1.1.23 为 0 条向下游逆升、0 个中心线断水点；采样地面高度变化为 0。
- 河段数 953、终点 103、特殊地貌 20 未变化；中心线 293 点水面/水体归属改变。
- 6 个控制区域共 1,014 个 Go/Java 采样点：新增的 108 个水体变化在两端完全相同，旧版本及新版本都存在相同的 16 个干地高度差异（最大归一化差 0.002784283）；不能宣称全量 parity 通过。
- Java Runtime javac 编译成功；NeoForge 1.21.1 compileJava 成功。
- R8.1c 单元测试检查新旧版本隔离、低岸候选资格与不误淹不相干内陆低地。

## 真实 Minecraft 方块验收（2026-11-14）

使用种子 -1818694639628404530、同一份 v1.1.23 持久化模型，在 NeoForge 1.21.1 中复制旧验收世界并删除目标 Region 后重新生成真实区块。旧世界保留为负向对照。

- #509 全区域扫描范围为 x[-1136,-1008)、z[-1104,-976)，共 16,384 列，缺失 0。旧世界相邻湿格 >1 格水墙为 50 对、最大水面差 2 格；新世界为 **0 对、最大水面差 0**。
- #915 全区域扫描范围为 x[1632,1712)、z[10400,10480)，共 6,400 列，缺失 0；相邻湿格 >1 格水墙为 **0 对、最大水面差 0**，Y61/Y62 连续水体检查仍通过。
- 最后一处 #509 水墙根因不是 OceanMouth 接收器半径不足，而是 14 个河口 Lake 重叠格在 WorldSample/Java Runtime 中先命中 Lake 所有权，保留了更高湖面。该阶段 v1.1.23 仅在正式 OceanMouth、低于海平面且存在沿岸海洋影响的版本化走廊内，让海平面接收器先于局部 Lake；当时 RiverGeometryVersion 仍为 7。后续视觉破碎修复见下节。
- 该最后优先级修复直接改变 14 个 Lake 重叠格，旧方块 Y64 统一到新方块海平面 Y62。旧验收世界由增强后的全区域脚本重新检查时会明确失败并报告 50 对水墙；新世界通过，因此验收脚本不再存在只检查中心线而漏掉内部湿格水墙的问题。
- Go 全仓 `go test ./...` 通过；NeoForge `compileJava` 通过；Gradle `test` 构建链通过（适配器当前无 Java test source）。

证据：`docs/acceptance/r81c_actual_mouth_voxels.json`、`docs/acceptance/r81c_actual_anvil_water_scan.json`、`docs/acceptance/r81c_receiver_model_candidate.json`。

## pre-v8 发布收敛历史（2026-11-15）

- 当时的发布矩阵 9/9 目标完成了 `clean check build`：26.3、26.2、26.1.2 的 NeoForge/Fabric，1.21.1 的 NeoForge/Fabric，以及 1.20.1 Fabric。
- 当时的 9 个候选包均使用 `-Pmod_version=0.1.6` 实际构建，Loader 元数据版本为 0.1.6，并验证包含 Windows/Linux/macOS × x64/arm64 六套 GeoWorld 原生引擎。
- **该矩阵已被 2026-11-15 的游戏内地形破碎失败作废为发布证据**；它只作为 pre-v8 历史记录保留。当前 `docs/acceptance/v016_release_readiness.json` 为 schema 2，明确标记 `automatedReleaseReadiness=revalidation_pending`、`manualVisualAcceptance=pending_retest`、`releaseAllowed=false`。
- 当前未创建 `v0.1.6` tag，也未发布 GitHub Release。

## 游戏内视觉失败与 RiverGeometryVersion 8 修复（2026-11-15～16）

2026-11-15 使用种子 `-1818694639628404530` 的新世界进行人工视觉验收时，#509 周边沙岸、河口大尺度区域及 #915 冻河区域出现明显薄墙、尖柱、机械断崖和一格宽地形裂缝，因此 **pre-v8 v0.1.6 候选被明确判定为视觉验收失败**。此前“水墙为 0”的自动指标不足以覆盖这种地表形态破坏。

根因通过相邻方块反查确认：

- v7 的曲线河道 `pointRiverSegment` 使用无保护 Newton 精化。#509 的相邻方块中，正确最近距离约为 71 格的点会错误跳到 `t=0`，算成约 189 格，从而让相邻一格在“河槽/非河槽”之间离散翻转。
- 修正最近点后，又暴露出固定最小水深阈值会把几何湿区边缘重新判为干地；v8 将可见水深阈值与 RiverWeight 一致化。
- 多河交汇处，旧逻辑会把“只以约 0.2% 权重擦入 bankRadius”的较高河段水面全量传给另一河段的干岸塑形；v8 改为按实际 bank/delta 权重连续混合。

同参数 v7→v8 固定种子复扫使用“相邻最终高度跳变 ≥4 格，且比原始地形相邻跳变至少多 3 格”作为人工断层判据：

| 区域 | v7 人工断层边 | v8 人工断层边 | v7 最大人工跳变 | v8 最大人工跳变 |
|---|---:|---:|---:|---:|
| `(-1047,-1076) r=72` | 618 | **0** | 14 | **0** |
| `(-1015,-988) r=96` | 625 | **0** | 11 | **0** |
| `(-1100,-1070) r=112` | 714 | **0** | 14 | **0** |
| `(1653,10405) r=112` | 342 | **0** | 6 | **0** |
| 合计 | **2299** | **0** | — | — |

兼容性复验中 v7 的四区原始计数完整恢复，确认新逻辑只由 RiverGeometryVersion 8 启用，不热改已持久化 v7 模型。

最终代码状态的 `go test ./...` 通过；Minecraft 1.21.1 NeoForge `clean check build -Pmod_version=0.1.6` 通过。WaterOwnership 的 `unownedSurfaceWaterSources`、`unsafeExposedWaterSources`、`freeFallWaterFaces`、`runtimeWaterLost`、`riverWaterBoundaryModelMismatches`、`riverLocalWaterPeaks`、`wetlandPoolLeaks` 均为 0。新的 1.21.1 NeoForge 候选 JAR SHA256 为 `6b5f56afd95fe605e65eafd4428d2f2593ba9c4f47cb4d2a62aac52f94881117`。

证据：`docs/acceptance/r81c_fragmentation_fix.json`。

## 当前剩余门槛（更新于 2026-11-16）

1. 必须使用新 v8 候选包**新建世界**重新做游戏内视觉验收；pre-v8 失败世界不能作为新候选验收结果。
2. 视觉通过后，对当前 v8 代码重新执行完整 9 目标发布矩阵。2026-11-15 的 9/9 结果仅为 pre-v8 历史证据，不再代表当前发布候选。
3. 上述两项通过前，`releaseAllowed=false`，不创建 `v0.1.6` tag，不发布 GitHub Release。

另外，天坑 #11 对河段 #735 的未规划排水冲突仍属于 R8.1d，岛屿及雪山盆地选址属于 R8.2。本轮不同时修改这些功能。
## v8 视觉复测被旧模型缓存污染（2026-11-16）

2026-11-16 的第二次游戏截图仍显示放射状/同心圆式沙墙。直接读取该新存档的 `data/geoworld/world.geoworld` 后确认：模型仍是 `RiverGeometryVersion=7`，并非 v8。存档模型 SHA256 `eb96a9168259d9a8f1ce6f9583e12834904bf63da2ea3bc578a0accab17b70b8` 与 eadm `geoworld-cache/446c3c5c195674ed8b8d565eceba8e10.geoworld` 完全一致；缓存创建时间为 2026-11-15 17:40，而该文件在 2026-11-16 18:18 被复制持久化到新世界。

根因位于 `GeoWorldEngineBridge` 的模型缓存身份：旧 key 包含 rules/profile/config，但没有 native engine 内容指纹；`validateModel()` 也未校验 `RiverGeometryVersion`。因为 v7 与 v8 候选仍都叫 `geoworld-rules-v1.1.23`，旧 v7 缓存被错误视为合法。

修复后缓存格式为 `v27`，key 额外包含期望 `RiverGeometryVersion=8` 和实际嵌入 native engine SHA256；加载缓存时再硬校验模型内部 geometry version。桥接 verifier 已真实执行首次 engine 生成与第二次缓存命中，两次均要求 geomV8；1.21.1 NeoForge 完整 `clean check build` 通过。

这意味着该截图**不构成 v8 视觉失败或通过证据**。后续人工视觉验收必须先确认实际存档 `world.geoworld` 为 geomV8，再观察地形。
