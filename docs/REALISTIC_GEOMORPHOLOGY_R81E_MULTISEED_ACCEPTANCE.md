# GeoWorld R8.1e — 多种子地貌坡面验收（模型级）

日期：2026-11-12  
分支：`adm/geoworld-r81e-sinkhole-geomorph-20261111`  
基线提交：`1ae1d42`（R8.1e 单种子完整模型/Java/真实方块专项）  
新规则：`geoworld-rules-v1.1.25`；对照：`geoworld-rules-v1.1.24`  
**阶段结论：3 个新增种子全部通过模型级坡面/历史布局隔离回归；其中种子 123456 的天坑 #9 已额外通过真实 Minecraft 方块专项验收。其他两个新增种子的真实方块尚未检查。整版 RC 不准发布。**

## 方法与边界

使用已有 **真实 Minecraft 1.21.1 NeoForge 保存模型**的创建参数、世界几何和海平面，分别为每个新种子调用完整 `generator.GenerateWorld` 两次，生成 v1.1.24 / v1.1.25 对照模型，而不是改写原世界。模型不写入或覆盖用户旧存档。

每组比较：
- 已规划河段 `Waterbed.Rivers` **完整结构深比较**；不是仅比较数量。
- `SpecialFeatures` **完整结构深比较**（包括类型、ID、坐标、半径、坑底和盆地排水）；不是仅比较数量。
- 底层 `Surface` 深比较，防止无关宏观地形变化。
- 每个天坑东西/南北两条固定坐标剖面，按 4 Minecraft 格间隔计算新/旧高度变化；坡率是归一化高度乘 514 的模型代理值，**不是**真实 Minecraft 方块坡度。
- 每个样本检查新版地面是否抬高了原始地面、是否无意修改水域样本。
- 本轮全程不改地形生成算法，不做新版本号升级。

## 三个新增种子的结果

| 种子 | River segments | SpecialFeatures | Sinkholes | v1.1.24 / v1.1.25 |
|---|---:|---:|---:|---|
| `123456` | 955 | 18 | 4 | 完整河网、特殊地貌和宏观 Surface 一致 |
| `20261112` | 873 | 19 | 4 | 完整河网、特殊地貌和宏观 Surface 一致 |
| `-90210` | 896 | 19 | 4 | 完整河网、特殊地貌和宏观 Surface 一致 |

**总计 12 个新种子天坑、24 条东西/南北剖面**，全部满足：

- 坡面最大坡率相对于对应旧版有下降；无新增采样处更陡。
- 旧版内侧大片同高地面样本数显著下降。
- 在剖面样本中水域原有高度保持不变；新地面没有高于未经天坑改造的原地面。
- 各种子测试没有报告异常告警。

模型坡率的合并统计：

| 指标 | v1.1.24 | v1.1.25 |
|---|---:|---:|
| 24 剖面中的最大代理坡率 | 3.129 | 0.540 |
| 坡率中位数 | 2.042 | 0.420 |
| 接近坑底的同高采样数 | 各剖面至少 34 | 各剖面最多 11 |

上述 24 条剖面中最大新版坡率 / 对应旧版坡率的比例约为 0.233。**这不是 3 种子真实方块验收或视觉评估结果**，只能证明对应模型采样的改进。

完整可重复证据：

- `cmd/geoworld-r81e-multiseed-audit/main.go`：运行真正的 `GenerateWorld` 双版本对照，按种子输出完整采样与警告。
- `docs/acceptance/r81e_multiseed_model_validation.json`：3 个种子、12 个天坑、24 条剖面的原始结果。
- `docs/acceptance/r81e_multiseed_summary.json`：合计统计、模型级限制。
- `scripts/test_r81e_multiseed_model_validation.py`：5 项可重复回归测试。

## 新增真实 Minecraft 种子验收：123456 / 天坑 #9（已通过）

**更正前期状态判断：** 第一次服务器启动被误判为一直卡在 `Preparing level`。完整日志表明：

- 2026-11-12 **09:56:18** 进入新世界准备；**10:05:13** 实际打印 `Done (534.574s)`；**10:05:31** 将 v1.1.25 GeoWorld 模型保存并绑定，`seed=123456`。约 535 秒是异常偏长的**首次启动**，不是失败。
- **10:05:45** 开始停止，**10:06:10** 完整保存后释放 GeoWorld 模型；首次运行在控制台连接异常期间已经结束。
- 后续在**同一个已经正确绑定的世界**重启，Minecraft 日志显示 **`Done (23.917s)`**，再次从原存档读取并激活 GeoWorld；未重新创建世界或覆盖旧存档。
- 使用 RCON 仅针对这个世界强制生成、保存天坑 #9 周围的横断面区块（69 + 60 个目标区块含相交部分），`save-all flush`，成功后撤销强制加载、再次保存并正常 `stop`。
- **原因边界：** 证实首次启动高耗时和该次 dadm CPU/内存/I/O 负载较高；但没有独立剖析到具体代码/原生调用，不能认定其为死锁、地形算法错误或明确根因，尚不算性能问题修复。

**真实 Minecraft 存档前置校验**由 `scripts/verify_geoworld_voxel_world.py` 完成，全部为真：世界生成器 `geoworld:geoworld`、`level.dat` seed=123456、`data/geoworld/geoworld.properties` binding rules `v1.1.25`、模型真实存在、region 文件真实存在。

天坑模型对应 ID **#9**，中心 **`(6080,7744)`**；真实存档只读 Anvil 方块剖面工具：
`scripts/r81e_multiseed_actual_voxel_audit.py`。

| 验收点 | 真实 Minecraft 保存方块结果 |
|---|---|
| 东西 X 轴（Z=7744），每 8 格一列 | **41/41 `minecraft:full`** |
| 南北 Z 轴（X=6080），每 8 格一列 | **41/41 `minecraft:full`** |
| 天坑中心天然土石基底 | **Y=74** |
| 天坑中心地表积水 | **未检出** |
| 东西最大自然地面坡率（8 格采样） | **0.5 Y/格** |
| 南北最大自然地面坡率（8 格采样） | **0.5 Y/格** |
| 中心同高度地面样本 | 两条剖面各 3 列 |
| 方块坐标缺失或未生成 | **0/82** |

新增完整证据文件：
- `docs/acceptance/r81e_seed123456_actual_voxels.json`：每个坐标的 Minecraft 原生区块状态、高度和土石方块类型，附种子/规则绑定信息和模型 SHA256。
- `docs/acceptance/r81e_seed123456_world_boot_evidence.json`：首次启动、同一存档重启、正式保存模型 River/Feature 数量和与 Go 多种子预测坐标的逐项核验。
- `docs/acceptance/r81e_seed123456_anvil_planview.png`：真实方块 OCEAN_FLOOR 顶视投影，仅 **4200/7921** 个 4 格网格具备完整区块，剩余点灰色且不作通过声明；**不是游戏相机截图**。
- `scripts/test_r81e_seed123456_actual_voxel.py`：5 个固定证据检查。
- `scripts/test_r81e_multiseed_actual_voxel_audit.py`：缺失区块必须显式拒绝，不允许用模型高度填补。

**该种子 #9 的实际方块局部剖面已完成，不表示其他 3 个天坑、全部区块、游戏视觉或其他 2 个新增种子通过。**

## 结论与剩余门槛

**通过：** R8.1e 固定种子（此前已实际 Minecraft 方块验收）；额外三个种子、十二个天坑的 Go 模型级回归；其中 `123456` 的天坑 #9 又完成了一次独立的真实 Minecraft 82 列方块专项验收。旧版本配置及河网隔离正常。

**未通过或未做：** `20261112` / `-90210` 的真实 Minecraft 方块、其他天坑的方块、游戏内相机视角、更多气候与预设配置、整个 v0.1.5 RC 的九目标正式 CI；首次创建世界耗时偏长的原因也未明确。当前记录不应被当作最终发布许可。

本轮不合并 master、不推送、不发布 JAR、不修改任何旧存档，也不扩展坡面算法。

## 复验命令

```bash
go run ./cmd/geoworld-r81e-multiseed-audit \
 java/adapters/1.21.1-neoforge/run/R81eVerifiedGeoWorld/data/geoworld/world.geoworld \
 docs/acceptance/r81e_multiseed_model_validation.json \
 123456 20261112 -90210

python3 -m unittest discover -s scripts \
 -p 'test_r81e_multiseed*.py' -v
```

上面为模型级复验。针对已保存的 **真实 Minecraft v1.1.25 世界**单独复验天坑 #9：

```bash
python3 scripts/r81e_multiseed_actual_voxel_audit.py \
 --world java/adapters/1.21.1-neoforge/run/R81eSeed123456VoxelAudit \
 --seed 123456 --center-x 6080 --center-z 7744 --site-id 9 \
 --output docs/acceptance/r81e_seed123456_actual_voxels.json \
 --image docs/acceptance/r81e_seed123456_anvil_planview.png
```

只能读取真正生成的 `minecraft:full` 方块；不会用模型预测补足尚未加载的地图。
