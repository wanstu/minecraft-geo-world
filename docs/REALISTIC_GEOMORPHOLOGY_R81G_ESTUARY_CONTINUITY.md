# GeoWorld R8.1g — v1.1.26 河流入海口水面连续性专项验收

日期：2026-11-12（dadm）  
工作树：`adm/geoworld-r81f-spawn-height-20261112`（R8.1g 变更尚未发布）  
生成规则候选：`geoworld-rules-v1.1.26`  
对照旧规则：`geoworld-rules-v1.1.25`  
Minecraft：1.21.1 NeoForge 21.1.211

**结论：固定种子的入海口相邻河面断层已修复，并通过 Go/Java 数值、全域 Java 规划器水文检查、独立新 Minecraft 世界的真实保存方块专项验收。整个 v0.1.5 RC 的其他 Loader/正式 Windows CI/客户端画面验收仍未完成，不发布。**

## 1. 原始缺陷和验收约束

旧版 `1.21.1 NeoForge clean check build` 的
`verifyGeoWorldWaterOwnership` 报错：1183 区块、302848 方块列的采样中有
**77 个相邻水面高差超过 1 格**。
证据：`docs/acceptance/r815_rc_neoforge1211_water_ownership_failure.json`。

复现坐标位于 **种子 123456**，相邻两列均属于同一条正式河流：

| 坐标 | v1.1.25 持久化 Runtime 水面（归一化） | v1.1.26 候选 |
|---|---:|---:|
| (10814, 2467) | 0.500000000000（强制海面） | 0.505909501491 |
| (10815, 2467) | 0.506182656291 | 0.506182656291 |

旧版 Go 与 Java 会对两侧分别进行“不分段的海面压低”与“保持河道原有自由水面”处理，产生 Y63 与 Y66 水面边界差 3 格。
这不是 R8.1e 天坑改动导致的位移，已记录的 v1.1.24、v1.1.25 存档使用
`CoastalBackwaterVersion=1`。

**本轮绝不：** 调低 `freeFallWaterFaces==0` 的验收要求；改动河网、地形宏观分区、天坑选址；迁移用户 v1.1.25 和更早存档；取消现有海洋/湖泊/湿地所有权；将未生成区块当成已经验收的地形。

## 2. 新规则的水面算法（仅 v1.1.26）

将 `CoastalBackwaterVersion` 定义为：

- `0`：旧版无海洋回水语义。
- `1`：R8.1c 海洋回水，**保持旧持久化规则原样**。
- `2`：R8.1g 新水面连续性语义，**仅新生成 v1.1.26 世界使用**。

原版 `1` 在具备海洋接收点、河床低于海面时直接返回海面高度。
新版 `2` 在同样位置额外检查**已有河水表面**：
若这列仍有有效的正式河流水面，且高于海面 **半个 Minecraft 方块的归一化高度**，则暂不截断原河面，维持天然河流纵剖面。河面真正进入接近海平面的范围后才交给海洋回水处理。

这里的**半格阈值**来自 Minecraft 整数方块水面取整：原先以一个完整方块作为切换阈值时，采样连续浮点表面在阈值两侧仍可跳出 2 格的真实水面阶梯；经局部河道网格对照后收敛到半格切换，才同时消除原 3 格落差及其他 2 格内河断面。内河既有水位、湖泊、湿地与海洋所有权仍由现有逻辑处理。

修改落实在同构 Go/Java 两端：

- Go `internal/worldsample/worldsample.go` / `model.go`；
- Java `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldV1.java`；
- Go 默认规则及模型持久化与运行时支持 v1.1.26；
- Minecraft Java 引擎、表层、水文、矿物/特殊地貌、生成规则验证器显式识别 v1.1.26，并沿用 v1.1.25 的其他生成语义。

其他地貌算法没有改动。

## 3. 完整水文验收（不是只检查两个点）

基于独立生成的 seed=123456 v1.1.26 世界模型，
在 Minecraft Java 最终方块列规划器中验证：

| 断言 | 修复前 v1.1.25 | 修复后 v1.1.26 |
|---|---:|---:|
| 检查区块数 | 1183 | 1183 |
| 检查方块列 | 302848 | 302848 |
| 海岸/河/湖/湿地水域归属不明 | 0 | 0 |
| 不安全裸露水面 | 0 | 0 |
| **相邻水面自由落差超过 1 格** | **77 FAIL** | **0 PASS** |
| 运行时河面数据与规划高度不符 | 0 | 0 |
| 河流局部孤峰 | 0 | 0 |
| 湿地漏水 | 0 | 0 |

原 `verifyGeoWorldWaterOwnership` 任务返回 **BUILD SUCCESSFUL in 59s**，未放宽任何失败条件。

同时在原缺陷附近的 144×144 方块列规划区域，对照统计：
**相邻 River↔River 超过 1 格水面落差从 78 处降为 0 处。**
该局部 78 与完整扫描 77 的数量不同，因为前者包含区域内跨区块边缘，不等同于完整验证器的内部采样范围。

## 4. Go 与 Java、旧世界兼容

使用**两个真实持久化的 v1.1.25 / v1.1.26 世界模型**对河口及附近多组坐标作测试：

- 旧 v1.1.25：**141/141** Go/Java 点高度与水域类型、定义状态、水面一致，原来的 v1 水位逻辑未变。
- 新 v1.1.26：**141/141** Go/Java 点一致。
- 归一化表面/水面在两端保留的 12 位小数精度内误差均为 **0**。
- 两模型的宏观 `surface`、`terrain`、完整 `waterbed` 河网及 `specialFeatures` **完全一致**：河段 **955**、特殊地貌 **18**。
- 用户此前旧测试存档的 `SurfaceOrePile=false`，此次标准新建世界 `SurfaceOrePile=true`，矿物资源分配也有所差异。**不能假称全部模型 payload 只有规则版本不同**；这些差异来自创建设置/矿物初始状态，不影响这次河流水面逻辑对照。

证据：`docs/acceptance/r81g_go_java_parity_seed123456.json` 和
`docs/acceptance/r81g_model_and_runtime_acceptance.json`。

## 5. 实际 Minecraft Anvil 方块验收

从**空目录**新建独立 `R81gEstuarySeed123456`，seed=123456、
世界类型 `geoworld:geoworld`、规则 v1.1.26；
Minecraft 1.21.1 NeoForge 启动 `Done (28.381s)`，随后通过本地 RCON 生成并保存原缺陷区域区块；清除强制加载后 `save-all flush`、正常 `stop`。

只读解析真实保存的 `region/*.mca` 方块状态，在
X=10804..10840、Z=2465..2505 的矩形范围检查：

| 原生 Minecraft Anvil 验收项 | 实际值 |
|---|---:|
| 确定完整方块列（`minecraft:full`） | **1517 / 1517** |
| 相邻的真实水面方块对 | **2762** |
| 水面相邻落差超过 1 格 | **0** |
| 水向较低干地暴露信号 | **0** |
| 原缺陷 (10814,2467) / (10815,2467) 的实际水面顶格 | **Y65 / Y65** |
| 所有必需采样点缺失 | **0** |

Minecraft 的“水面方块顶格 Y65”与 Java 规划器水位边界 Y66 的表示方式差 1；实际方块顶格由 `minecraft:water` 等真实方块状态解码得到，而不是把规划高度填进验收表。两端均表现为**相邻河水连续**。

证据：`docs/acceptance/r81g_seed123456_estuary_actual_anvil.json`。
工具：`scripts/r81g_estuary_anvil_audit.py`，明确拒绝未生成/未完成的区块。

**没有实际 Minecraft 客户端相机截图，不能把该矩形当作整张世界或地貌美观的视觉验收。**

## 6. 自动测试、版本边界和剩余发布门槛

新增 Go 单测 `TestR81gCoastalRiverTransitionVersioned` 锁定旧回水语义、新半格切换与拒绝非海洋关联河道漫灌。

新增 `TestR81gV126KeepsV125WorldConfigExceptCoastalHandoff` 锁定除了 `WorldSample.CoastalBackwaterVersion` 和规则版本外，**同一创建设置**的全部宏观默认参数不应被调整。

Python `scripts/test_r81g_estuary_acceptance.py` 对 141×2 的 Go/Java 数值、完整水文验证器及 1517 个真实 Minecraft 方块列进行固定证据测试。

### 本轮已实际完成的回归

- **1.21.1 NeoForge `clean check build` PASS：** `BUILD SUCCESSFUL in 5m 15s`，45 个 Gradle 可执行任务（44 个执行、1 个使用缓存）。其中完整 `verifyGeoWorldWaterOwnership` 最终 `freeFallWaterFaces=0`，其它水域指标均为 0。
- **NeoForge 本地 JAR 验证 PASS：** `geoworld-0.1.4.jar`（开发分支模块元数据而非发布），SHA256 已保存到 `docs/acceptance/r81g_neoforge1211_jar_inspection.json`；六套 Windows/Linux/macOS x64/ARM64 原生引擎及必要 Loader 元数据均齐全。
- **Go 7 组相关包完整测试 PASS：** `internal/generator`、`internal/worldsample`、`internal/specialfeatures`、`internal/runtimeworld`、`internal/worldmodelio`、`internal/waterbed`、`cmd/geoworld-r81-hydrology-audit`，未采用 `-run '^$'` 的只编译代替。
- **26.3 Fabric v1.1.26 源码编译 PASS：** `compileJava BUILD SUCCESSFUL in 23s`，不等于该 Loader 完整构建通过。
- **严格的版本区分：** v1.1.26 候选当前仅 **1/9 个目标本地完整构建 PASS**（1.21.1 NeoForge）；之前 1.21.1 Fabric/26.3 Fabric 的两次完整构建属于 v1.1.25，**必须重新跑 v1.1.26 才能计入本版通过**。

机器可检查的本地九目标状态表：`docs/acceptance/r81g_current_rules_local_matrix.json`。

仍未完成：其他 Loader/版本的九目标官方 GitHub Windows CI，真实 Minecraft 客户端游戏视角，其他种子/地貌气候的河口覆盖。旧世界不做自动迁移或修复；v1.1.25 仍以旧规则加载。

**当前仍不合并、不推送、不发布任何 JAR，不赋予整版 v0.1.5 RC 发布资格。**


## R8.1h（2026-11-12）后续构建复验

此报告中“1/9 完整构建通过”的数据是当时的阶段快照。后续已在**同一个 v1.1.26 规则源码提交 `fa49b91`**上，分别为 1.21.1 Fabric、26.3 Fabric 完整执行 `clean check build` 并通过（17/17、41/41），并验证两个 JAR 含有编译后的 v1.1.26 引擎身份常量。

**当前本地完整构建通过 3/9**，而不是 1/9。其它 6 个目标尚未确认（26.3 NeoForge 资源阻塞 1；5 个未完整测试）。GitHub Windows 九目标 CI / Minecraft 客户端视觉验收仍未完成；发布资格仍为 HOLD。

参见 `docs/REALISTIC_GEOMORPHOLOGY_R81H_FABRIC_FULL_BUILD_20261112.md` 和 `docs/acceptance/r81h_current_rules_local_matrix.json`。
