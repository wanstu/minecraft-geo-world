# GeoWorld R8.1e — 天坑连续坡面与真实方块阶段验收

**规则候选：** `geoworld-rules-v1.1.25`（在 v1.1.24 已通过的河道避让基础上）  
**工作分支：** `adm/geoworld-r81e-sinkhole-geomorph-20261111`  
**固定种子：** `-1818694639628404530`，Minecraft 1.21.1 NeoForge 21.1.211  
**状态：** R8.1e 固定天坑坡面 **模型 / Java-Go 一致性 / 真实方块专项通过**；仍不等于全部地貌自然度通过，不批准发行。

## 1. 缺陷和限定修改范围

R8.1d 修好了天坑 #11 与河段 #735 相交、造成河道悬高的选址冲突，但新位置 `(320,6592)` 的实际 Minecraft 区块出现了明显不自然的几何形态：

- 大面积真实自然基底停留在同一个 Y=77（东西约 160 格、南北约 128 格）。
- 在很窄的外围边缘上，坡率从 1.875 格/格到 2.0 格/格。
- 这是 v1.1.24 与更早沉陷公式造成的“固定 inner 半径坑底 + 窄 outer 平滑壁”，不是新选址引入的错误。

本阶段**仅变更新规则下的 Sinkhole 地表剖面**。其他特殊地貌、河道、水位、历史存档及主世界边界都不改。

## 2. 实现

- **旧版版本隔离：** v1.1.24 及更早版本继续使用旧代码分支，生成语义完全不变；只在 `v1.1.25` 启用新坡面。
- **连续沉陷：** 将天坑从中心到外缘的地面过渡改为 `smoothstepRange(0.035, 0.98, radial)`，替代过去把约 56% 半径固定在坑底后再突然抬升。
- **局部细微起伏：** 使用确定性的低幅值 `sin/cos` 组合，最大变化量约为几个小数千分点的归一化地面高度，中心为 0、外缘衰减为 0；不生成无关联的大噪声锯齿，也不抬高原始天然地面。
- **保留原排水边界：** 地形采样仍先检查 `WaterKind`；水域不进入天坑下切，原有正式河道及 R8.1d 选址避让不变。
- **Go/Java 同构：** `internal/specialfeatures/model.go` 与 `java/runtime-reader/src/main/java/geoworld/runtime/GeoWorldV1.java` 的坡面公式一致。
- **新版本读取：** Go 生成器、序列化、运行时和 Java Loader 规则版本检查新增 v1.1.25，同时完整保留旧版本枚举。这是生成规则版本，**并未发布新的 mod 版本号**。

## 3. 两个真正的 Minecraft 世界的持久化模型对比

v1.1.24 和 v1.1.25 均使用 Minecraft 1.21.1 NeoForge 服务端的新世界实际保存的 `data/geoworld/world.geoworld`，严格校验 GeoWorld worldgen 类型、同一完整 `int64` seed 和规则绑定。

**持久化 `payload` 唯一变化的字段组是 `config.WorldProfile`。** 下列数据完全相同：

- 正式河网：**953** 个 river segment。
- 全部 20 个特殊地貌保存项（包括 #11 新位置 `320,6592`），没有一个再次换址。
- 地形基础、生态、矿物和水文等完整模型 payload。
- 正式海口：88 条模型检查中 `riverOceanBoundaryRiseSignals=0`。
- 河流与天坑：`sinkholeRiverOverlapSignals=0`，`sinkholeOverlapWithoutDrainage=0`。

机器可验收的摘要：`docs/acceptance/r81e_v125_sinkhole_acceptance.json`。离线 Go 重新生成模型与服务端实时生成的矿物环境可能存在差异，因此以上“只有规则配置变化”结论**只使用两份真正由服务端保存的模型**作对比，而非混用离线矿物数据。

## 4. Java 与 Go 真实保存模型数值一致性

从两个版本各取新天坑周围的 **41 个精确坐标**：

| 世界模型 | Go / Java 对照 | 归一化地面高度差（保留 12 位小数） |
|---|---:|---:|
| v1.1.24 | 41 / 41 | 0 |
| v1.1.25 | 41 / 41 | 0 |

说明新 Java Runtime 同时正确理解旧世界和新世界；没有出现“Go 模型对、Minecraft Java 方块解释错”的分歧。

## 5. Minecraft 真实 Anvil 地貌对比

新世界：`java/adapters/1.21.1-neoforge/run/R81eVerifiedGeoWorld`。通过正式 GeoWorld 生成器创建、按坐标生成区块、`save-all flush` 并正常关闭服务端，再只读解析保存的 `region/*.mca` 方块状态。

| 项目 | v1.1.24 | v1.1.25 |
|---|---:|---:|
| 东西剖面实际完整采样 | 22/22 | 22/22 |
| 南北剖面实际完整采样 | 18/18 | 18/18 |
| 坑底中心自然基底 | Y=77 | Y=77 |
| 东西自然外围高程 | Y115 / Y116 | Y115 / Y116 |
| 南北自然外围高程 | Y115 / Y116 | Y115 / Y116 |
| 东西剖面最大采样坡率 | 1.875 | **0.5** |
| 南北剖面最大采样坡率 | 2.0 | **0.5** |
| 坑底连续大面积同一 Y77 | 存在 | **消除** |
| 中心地表水 | 未检出 | 未检出 |

同坐标比较中：东西 22 个采样点有 18 处实际自然基底高度变化；南北 18 个采样点有 13 处变化。变化集中在坑内及坡脚，中心和外缘不动。

主要证据：

- `docs/acceptance/r81d_new_sinkhole_actual_voxels.json`（旧规则真实方块）
- `docs/acceptance/r81e_new_sinkhole_actual_voxels.json`（新规则真实方块）
- `docs/acceptance/r81e_v125_sinkhole_acceptance.json`（跨版本同坐标比对及模型约束）
- `docs/acceptance/r81e_v124_vs_v125_real_anvil_comparison.png`（同坐标真实方块俯视、东西/南北剖面示意）

**图像是 Anvil 已保存方块的诊断投影，不是 Minecraft 游戏内截图，也不是卫星照片。** 所有未知区块显示灰色，不能把其余外缘推断为已验收的自然地貌。

## 6. 自动化与发布门槛

- Go：`TestR81eSinkholeBroadSlopeReplacesFlatCenterOnlyForV125` 和 `TestR81eOldSinkholeVersionsRemainFlatAndOtherFeatureKindsUnchanged` 直接校验新坡度、旧版封存、坑外不增高、其他地貌不变化。
- Go：`TestR81eV125KeepsV124HydrologyAndMacroProfileDefaults` 检查默认水文、海口、沉积、水文回归配置未变。
- Python：`scripts/test_r81e_sinkhole_voxel_audit.py` 校验两世界绑定、40 个真实 Anvil 样本、完整数值证据、Go/Java 对齐和不允许提前宣布发布。
- NeoForge 1.21.1 Java：`compileJava` 通过，服务端真实创建、保存、关闭 v1.1.25 GeoWorld 世界。
- Fabric 26.3 Java：`compileJava` 通过（Gradle 9.7.1、Fabric Loom 1.18.4，JDK25），仍未执行本轮的完整 `check build` 和真实客户端验收。
- 完整相关 Go 测试：`internal/generator`（约 192 秒）、`internal/specialfeatures`、`internal/runtimeworld`、`internal/worldmodelio`、`internal/worldsample`、`internal/waterbed`、`cmd/geoworld-r81-hydrology-audit` 全部通过。
- Go 全项目编译：`go test -p 2 -run '^$' ./...` 通过，但此命令不执行所有测试用例。
- Python 固定证据回归：`python3 -m unittest discover -s scripts -p 'test_r81*.py' -q`，**29 项通过**。

- 本阶段不修改官方 v0.1.4、v0.1.5 RC 正式发行元数据，不推送、不发布 JAR、不覆盖任何旧世界。

**仍待验收：** 客户端游戏内实景的坡面、纹理、地貌尺度和区域地质合理性；其他代表种子 / 坑型 / 气候区的稳定性；正式 9 目标 CI 全构建。以上任何一项未达标都不应标记“整个 GeoWorld 地貌真实度验收通过”。

## 7. 可重复的检查命令

```bash
go test -p 2 -count=1 ./internal/generator ./internal/specialfeatures \
  ./internal/runtimeworld ./internal/worldmodelio ./internal/worldsample \
  ./internal/waterbed ./cmd/geoworld-r81-hydrology-audit

python3 -m unittest discover -s scripts -p 'test_r81e*.py' -v

python3 scripts/r81e_new_sinkhole_voxel_audit.py \
  --world java/adapters/1.21.1-neoforge/run/R81eVerifiedGeoWorld \
  --output docs/acceptance/r81e_new_sinkhole_actual_voxels.json \
  --image docs/acceptance/r81e_new_sinkhole_anvil_planview.png
```
