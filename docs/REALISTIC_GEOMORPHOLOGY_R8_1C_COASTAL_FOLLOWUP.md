# GeoWorld R8.1c 收尾复验：#535、#851 河海接缝与 #509 自然基底

**日期：** 2026-11-11  
**代码基线：** `60c8286`（规则 `geoworld-rules-v1.1.23`，仅隔离候选）  
**状态：** 4 个历史模型逆坡河口完成针对性的 Minecraft 方块复验；R8.1c **仍不能认定所有河口已通过视觉验收，不合并、不发布**。

机器可读原始证据：`docs/acceptance/r81c_coastal_followup_evidence.json`。与上轮证据 `docs/acceptance/r81c_v23_actual_voxel_evidence.json` 配套，不替代旧结论。

## 1. 固定证据和隔离条件

- Minecraft 1.21.1 / NeoForge 21.1.211，种子 `-1818694639628404530`。
- 旧 v1.1.22 保存世界：`/home/wanstu/projects/minecraft-geo-world/java/adapters/1.21.1-neoforge/run/saves/R81VoxelAudit`。
- 新 v1.1.23 保存世界：`java/adapters/1.21.1-neoforge/run/R81cVerifiedGeoWorld`（本隔离工作树内）。
- 两个世界的 `level.dat`、GeoWorld 绑定、精确 seed、模型及已保存的 `minecraft:full` 区块通过预检；旧模型**没有被新版模型覆盖或迁移**。
- 使用 `cmd/geoworld-r81-mouth-points` 从**新世界真实保存的模型**导出正式 river #509/#535/#851/#915 的精确曲线及各 129 个参数点；只检查明确知道已经生成的 Anvil 方块，未知不计通过。
- 主验收脚本：`scripts/r81c_coastal_followup.py`。

## 2. #535、#851：实际方块河海接缝通过

两个河口在原 v1.1.22 **模型**里曾出现 1 格逆坡；旧世界这两个地点没有可用的旧版 Anvil 实际方块，不能假装有新旧方块直接对照。

按保存模型中的真正河道中心线，在 `t=0.625..0.6875` 的河海交界窗口逐个检查：

| 检查 | #535 | #851 |
|---|---:|---:|
| 交界窗口已生成方块列 | 9/9 | 9/9 |
| 交界窗口共用水面 Y62 | 9/9 | 9/9 |
| 沿模型曲线延伸已生成列 | 49/49 | 51/51 |
| 延伸段共用水面 Y62 | 49/49 | 51/51 |
| 5 个横断面已生成列 | 28 | 37 |
| 横断面低岸与高水面直接贴邻信号 | 0 | 0 |

注：横断面外缘分别还有 17、8 个未生成样本，**不计通过**；采样是稀疏横断面，不等于周边所有岸坡均已检查。符合的结论只能是“旧模型报告的这两处河海逆坡，在新版已生成的相关河道中心线实际水方块中没有重现”。

## 3. #509：区分 Minecraft OCEAN_FLOOR 与自然基底

将 `x=-1120..-1010`、`z=-1052..-998` 的 **6,105** 个同坐标列与 v1.1.22 世界对照，全部都是已生成的实际 `minecraft:full` 区块。

### 3.1 高度图不是自然河床

`OCEAN_FLOOR` 会将珊瑚、树叶等计入地表高度，原始高度图的 602 列差异中包括如下情况：

- 旧列水下原本 Y61 是珊瑚，真实砂床在 Y57；新世界珊瑚消失、变为水，`OCEAN_FLOOR` 记录会降低 4 格，但自然砂床本身仍在 Y57。
- 旧列有金合欢树叶在 Y67，而自然土层在 Y61；新版水下树叶消失，这不等于河床被削低 6 格。
- 河口水面抬升之后，仍有真实砂床、沙砾及其他表面地层被 Java 岸线规则抬升/重塑的情况。

因此新脚本对每列自上而下识别第一层泥土、沙、砾石、草方块、石头、雪等已知**自然表层材料**，过滤流体、珊瑚、叶片、海草及空气。这个判定用于当前河口验收区域；不能直接推广到任意模组、任意地形方块。

### 3.2 自然基底真实差异

| 同坐标自然表层最高固体块 Y 变化 | 列数 |
|---|---:|
| 未变 | 5,529 |
| 抬高 +1 | 511 |
| 抬高 +2 | 15 |
| 降低 -1 | 50 |
| 合计 | 6,105 |

**共 576 列自然基底确实改变，最大只到 +2/-1 格。** 不是“大面积 6 格削低”，但也**不能说修复纯粹只改水、不改岸线地形**。这些是最终 Minecraft 方块层面的变化，不可用宏观模型地面不变来抹掉。

### 3.3 实现归因，保留一个特殊点

- 旧/新持久化模型的 22 个代表坐标，Go 的 `groundY` **22/22 一致**。
- 对同样 22 点分别使用 Java `GeoWorldShorelinePlanner.planModel` 和 Minecraft 真实自然表层方块比较：**21/22 的高度变化相符**，说明多数改变可以归因到 Java 的水岸整形，而非 WorldModel 地形基础数据变化。
- 另一坐标 `(-1010,-998)`：Java 岸线规划的 groundBoundary 不变，实际新世界多了 Y63 的一层沙，旧版顶层为 Y62 的砂岩；这是一处**岸边表层材料变化**，不是已证明的宏观地形或侵蚀算法错误。保留它作为后续地表材料/覆盖规则的专门回归点。
- 对 6,105 列范围的所有相邻高度图列检测：高度差至少 6 格的相邻对旧版有 7，新版为 0；地形边界不因此自动判定视觉自然，仍需游戏中检查。

### 3.4 不能据此断言全部完成

R8.1c 主要修复正式海水 receiver 与低岸接缝的水位一致性。新水域归属会改变 Java 岸线邻域，从而改变部分真实土石层位置；这是已实际观察到的副作用。总体坡度、沉积合理性、河道外形是否逼真，仍需在真正的游戏视角验收，不用模型数值代替。

## 4. 构建、测试和下一阶段界限

- 只新增只读检查命令、只读 Python 验收工具、测试和证据文档。**没有修改生成规则 v1.1.23 的地形/水文算法，也没有触碰老世界**。
- Python 自动化会检查：冻结水面与冰山不能误判、未生成区块不得计通过、珊瑚/树叶不得算自然河床、真实中心线接缝 9 个样本必须全部存在。
- 旧版 #535/#851 并无实际方块可直接比较；四处固定河口的最新实际方块结果不能外推成“88 条入海河段全部方块验收”。
- 这次关闭 R8.1c 固定河口的逆坡专项。**下一阶段 R8.1d 只处理天坑 #11 和河段 #735 未建立排水通路的问题**；雪山盆地、岛屿形成机理归入 R8.2。R8.1c 仍须根据正式 RC 清单做游戏视角检查及必要的 Loader / 版本构建之后才能发布。

## 5. 重复执行

在仓库根目录：

```bash
go run ./cmd/geoworld-r81-mouth-points \
  java/adapters/1.21.1-neoforge/run/R81cVerifiedGeoWorld/data/geoworld/world.geoworld \
  .tmp/r81c-official-mouth-paths.json

python3 -m unittest discover -s scripts -p 'test_r81c_*py' -v

python3 scripts/r81c_coastal_followup.py \
  --legacy-world /home/wanstu/projects/minecraft-geo-world/java/adapters/1.21.1-neoforge/run/saves/R81VoxelAudit \
  --candidate-world java/adapters/1.21.1-neoforge/run/R81cVerifiedGeoWorld \
  --centerlines .tmp/r81c-official-mouth-paths.json \
  --old-model-audit .tmp/r81c-floor-v22-model.json \
  --new-model-audit .tmp/r81c-floor-v23-model.json \
  --java-shoreline-csv .tmp/r81c-shore-compare/shores.csv \
  --output docs/acceptance/r81c_coastal_followup_evidence.json
```

选填的 `.tmp/r81c-floor-*` 模型报告和 `.tmp/r81c-shore-compare/shores.csv` 来自本轮只读 Go/Java 临时验证；现已将逐点数据和摘要固化在证据 JSON 中。缺少选填项时，主脚本仍能运行真实方块、河海接缝和自然基底比对。
