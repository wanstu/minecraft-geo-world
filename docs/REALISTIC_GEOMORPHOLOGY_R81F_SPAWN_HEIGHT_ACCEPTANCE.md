# GeoWorld R8.1f — 1.21.1 NeoForge 出生点高度修复专项

日期：2026-11-12  
分支：`adm/geoworld-r81f-spawn-height-20261112`  
来源：R8.1e 已通过的独立候选提交 `34daa95`  
适用：Minecraft 1.21.1 NeoForge；生成规则 `geoworld-rules-v1.1.25`（**未升级规则版本**）。

**状态：出生点高度故障已复现，修复后通过 Minecraft 服务端事件及真实已保存世界专项复验；首次创建全新世界、客户端视觉、正式九目标 CI 仍未完成；不发布。**

## 1. 根因范围：不修改出生点规划器

已保存的 GeoWorld 真世界 `R81eSeed123456VoxelAudit`：

| 项目 | 原有存档值 | 同坐标真实方块 |
|---|---:|---:|
| X / Z | `1088 / 448` | `1088 / 448` |
| 出生点 Y（`level.dat`） | **-64** | - |
| 自然土石表面 | - | **Y=106** |
| 第一个可站立的地面上方 Y | - | **Y=107** |
| 世界生成器、规则 | `geoworld:geoworld` / v1.1.25 | 绑定正确 |

`GeoWorldSpawnPlanner.find` 在本种子两个出生点权重下完整遍历耗时
约 1.24 秒 / 0.81 秒，不能解释首次创建世界 535 秒的延迟。
没有修改出生点规划器、候选选择、种子、评分或任何天坑/河流水文算法。

1.21.1 的 `GeoWorldWorldService.ensureSafeSpawn` 在 `ServerStartedEvent` 回调中，将候选的 `chunkX*16/chunkZ*16` 设为出生点 X/Z，但用 `overworld.getHeight(MOTION_BLOCKING_NO_LEAVES, x,z)` 求 Y。
在原始存档中，该调用的实际结果最终表现为 **-64**，与真实方块高度相差 171 格；考虑到目标区块高度图尚未就绪时可能返回底部高度，**不能再依赖该调用设置新世界出生点**。

这里可以确认高度值错误与该代码路径存在，不声称已从 Minecraft 内部源码证明为何高度图返回 -64。

## 2. 修复方法（严格限制范围）

新增：

`java/minecraft-common/src/main/java/geoworld/minecraft/common/worldgen/GeoWorldSpawnHeight.java`

唯一职责：通过已绑定的不可变 `GeoWorldV1` 模型，读取准确的出生点 X/Z 地表归一化高度，再使用当前 Minecraft 版本已有的 `HeightMapping.boundaryY` 映射到游戏 Y；非有限高度直接拒绝，不把错误映射成最底层。

在：

`java/minecraft-v1_21_1-common/src/main/java/geoworld/minecraft/v1_21_1common/runtime/GeoWorldWorldService.java`

将计算出生点 Y 的 `overworld.getHeight(...)` 替换为上述固定模型计算。

**不改任何出生点水平坐标、不改出生点候选排序、不添加新生物群系、不改 Go 世界模型或当前生成规则、不迁移旧存档。**
`GeoWorldWorldService` 只在已经绑定的动态 GeoWorld 世界和原逻辑允许更换出生点时设置该位置；其他场景保持原样。

## 3. Java 模型级核对

独立编译并执行：

`java/minecraft-common/src/test/java/geoworld/minecraft/common/worldgen/GeoWorldSpawnHeightVerifier.java`

输入为**真实 NeoForge 保存的 v1.1.25 模型**，精确 seed=123456。原始出生点规划结果的 X/Z 必须为 `1088,448`，原模型地表为 `0.585156080473`（归一化）：

```text
PASS seed=123456 spawnX=1088 spawnY=107 spawnZ=448 ground=0.585156080473 candidateScore=0.046703619
```

这证明 Java Runtime 模型映射 Y=107、X/Z 未更换。

## 4. 真实服务端隔离复验

验证不是只靠 Java 数字：在新工作树内**复制**旧版 v1.1.25 GeoWorld 存档到
`java/adapters/1.21.1-neoforge/run/R81fSpawnReplay`。仅在该**测试副本**的
`level.dat` 临时将出生点重置为 `(0,63,0)`，模拟首次需要重新选择出生点的 ServerStartedEvent 路径。

启动已包含补丁的 **Minecraft 1.21.1 NeoForge 21.1.211** 服务端，正确检测现有 GeoWorld 绑定，成功触发服务器启动事件，`save-all flush` 并正常关闭；再只读检查复制世界的 `level.dat` 和 Anvil 方块：

| | 旧存档（只读） | 修复后的测试副本 |
|---|---:|---:|
| 出生点 X | 1088 | **1088** |
| 出生点 Y | **-64** | **107** |
| 出生点 Z | 448 | **448** |
| 真方块自然地面 | Y106 | **Y106** |
| 规则/种子/生成器 | v1.1.25 / 123456 / GeoWorld | 全部相同 |

旧存档与测试副本的 `data/geoworld/world.geoworld` **SHA256 完全一致**，没有重新计算或迁移模型。原始世界未写入，仍保存旧异常值，符合禁止历史迁移要求。

证据：`docs/acceptance/r81f_spawn_height_replay.json`；自动化冻结证据：`scripts/test_r81f_spawn_height_replay.py`。

## 5. 编译与范围限制

- 1.21.1 NeoForge `compileJava` **通过**，真实 ServerStartedEvent、保存退出通过。
- 26.3 Fabric `compileJava` **通过**，说明通用 Java 辅助类没有破坏该 Loader 源码编译。但该版本并未引入新出生点处理逻辑；不能据此断言其他目标游戏内行为已验证。
- 独立 Java `GeoWorldSpawnHeightVerifier` 验证归一化模型高度与 X/Z 不变。
- 此次没有修改 Go 代码；完整 Go 回归及 9 个正式 CI 目标的 `clean check build` 仍需在正式验收中完成。
- 当前专项只完成**已有世界复制并重新触发服务器出生点设置**。尚未通过“从空文件夹首次创建新世界”的实际启动/保存验收，**不能把本专项说成整套新世界创建验收完成**。
- 旧世界出生点仍是历史保存值，后续是否需修复须另立迁移需求，绝不隐式改写。
- 初次创建世界耗时约 535 秒、复用世界约 24 秒的差异尚未定位到最终根因，性能问题**未修复**。64,000 候选的出生点规划已单独实测不足 2 秒，不应盲目改算法。

## 6. 待完成发行门槛

全新 v1.1.25 世界真实创建并检查 SpawnY（最好再验一个非相同种子）；真实 Minecraft 客户端出生点落点/碰撞视觉验收；九个正式 Loader/游戏版本 CI；首次世界启动耗时独立性能分析。

**本阶段无推送、无主干合并、无 Jar 发布，不改旧存档、不动地貌算法。**



## 2026-11-12 更新：新世界首次创建验收已经通过

之前文档第 5、6 节中的“尚未完成全新世界首次创建”系编写当时状态。现已在**两个独立空目录新世界**完成首次创建、`save-all flush`、正常 `stop`，逐一读取其持久化 `level.dat` 和真实 `minecraft:full` Anvil 区块。

- **种子 123456（创建时已有模型缓存）**：`Done (16.678s)`，Spawn=(1088,107,448)，地面 Y106。
- **种子 20261112（创建时无对应模型缓存）**：`Done (45.427s)`，Spawn=(192,104,192)，地面 Y103。
- 两种情况的种子、世界生成器 `geoworld:geoworld`、绑定规则 v1.1.25 均正确；出生点 Y 相比真实自然地面高 1 格，没有落到 -64。
- 完整验收文档：`docs/REALISTIC_GEOMORPHOLOGY_R81F_FRESH_WORLD_FIRST_BOOT.md`。
- 证据：`docs/acceptance/r81f_fresh_world_first_boot.json`；回归脚本：`scripts/test_r81f_fresh_world_first_boot.py`。

**更新后的结论：R8.1f 出生点高度专项通过实际全新世界首次创建验收。** 之前 535 秒的偶发启动耗时未复现，也未找到最终根因，不能说性能问题已彻底修复；客户端游戏画面和九目标正式 CI 仍没有完成，因此 RC 仍 HOLD。没有修改任何历史存档。
