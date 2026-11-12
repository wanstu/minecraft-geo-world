# GeoWorld R8.1e — 多种子地貌坡面验收（模型级）

日期：2026-11-12  
分支：`adm/geoworld-r81e-sinkhole-geomorph-20261111`  
基线提交：`1ae1d42`（R8.1e 单种子完整模型/Java/真实方块专项）  
新规则：`geoworld-rules-v1.1.25`；对照：`geoworld-rules-v1.1.24`  
**阶段结论：3 个新增种子全部通过模型级坡面/历史布局隔离回归；新增种子的真实 Minecraft 方块验收尚未完成。整版 RC 不准发布。**

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

## 新增真实 Minecraft 种子验证尝试（未完成）

在本分支 NeoForge 1.21.1 开始生成一个**全新、隔离**的 `R81eSeed123456VoxelAudit` 世界：

- 新世界种子 `123456`，`level-type=geoworld:geoworld`，指定 loopback 端口，目标天坑 #9（模型预计 `6080,7744`）。
- 服务端启动并到达 `Preparing level`，随后较长时间没有进入 `Done` 或输出 `data/geoworld/world.geoworld`。
- dadm 的 ADM MCP 连接曾在此期间短暂不可用，机器出现资源压力；没有足够证据判断到底是 Minecraft 世界初始化、宿主机 I/O、进程或 ADM 服务的问题，**不据此创造或宣布新地形 bug**。
- 在生成绑定模型和目标区块前终止本次独立试验；没有验证 `123456` 的真实方块，也没有覆盖 `R81eVerifiedGeoWorld` 或历史存档。
- `scripts/r81e_multiseed_actual_voxel_audit.py` 已准备好做真正的存档预检查及读只读 Anvil 剖面。缺失或未生成区块必须拒绝验收；不能以模型高度替代 Minecraft 方块。此脚本此轮**没有成功完成真实世界审计**。
- `scripts/test_r81e_multiseed_actual_voxel_audit.py` 专项检查缺块、采样坐标、未证实高度不可计算坡度等负向边界。

## 结论与剩余门槛

**通过：** R8.1e 新天坑坡面变更在固定种子（此前已实际 Minecraft 方块验收）之外，额外三个种子、十二个天坑的 Go 模型级回归；旧版本配置及河网隔离正常。

**未通过或未做：** 第二种子的真实方块证据、游戏内相机视角、更多气候与预设配置、整个 v0.1.5 RC 的九目标正式 CI。当前记录不应被当作最终发布许可。

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

上面为模型级复验。真实 Minecraft 方块工具需给 `--world` 一个已生成且版本/种子绑定一致的 NeoForge 世界；对不完整的 `R81eSeed123456VoxelAudit` 不可用。
