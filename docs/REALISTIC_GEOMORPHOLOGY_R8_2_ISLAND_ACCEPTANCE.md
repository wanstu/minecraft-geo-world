# GeoWorld R8.2 Slice C — Island siting 与水下根部验收

日期：2026-11-17
目标版本：v0.1.7
新世界规则：`geoworld-rules-v1.1.25`
RiverGeometryVersion：8

## 1. 目标

Slice C 只处理特殊岛屿的地理背景与水下根部连续性：

- Ocean island 必须位于已有水下局部高地；
- 外围必须保持连续 Ocean moat；
- Lake island 没有真实湖底高点时允许不生成；
- 岛体必须从海床连续抬升，不能在平海床上直接插入岛柱；
- 不修改 RiverGeometry、Waterbed、海面、湖面、R8.1d 天坑规则或 Snow siting。

## 2. v1.1.24 基线

固定种子 `-1818694639628404530`：

- Ocean island：8 个，但选中点中心相对外围海床只有约 -1～1 格高差；
- Lake island：2 个，root lift 分别为 -2 / 0；
- 4 个足够大的湖组 root lift 为 `0, -2, -5, -12`，没有合格湖底山头；
- Ocean interior 候选池 31,843 个，其中：
  - root lift ≥4、depth≤48、384 格环带 32/32 Ocean：760；
  - root lift ≥8：281；
  - root lift ≥12：11。

因此旧岛屿选址失败不是候选不足。

## 3. v1.1.25 选址

Ocean island 候选资格：

- 384 格环带 32 点全部为 Ocean；
- 中心原始海床比环带海床中位值高至少 8 格；
- 中心水深不超过 48 格。

Lake island 候选资格：

- 160 格环带 32 点全部为同一 Lake；
- 中心湖底比环带湖底中位值高至少 4 格；
- 中心水深不超过 24 格。

固定 seed 结果：

- RiverSegment：953 → 953，完全不变；
- 非岛 SpecialFeatures：10/10 逐项 DeepEqual；
- Ocean island：8 → 8；
- Lake island：2 → **0**；
- Snow / sinkhole / crater / basin 不随岛屿重选发生位移。

8 个 Ocean island 的实际资格值全部通过，root lift 约 8.08～10.29 格，中心水深约 17.93～35.43 格，外围全部 32/32 Ocean。

为避免“Lake island 被拒绝后改变后续地貌 deterministic spacing”，v1.1.25 保留 v1.1.24 岛屿候选的 shadow reservation，仅用于后续非岛地貌的 ID / spacing 隔离，不把这些旧岛实际生成出来。

## 4. 水下根部几何

仅选对水下高地还不够。第一版 v1.1.25 沿用旧 Apply 公式：

`setLand(water + 0.008 + 0.052 * influence)`

实际 Minecraft 剖面显示，岛边界存在海床 Y31 → 岛岸 Y63 的约 32 格/4 格水下柱壁。

最终 v1.1.25 改为：

`target = lerp(baseGround, oldIslandPeak, influence)`

其中 `influence` 本身已经是平滑椭圆 / lobe influence：

- influence 很小时仅抬升海床，仍保持 Ocean/Lake；
- target 低于水面时只更新 GroundHeight / WaterDepth；
- target 穿过水面后才转为陆地并应用岛屿生态；
- influence=1 时保持旧岛心目标高度；
- v1.1.24 及更早规则继续使用旧公式。

## 5. 实际 Minecraft Anvil 验收

代表岛：

- kind：`mushroom_island`
- center：`(16448,-2880)`
- raw submerged root lift：约 9.07 格；
- raw center depth：约 29.08 格。

使用真实 Minecraft 1.21.1 NeoForge dedicated-server 世界，在 z=-2880 强制生成 x=15936..16960 的 65 个 Chunk；65/65 Anvil location entry 已写入，服务器正常保存退出。

正式细剖面：

- 两侧共 74 个采样，74/74 `full`，0 缺失；
- 排除 giant mushroom 等结构后，仅统计 sand / mycelium / stone 等天然基底；
- 原旧公式最大水下跳升：约 **32 格 / 4 格**；
- 新公式两侧天然基底最大变化：**4 格 / 4 格**；
- 水下 shoulder 大多数为 1～2 格 / 4 格；
- 左岸实际：Y31 → 33 → 34 → 35 → 36 → 38 → 40 → … → 59 → 61 → 63；
- 穿出水面后继续 Y63 → 67 → 70 → 73…，没有单格薄墙或悬空柱。

模型 / Runtime / Anvil 三层一致。

## 6. 兼容边界

- `v1.1.25` 才启用新 island siting 与 submerged shoulder；
- `v1.1.24` 及更早世界保持旧岛屿解释；
- RiverGeometryVersion 仍为 8；
- CoastalBackwaterVersion 仍为 1；
- Engine cache 仍通过 native engine SHA256 防止旧 runtime 结果被复用；
- 不迁移或重选已持久化 v1.1.24 世界。

## 7. 结论

**R8.2 Slice C 通过模型 + 实际 voxel 验收。**

固定 seed 已从“平海床硬贴岛 + 无条件 Lake island”收敛为：

- Ocean island 只落在真实水下高地；
- 无合格湖底高点时 Lake island=0；
- 岛体拥有连续 submerged shoulder；
- 河网与全部非岛特殊地貌保持隔离不变。

这不代表 v0.1.7 可发布。下一步进入 Slice D：Snow basin / crater 的 mountain + relief + climate + drainage siting。
