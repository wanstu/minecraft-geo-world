# GeoWorld R8.1d — 天坑 #11 / 正式河道 #735 冲突修复验收

日期：2026-11-11  
工作分支：`adm/geoworld-r81d-sinkhole-20261111`  
候选规则：`geoworld-rules-v1.1.24`（独立新世界，**不替换 v1.1.23、v0.1.4 或其他旧世界**）  
环境：dadm / Minecraft 1.21.1 / NeoForge 21.1.211，世界种子：`-1818694639628404530`（保留完整 int64 精度）。  
**结论：固定的 #11 天坑与 #735 河流的真实 Minecraft 方块缺陷已修复并通过局部专项验收；尚不满足整个 RC 的发布门槛。**

## 1. 原始缺陷不是模型数字误报

基于旧 `geoworld-rules-v1.1.23` 的真实 Minecraft Anvil 保存区块，`docs/acceptance/r81d_actual_voxel_baseline_v123.json` 记录：

- 特殊地貌 #11：sinkhole，中心 `(-1600,5312)`，椭圆半径约 `153.6 × 130.5` 方块。
- 正式河段 #735：从 `(-1600,5440)` 到 `(-1728,5312)`。
- 河道水面多处实际 Y96（不是模型估计），旁边 16 格处干岸 Y79，32/48 格处坑底 Y69；原天坑中心自然基底 Y69。
- 天坑被规划到了已有河道上；原有河流水位和河床被保留，但两侧干岸仍被下切，产生高悬河道与无排水规划的坑底。
- 旧固定基线共 32 个采样点，31 已生成，1 未生成。未知点没有被记作“干地”。

## 2. 修复范围与正确语义

新世界规则版本增加到 `v1.1.24`。只有新规则采用本方案：

1. **保留完整正式河网。** 在已经完成 `Waterbed` 的候选天坑选址阶段，使用河段的真实弯曲中心线、河宽和 24 格以上缓冲判断天坑是否与河流冲突。
2. 对冲突天坑，仅重选其位置；原来已使用的选址仍占用其他特殊地貌的间距，防止后续火山坑、盆地因回填空位而一起发生非预期移动。
3. 保留冲突天坑原有 ID，替代位置重新计算新地点的基准高度、坑底与坑缘。无合适候选则宁可不放置该天坑，不能继续切断原河道。
4. 不改变自然河网、河面水位、河床纵坡、水流方向、入海判定；不凭空添加支流或地下排水。
5. `v1.1.23` 及所有历史规则继续按原来的 `SpecialFeatures` 逻辑运行；旧存档的特殊地貌、世界模型不迁移。

相关实现：`internal/specialfeatures/model.go` 的 `sinkholeConflictsRiver` 和 `usesRiverSafeSinkholeSites`；`internal/generator/profile.go` 及 Go/Java Runtime 的新规则别名兼容。

## 3. 新旧真实游戏模型对照

分别从**真正由 Minecraft 服务端保存**的两个绑定模型比较各层 payload：

| 项目 | 结果 |
|---|---|
| 正式河段 | 953，`waterbed` 持久化数据完全一致 |
| 入海河段模型逆坡信号 | 新规则下 88 条中为 0 |
| 特殊地貌 | 原有 20 个，新规则仍为 20 个 |
| 变化的特殊地貌 ID | **仅 #11** |
| #11 旧位置 | `(-1600,5312)` |
| #11 新位置 | `(320,6592)` |
| 其他 19 个特殊地貌 | 保存参数逐项一致 |
| 未变化的模型部分 | `surface, climatePreview, minerals, terrain, waterbed, profile, tectonicQuery, hydrologyQuery, ecology` |
| 发生变化的 payload 部分 | 只有 `config.WorldProfile` 的规则标识和 `specialFeatures` 的 #11 |

证据：`docs/acceptance/r81d_v124_candidate_summary.json`、`docs/acceptance/r81d_sinkhole_model_evidence_v123.json`、`docs/acceptance/r81d_sinkhole_model_evidence_v124.json`。

特别说明：曾在未保留原天坑占位的候选算法里发现一个盆地被动换址，已改用保留原占位的实现，最终服务端保存模型确认**不再有任何其他特殊地貌换址**。

## 4. 真实 Minecraft Anvil 方块复验

新世界 `R81dVerifiedGeoWorld` 由 1.21.1 NeoForge 服务端创建，绑定 v1.1.24，执行 `forceload`、`save-all flush`，再只读解析真实 region 的 `minecraft:full` 方块。并非离线模型模拟出方块。

比较 32 个相同坐标，共 30 个新旧双方均为 `full`，2 个仍有未生成区块。

### 必须通过的核心样本

| 测量位置 | v1.1.23 | v1.1.24 |
|---|---:|---:|
| 河段 #735 中心，t = .50 / .625 / .75 / .875，实际顶层河水 | 4/4 为 Y96 | **4/4 为 Y96** |
| t=.75，水道一侧 16 格自然基底 | Y79 | **Y100** |
| t=.75，水道一侧 32 格自然基底 | Y69 | **Y102** |
| t=.75，水道一侧 48 格自然基底 | Y69 | **Y102** |
| 天坑原中心 `(-1600,5312)` | Y69 | **Y103** |

另 3 组横断面分别在 t=.5/.625/.875：**共 12/12 个河岸偏移样本**都有真实方块；新版自然基底全部不低于对应河道水面，旧版对应地面全部低于河面。河道中心水位 4/4 不变。

核心验收条件全部为真：`fixedRiverAndBankCasePass=true`。

### 未计入已验证的外围采样

- `(-1704,5416)`：旧/新均未生成。
- `(-1750,5359)`：旧版已生成，新版尚未生成。

两者都在扩展横断面 **-48 格**侧，不是中心线或已确认的 12 个错误低岸点。保留 `full32PointCoverage=false`，不把它们计入 30 个有效配对或宣称“全部 32 点通过”。

实际方块逐点材料、自然基底、水方块及世界模型哈希均在：
`docs/acceptance/r81d_actual_voxel_v124_comparison.json`。

## 5. 测试、边界与发布限制

- 新增 Go 专项测试：河道穿坑、临近岸坡、弯道、远离河道、旧规则隔离及其他特殊地貌不受误伤。
- Go 完整相关包回归：`internal/generator`、`internal/specialfeatures`、`internal/runtimeworld`、`internal/worldmodelio`、`internal/worldsample`、`internal/waterbed`、`cmd/geoworld-r81-hydrology-audit` **全部通过**。
- Java 1.21.1 NeoForge `compileJava` **通过**；Java Runtime 固定点对照正常，新版本实际 Minecraft 世界绑定正常。
- Python `test_r81d_sinkhole_voxel_audit.py` **7 项通过**；真正的 Anvil 专项验收按必须存在的 4 个河心及 12 个低岸样本判定。
- 此次完成的是 **R8.1d 固定案例专项**，不等于整个 GeoWorld v0.1.5 RC 的视觉验收、多 Loader、多游戏版本 CI 完成。
- **不发布测试 JAR、不合并 master、不改旧存档、不将旧模型重新绑定为新规则。**

## 6. 离线复查

在同一工作树，模型核查（可使用已保存世界文件）：

```bash
go test -p 2 -count=1 \
 ./internal/generator ./internal/specialfeatures \
 ./internal/runtimeworld ./internal/worldmodelio \
 ./internal/worldsample ./internal/waterbed \
 ./cmd/geoworld-r81-hydrology-audit

python3 -m unittest discover -s scripts -p 'test_r81d*.py' -v
```

方块重审需要两个独立已生成的真实世界，输入通过 `--legacy-world`、`--new-world` 提供；命令如下：

```bash
python3 scripts/r81d_sinkhole_voxel_audit.py \
 --legacy-world <v1.1.23_world_path> \
 --new-world <v1.1.24_world_path> \
 --sites docs/acceptance/r81d_actual_voxel_baseline_v123.json \
 --output docs/acceptance/r81d_actual_voxel_v124_comparison.json
```

**下一验收门槛：** 游戏视角核查 #735 的整体河谷形态，及新的 #11 天坑所在地形是否合理；随后通过所有目标 Minecraft 版本/Loader 的构建和兼容回归，再决定新 RC 是否可发布。R8.2 雪山盆地和岛屿形态仍独立处理。
