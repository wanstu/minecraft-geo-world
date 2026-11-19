# GeoWorld R8.1d — 天坑与正式河网选址验收

日期：2026-11-17
目标版本：v0.1.7
新世界规则：`geoworld-rules-v1.1.24`
RiverGeometryVersion：8

## 1. 问题

固定种子 `-1818694639628404530` 中，v0.1.6 / v1.1.23 的天坑 #11 位于约 `(-1600, 5312)`，与正式河段 #735 的活动走廊相交。天坑没有持久化排水计划，但会在河的一侧切掉原本正常的岸坡。

Slice A 使用真实 Minecraft 1.21.1 NeoForge dedicated-server 存档，对 32 个固定坐标读取 Anvil section/block state，确认这不是模型推测：

- 32/32 样本均为 `full` Chunk，0 缺失；
- 河 #735 四条横断面中心水面均为 Y=96；
- 天坑侧 12/12 岸点地面低于同截面河面 11～27 格；
- 天坑中心自然地面 Y=69；
- 命名岸点 `(-1665,5313)` 自然地面 Y=69；
- 对侧正常河岸均高于河面 4～9 格。

因此责任层确定为 **SpecialFeatures 的天坑选址**，不是 RiverGeometry v8、河床或水位。

## 2. Slice B 修复

v1.1.24 只改变新世界的 sinkhole candidate siting：

1. 候选天坑先通过原有 deterministic spacing；
2. 计算天坑主体及 lobe 的实际最大影响范围；
3. 对正式 `Ground.Rivers` 做 broad-phase；
4. 沿真实曲线河道中心线按不大于约 12 格间距采样；
5. 使用河道实际半宽 + 岸带安全余量做 clearance；
6. 冲突候选被拒绝，并保留其占位禁区；
7. 继续 deterministic candidate 序列，寻找下一个安全天坑。

没有修改：

- 河网拓扑；
- RiverSegment；
- 河床；
- 水面；
- RiverGeometryVersion；
- R8.1c 河口 / CoastalBackwater；
- 旧 v1.1.23 持久化模型。

新规则仍为 RiverGeometryVersion 8，CoastalBackwaterVersion 1。

## 3. 固定种子模型结果

使用 Slice A 的实际 geomV8 模型解析参数，仅把 rules 从 v1.1.23 切换到 v1.1.24 后重新生成：

- 河段：953 → 953；
- RiverSegment 数组：完全相等；
- 特殊地貌总数：20 → 20；
- 天坑数：4 → 4；
- 原冲突天坑 `(-1600,5312)` 被拒绝；
- 其余三个原天坑保持候选序列；
- deterministic 补选天坑：约 `(320,6592)`；
- `sinkholeRiverOverlapSignals`：1 → **0**；
- `sinkholeOverlapWithoutDrainage`：1 → **0**。

旧问题点在模型中恢复为自然地形，不再属于特殊地貌 footprint。

## 4. 实际 Minecraft voxel 对照

同一 seed、同一 32 点横断面，在真实 v1.1.24 / geomV8 NeoForge dedicated-server 世界中重新生成 182 个目标 Chunk；Anvil header 确认 182/182 写入，随后正常停服保存。

结果：

| 指标 | v1.1.23 基线 | v1.1.24 |
|---|---:|---:|
| full 样本 | 32/32 | 32/32 |
| 右岸低于河面的样本 | 12/12 | **0/12** |
| 最大“河面高于地面” | 27 格 | **-4 格** |
| 天坑中心自然地面 | Y69 | **Y103** |
| `(-1665,5313)` 自然地面 | Y69 | **Y102** |
| 河 #735 命名点水面 | Y96 | **Y96** |
| 河末端命名点水面 | Y96 | **Y96** |

四条河中心线截面的水面仍全部是 Y=96。也就是说，修复没有靠降低河流、抬高河床或改变水位通过验收，而是恢复了原本不应被天坑切掉的自然岸坡。

## 5. 旧世界兼容性

使用当前 v0.1.7 代码直接启动 Slice A 的持久化 v1.1.23 世界：

- Runtime 仍绑定 `geoworld-rules-v1.1.23`；
- 启动前模型 SHA256：
  `239b1b22703843e7ce8351a633258974c59781b0897f3ef3e7c3fa499758544d`
- 正常保存退出后 SHA256 完全相同。

因此 v1.1.24 只影响新世界，不迁移、不重选、不重算 v1.1.23 世界。

## 6. 自动回归

当前 Slice B 代码状态：

- `go test ./...`：通过；
- `geoworld-tester`：通过；
- generator 全量测试：通过；
- R8.1d voxel audit Python 自测：3/3；
- Java↔Go Engine Bridge：rules=v1.1.24、geomV8、cache-v27 通过；
- Minecraft 1.21.1 NeoForge `clean check build -Pmod_version=0.1.7`：**BUILD SUCCESSFUL**；
- Model Verifier：34/34；
- WaterOwnership 关键异常全部为 0；
- Java `test` 任务为 `NO-SOURCE`，因此只声明构建/验证链通过；
- 1.21.1 NeoForge JAR 元数据版本 0.1.7，六套 native engine 校验通过。

## 7. 结论与下一步

**R8.1d Slice A / B 已完成。**

结论：当特殊地貌与权威河网发生无规划排水冲突时，优先在候选选址阶段拒绝 / 重选特殊地貌，而不是修改已有河流来迁就它。

这套结果只关闭 R8.1d，不代表 v0.1.7 可发布。下一步进入 R8.2：

1. Ocean / Lake island 的水下根部与 moat 选址约束；
2. Snow basin / crater 的山地 relief、高程、温度与排水背景；
3. 先做只读取证，再做最小候选筛选；
4. v0.1.6 河口与 geomV8 固定回归点继续作为不可退化门槛。

证据文件：

- `docs/acceptance/v017_r81d_geomv8_model_baseline.json`
- `docs/acceptance/v017_r81d_geomv8_actual_voxels.json`
- `docs/acceptance/v017_r81d_geomv124_model_candidate.json`
- `docs/acceptance/v017_r81d_geomv124_actual_voxels.json`
- `docs/acceptance/v017_r81d_slice_b_summary.json`
