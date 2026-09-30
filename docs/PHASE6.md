# Phase 6 — 统一世界模型与最小 Generator

更新时间：2026-09-30。当前进度以 STATUS.md 为准；本文保留评审依据、实施顺序与验收契约。

## 目标与边界

最近里程碑：可实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

保留现有板块、气候、水文、侵蚀、地貌、生态和 Profile 算法；停止扩展独立 Lab 页面和新分类。先补跨阶段约束，再扩展产品界面。Phase 1–5 的“完成第一版”是算法实验完成，不表示游戏世界或跨模块一致性已通过验收。

## 2026-09-29 评审证据（修正前快照）

代码基线：master / 661544d，叠加当时未提交的 Phase 5.2。
已重跑 go test ./... -count=1，以及 4 Seed / 640×400 geoworld-mcprofile-audit，均通过。

额外诊断使用 Seed 731894205、10400；PlateCount=18、OceanFraction=0.68；640×400；CellSizeBlocks=128；normalized sea level=0.5。在每个宏观节点采样 terrain.SampleNormalized(x*128,z*128)。

| 指标 | 731894205 | 10400 |
|---|---:|---:|
| 原始海洋标签但宏观高程 >= 0.5 | 950 | 647 |
| 原始陆地标签但宏观高程 < 0.5 | 1465 | 1331 |
| 海洋标签但最终连续高程 >= 0.5 | 1273 | 915 |
| 陆地标签但最终连续高程 < 0.5 | 1568 | 1425 |
| 排除已标记湖泊的较强河流连线 | 1081 | 1592 |
| 上述连线中原始高程向下的数量 | 873 | 1265 |
| 原本向下、最终高程变成向上的数量 | 82 | 124 |
| 最大逆向归一化高差 | 0.00908293 | 0.00497773 |

河网诊断口径：RiverStrength >= 0.30，起终点均为 LandMask，LakeID 均为 0；原始高差 > 1e-6；最终下游高于上游 > 1e-5。这里只检查节点间连线，未证明未来方块级河道已经出现逆流。最终河床、水面和河道几何尚未实现。

低于海平面的大陆地壳不一定应该灌海，可能是封闭洼地。海洋必须由与外海连通的淹水范围定义，不能简单改成 elevation < seaLevel，也不能继续把 crust == Continental 当作实际陆海判定。

发现：
1. climate/hydrology/erosion/landform adapter 分别从 Crust 重建 LandMask；构造高程允许跨海平面，因而语义不一致。
2. Terrain 与 Ecology/Profile 实际是分支；Ecology 使用侵蚀后的宏观高程，不是最终采样地形。允许气候仍使用宏观尺度，但最终海陆/湖面/群系必须协调。
3. Phase 1 构造带宽、部分气候距离与坡度仍使用固定 Cell 单位。已有山地比例回归只是部分护栏。
4. 置信度是内部启发式分数，不是地理正确率；比例审计和浮点连续性不能替代水文、存档及游戏验收。

## 分阶段交付

### 6.0 — 基线与进度保全
- 收口提交 Phase 5.2。
- STATUS 只保留当前状态；既往阶段原文完整移入 PROGRESS，不删除历史指标和决定。
- ROADMAP / ARCHITECTURE / DECISIONS 同步本次调整。

### 6.1 — 统一世界定义
6.1a：
- 世界范围（blocks）、原点、网格节点数、海平面单独定义。
- 从尺寸推导节点间距，区分物理世界大小与计算分辨率。
- 海洋：海平面以下且与模型边界外海连通；封闭低地保留陆地/洼地身份。
- Crust 与 LandMask 分离；同一宏观海陆结果被气候、水文、侵蚀、地貌共同使用。
- 地壳面积目标与实际海洋面积分别统计，不再把二者视为恒等。

6.1b：
- 逐项迁移空间尺度：构造带、海岸距离、海温/水汽衰减、坡度、河谷扩散。
- 使用明确的参考物理尺度迁移旧默认值；不能仅把参数名从 Cells 改成 Blocks。
- 固定世界范围变更精度时，允许离散化误差，但不应系统性改变地形作用范围。

### 6.2 — 统一生成入口与一致性校正
- GeneratorConfig -> GenerateWorld -> WorldModel。
- 解析一次默认值、验证范围并保存解析后的完整配置。
- 暂态调试字段与持久化/运行时所需字段分开，不直接导出所有 Lab 模型。
- 显式阶段依赖、错误、取消和进度；Lab 与正式入口逐步共享调度。
- 最终海岸与海洋连通、水体覆盖和生态同步。
- 河道纵剖面/河床、水位、湖面和溢流口约束；不能简单用填洼高程抹平真实地形。
- 最终地形修改后重新检查水文；不要通过隐藏逆向连线或调宽断言来通过。
- 保留连续地形、负坐标及模型外围过渡的约束。

### 6.3 — 可复现的最小 Generator
- 真实参数生成、阶段进度、基本图层及坐标查询。
- 保存和重新打开 World Model；中断/失败不能覆盖已有完整文件。
- 版本化 manifest：schemaVersion、generatorVersion、seed、解析配置、坐标约定、字段编码与校验。
- 相同模型重开后的固定样点完全一致；未知版本明确拒绝或显式迁移。
- 预览渲染精度不得悄悄改变世界生成配置。

### 6.4 — 跨语言与首次游戏验证（提前）
- 首先验证 Go 导出 -> Java 读取与连续采样；先不重写全部全球模拟。
- 固定样点包括负坐标、海岸、河口、湖泊、宏观节点/Chunk 边界、外围与角落。
- 明确哈希、整数溢出、浮点运算/量化及最终方块取整规则；保存 golden vectors。
- 选择一个 Minecraft 版本和一个 Loader，验证小范围真实 Chunk、地表、水体和群系。
- 增加高度/纵列查询与生成结果的一致性、区块生成顺序独立性、存档重开测试。
- 记录生成耗时、内存、模型大小与采样成本；目标预算按实际硬件测量后确定。
- 游戏内自主创建宏观模型仍是 Runtime 后续目标；外部导入验证不冒充完成该功能。

### 6.5 / Phase 7
首次游戏验证后再扩展完整 UI、多 Seed 对比、批量世界预生成、结构/洞穴/装饰兼容及外围玩法。多个 Loader 后续逐个适配，不同时展开。

## 验收矩阵

| 范围 | 必须满足 |
|---|---|
| 配置 | 非法尺寸/溢出/NaN/Inf 拒绝；世界范围与精度独立；已解析配置可持久化 |
| 宏观海陆 | 海洋节点不高于定义水位；外海连通；封闭洼地不被无条件灌海；出露岛弧参与陆地气候 |
| 河湖 | 河床/水位、出入口与最终地形相容；湖面统一；旧 FlowTo 逆坡显式暴露 |
| 地形 | 同坐标同结果；Chunk/Macro 连续；海岸及外边界纳入检查 |
| 生态/Profile | 与最终海陆/水体协调；离散标签不制造地形墙；Unknown=0 |
| 存档 | 同模型重开样点一致；版本/损坏检查；失败不破坏完整模型 |
| Runtime | 与 Go golden vectors 一致；实际方块与高度查询一致；区块顺序独立 |
| 性能 | 分阶段耗时、峰值/常驻内存、文件大小、采样成本有实测记录 |

## 持续记录规则

每个功能提交同步 STATUS 与本阶段勾选/结果。已完成、当前进行、未解决必须明确区分。审计数值标明 Seed、网格、配置、代码版本；修改模型后旧指标保留为历史，不冒充当前输出。新会话先读 STATUS -> PHASE6 -> DECISIONS，必要时读 PROGRESS 和各 PHASE 文档。

## Phase 6.1a 实施结果（2026-09-29）

基线 4a5f685；开发分支 agent/world-model-foundation；功能提交标题 feat: establish world geometry and shared macro surface。

已完成：
- [x] Phase 5.2 与调整后的规划提交。
- [x] GridGeometry：范围/原点/节点数分离；节点间距按 span/(nodes-1) 计算。
- [x] 几何有限性、乘积溢出、零/负范围、巨大原点丢失节点精度的检查。
- [x] ScalarGrid 使用统一几何校验；暂时明确拒绝非等距 X/Z 网格。
- [x] 统一默认归一化海平面 0.5；不是 Minecraft 方块 Y。
- [x] 高程 + 外海连通的宏观 Surface；地壳与实际陆海分离。
- [x] 四邻接且严格低于海平面才淹水；恰在海平面的门槛不开放；封闭低地不自动变海。
- [x] 各 adapter 传递共享的宏观 LandMask，不再从 Crust 重建。
- [x] 出露海洋地壳参与陆地气候、地貌与生态，输入切片不反向修改上游模型。
- [x] 持久保留一致性审计命令。
- [x] 所有当前已知旧空间参数迁移（6.1b）；最终残余审计已完成，后续新增空间参数继续遵守 GridGeometry / blocks 规则。
- [ ] 最终海岸、河床、水面与湖泊协调（6.2）。
- [ ] 统一入口、存取闭环及游戏验证。

### 验证结果

全量 go test ./... -count=1 通过；正式 Profile audit 的 4 个 Seed（10400、15723、21046、26369），640×400 通过。置信度 0.691–0.764；低置信度 0.086%–0.216%；Unknown=0。这仍是多样性护栏，不是最终世界正确率。

一致性审计：配置与修正前快照一致；模型输入已改变，因此流域/河网集合与分母可能变化，不能只比较逆坡数量就声称河道改善。

| 指标 | 731894205 | 10400 |
|---|---:|---:|
| 海洋地壳面积 | 68% | 68% |
| 实际宏观海洋面积 | 67.84140625% | 68.14140625% |
| 海洋地壳上出露的宏观节点 | 950 | 647 |
| 封闭且低于海平面的宏观节点 | 921 | 322 |
| 宏观海洋节点高于/等于海平面 | 0 | 0 |
| 各模块宏观海陆标签不一致 | 0 | 0 |
| 最终地形高于海面的旧海洋标签节点 | 336 | 326 |
| 最终地形低于海面的旧陆地标签节点（含合法封闭低地） | 1140 | 454 |
| 重新判定最终地形后海陆标签发生变化 | 806 | 608 |
| 非湖泊较强河流连线 | 1108 | 1685 |
| 其中原始地形向下的连线 | 908 | 1342 |
| 原本向下、最终变为向上的连线 | 72 | 126 |
| 最大逆向归一化高差 | 0.00658273 | 0.00681178 |

### 可重复执行

```text
go test ./... -count=1
go run ./cmd/geoworld-mcprofile-audit -count 4
go run ./cmd/geoworld-consistency-audit
go run ./cmd/geoworld-consistency-audit -strict-final
```

consistency-audit 默认对已实现的宏观契约执行失败检查，同时明确输出 UNRESOLVED 与最终地形诊断。-strict-final 将最终海陆变化或河网逆坡也列为失败条件；当前应失败，因为 6.2 尚未完成，不能作为已经通过的门槛汇报。

当前海洋四邻接属于宏观保守近似；细化时必须与水文 D8、海峡/河口和连续高程一起协调。旧 Phase 1 的地壳边缘保护仍不自动等于最终地形外围已验收。

phase1.Generate 仍保留旧的非 error 返回接口：内部产生非法高程会 panic。6.2 统一入口需先完整验证配置并改为可报告错误；本轮不把这项旧接口限制隐藏掉。

静态原型图片仍是先前实验快照，本轮没有批量重绘所有历史图。当前算法数值以命令审计为准，正式实时图层接入仍在 6.3。


## Phase 6.1b 第一切片实施结果（2026-09-30）

基线：7de68a4；开发分支：agent/world-model-foundation。

本切片只处理 Phase 1 的第一组空间尺度，不把整个 6.1b 误报为完成。

已完成：
- [x] Phase 1 Config 增加 OriginX / OriginZ / SpanXBlocks / SpanZBlocks。
- [x] 未显式给 Span 时按旧基线解析为 128 blocks/node，保留现有命令与审计默认行为。
- [x] Phase 1 World 保存已解析 GridGeometry；当前通过 SquareCellSize 明确拒绝 X/Z 非等距采样。
- [x] CoastDistance 从宏观 Cell 数改为 Minecraft blocks。
- [x] 大陆内部厚度衰减、基础高程、大陆架/大陆坡/海盆、海岸/高原分区阈值同步迁移为 blocks。
- [x] 构造影响带的旧 Cell 半径先映射到 128-block 参考物理半径，再按当前节点间距采样；不是简单改参数名。
- [x] coast-distance 渲染标尺同步保持旧物理含义。
- [x] 新增固定物理范围的 128 → 64 blocks/node 回归：CoastDistance 与构造影响在相同世界坐标保持一致。
- [ ] 板块 warp、局部粗糙度等其它 Phase 1 cell-frequency 逻辑仍未全部空间化；本切片只保证已列出的距离型参数。
- [ ] Climate / Erosion / Hydrology / Geomorph 的距离与坡度迁移尚未完成。

兼容性策略：
- 旧命令只给 Width / Height 时，世界物理尺寸仍按旧 128 blocks/node 推导，因此 640×400 仍对应约 81.8k×51.1k blocks。
- 显式给定相同 Span、改变 Width / Height 时，已迁移的距离型算法保持物理作用范围；未迁移算法仍可能产生差异，后续切片逐项收口。
- phase1.Generate 仍是 legacy panic API；非法几何通过 GridGeometry/SquareCellSize 被拒绝，但统一 error 返回延后到 6.2。

### 验证

- `go test ./internal/phase1 -count=1`：通过。
- `go test ./... -count=1`：通过。
- `go run ./cmd/geoworld-mcprofile-audit -count 4`：通过；Unknown=0；平均置信度 0.691–0.764；低置信度 0.086%–0.216%。
- `go run ./cmd/geoworld-consistency-audit`：通过当前宏观门槛；两个正式 Seed 的结果与 6.1a 基线一致。
- 最终连续海岸/河床问题仍明确为 UNRESOLVED，留到 6.2。

下一切片：把 GridGeometry 传入 climate，优先迁移 CoastalTempDecay 与水汽/海温传播距离；之后处理 erosion slope 与 hydrology/geomorph 空间核。


## Phase 6.1b 第二切片实施结果（2026-09-30）

基线：1439bcb；开发分支：agent/world-model-foundation。

本切片迁移 Climate 的距离、传播步长和坡度量纲；6.1b 仍未整体完成。

已完成：
- [x] world.DefaultMacroSpacingBlocks = 128 提升为跨阶段共享的旧 Cell 默认值参考尺度。
- [x] climate.Input / Model 保存 GridGeometry；InputFromPhase1 传入 Phase 1 的真实 Geometry。
- [x] Legacy 手写 climate Input 未提供 Geometry 时才兼容解析为 128 blocks/node；Geometry 尺寸不匹配或非等距明确报错。
- [x] CoastalTempDecayCells 迁为 CoastalTempDecayBlocks，默认 2304 blocks。
- [x] OceanDistanceCells 迁为 OceanDistanceBlocks；季节大陆性和季风 ocean-access 的旧距离常量同步按 128-block 参考尺度换算。
- [x] SST 平流方向保留旧 cardinal/diagonal 量化，但一次传播距离固定为 128 blocks，并按当前节点间距解析采样位置。
- [x] SST 扩散按 128-block 物理邻域采样；默认 128-block 网格与旧算法结果一致。
- [x] 水汽平流/扩散按 128-block 物理步长采样；默认 MoistureIterations 按世界物理 Span 推导，而不是按节点数推导。
- [x] Orographic Lift 的地形梯度按实际 GridGeometry 节点间距修正，再映射到旧 128-block 基线量纲。
- [x] 固定世界范围 128 → 64 blocks/node 回归覆盖 OceanDistance、Coastal temperature、SST transport、Moisture 和 Precipitation。
- [ ] Erosion slope / relief、Hydrology / Geomorph 距离核仍待下一切片。
- [ ] Phase 1 中非距离型的 grid-frequency warp/noise 仍待单独评估，不在本切片冒充已解决。

### 验证

- go test ./internal/climate -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-climate-audit：16 Seed / 320×200 通过。
- go run ./cmd/geoworld-monsoon-audit：12 Seed / 240×150 通过。
- go run ./cmd/geoworld-mcprofile-audit -count 4：默认 128-block 输出与 1439bcb 第一切片一致；Unknown=0；平均 confidence 0.691–0.764。
- go run ./cmd/geoworld-consistency-audit：731894205 / 10400 的全部当前诊断与 1439bcb 一致；最终连续海岸/河床问题仍为 UNRESOLVED。

下一切片：Erosion 的 slope / local relief 物理尺度，然后 Hydrology / Geomorph。


## Phase 6.1b 第三切片实施结果（2026-09-30）

基线：6a3ecf0；开发分支：agent/world-model-foundation。

本切片迁移 Erosion 的坡度量纲与 Relief 观察尺度；6.1b 仍未整体完成。

已完成：
- [x] erosion.Input / Model 接入 GridGeometry；InputFromWorld 传入 Phase 1 的真实 Geometry。
- [x] Legacy 手写 erosion Input 未提供 Geometry 时兼容解析为 128 blocks/node；Geometry 尺寸不匹配或非等距明确报错。
- [x] localSlope 的中心差分按真实节点间距缩放，并归一回旧 128-block reference slope 单位；默认 SlopeScale=0.06 的历史语义保持。
- [x] ReliefRadiusCells 迁为 ReliefRadiusBlocks。
- [x] 默认 Relief 半径继续按旧规则解析：约世界短边 1%，在 128-block reference grid 上取整并夹到 1..32 个旧 Cell，即 128..4096 blocks。
- [x] LocalRelief 在当前 GridGeometry 上按物理方形邻域采样；显式 ReliefRadiusBlocks 不再随网格精度改变物理作用范围。
- [x] 固定世界范围 128 → 64 blocks/node 回归覆盖 Slope、HillslopeErosion、默认/显式 ReliefRadius 与 LocalRelief。
- [ ] Hydrology / Geomorph 的距离、坡降和扩散尺度仍待下一切片。

### 验证

- go test ./internal/erosion -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-erosion-audit：8 Seed / 240×150 通过。
- 默认 128-block 公式在 slope scale=1、Relief radius 整数 Cell 时与旧路径等价；4 Seed Profile 与 2 Seed Consistency 最终复验均通过，数值与 6a3ecf0 基线一致。

下一切片：Hydrology 的距离/坡降量纲，然后 Geomorph / 河谷扩散。


## Phase 6.1b 第四切片实施结果（2026-09-30）

基线：6b3b23d；开发分支：agent/world-model-foundation。

本切片迁移 Hydrology 的几何契约、D8 坡降、填洼 epsilon、径流面积量纲与最小湖泊面积；6.1b 仍未整体完成。

已完成：
- [x] hydrology.Input / Model 接入 GridGeometry；InputFromWorld 传入 Phase 1 的真实 Geometry。
- [x] Legacy 手写 Hydrology Input 未提供 Geometry 时兼容解析为 128 blocks/node；Geometry 尺寸不匹配或非等距明确报错。
- [x] Priority Flood 的 FillEpsilon 按实际节点间距缩放；相同物理排水路径不会因为网格加密而累计更多人工高程。
- [x] D8 Flow Direction 的最陡坡降与 downstreamSlope 按实际物理距离计算，再归一到旧 128-block reference slope 量纲。
- [x] LocalRunoff 的累加贡献乘以 Cell 相对 reference cell 的物理面积比例；64 blocks/node 的单 Cell 贡献为 128 blocks/node 的 1/4。
- [x] MinLakeCells 迁为 MinLakeAreaBlocks2；默认保持旧 2×(128×128) blocks² 最小湖泊面积。
- [x] Lake metadata 增加 AreaBlocks2。
- [x] 固定物理范围 128 → 64 blocks/node 回归覆盖 FillEpsilon 累积、downstream slope、本地径流面积缩放与最小湖泊面积。
- [ ] D8 流向拓扑本身仍允许随采样精度出现离散差异；本切片不伪装成“不同分辨率河网逐 Cell 完全相同”。
- [ ] Geomorph 的坡降、上游记忆和扩散尺度仍待下一切片。

### 验证

- go test ./internal/hydrology -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-hydrology-audit：8 Seed / 240×150 通过；river land 2.212%–3.374%，filled land 0.615%–7.212%，max Strahler order 2–3，explicit lakes 0–5。
- go run ./cmd/geoworld-mcprofile-audit -count 4：默认 128-block 输出与 6b3b23d 基线一致；Unknown=0，平均 confidence 0.691–0.764。
- go run ./cmd/geoworld-consistency-audit：731894205 / 10400 的全部当前诊断与 6b3b23d 一致；最终连续海岸 / 河床问题仍为 UNRESOLVED。

下一切片：Geomorph 的坡降、上游记忆与河谷 / 冲积扇扩散空间尺度。


## Phase 6.1b 第五切片实施结果（2026-09-30）

基线：1dae70c；开发分支：agent/world-model-foundation。

本切片迁移 Geomorph 的 Geometry 契约、上游记忆衰减与冲积扇扩散范围；6.1b 仍未直接宣告整体完成，下一步先做全仓残余尺度审计。

已完成：
- [x] geomorph.Input / Model 接入 GridGeometry；InputFromModels 继承 Hydrology Geometry。
- [x] Legacy 手写 Geomorph Input 未提供 Geometry 时兼容解析为 128 blocks/node；Geometry 尺寸不匹配或非等距明确报错。
- [x] UpstreamSteepMemory / UpstreamMountainMemory 的旧每-hop 衰减 0.78 / 0.82 改为 baseDecay^(nodeSpacing / 128)。
- [x] 默认 128 blocks/node 时记忆衰减路径与旧算法等价；64 blocks/node 时两个 hop 的累计衰减对应旧一个 128-block hop。
- [x] 冲积扇旧 2..7 Cell 扩散半径解释为 256..896 blocks。
- [x] 冲积扇前向 / 横向距离及衰减统一在 128-block reference 坐标计算，并按当前 GridGeometry 重采样。
- [x] 固定世界范围 128 → 64 blocks/node 回归覆盖 UpstreamSteepMemory、UpstreamMountainMemory、SlopeDrop 与 AlluvialFanWeight。
- [ ] UpstreamSlope 仍是即时一跳上游统计，但目前不参与 Geomorph 权重决策，只作为诊断字段；是否需要物理窗口化留给后续模型演进，不阻塞本轮核心尺度迁移。
- [ ] 不要求不同采样精度的 D8 FlowTo 拓扑逐 Cell 相同，只要求已迁移的物理作用尺度不系统漂移。

### 验证

- go test ./internal/geomorph -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-geomorph-audit：4 Seed / 640×400 通过；floodplain 0.237%–0.519%，alluvial plain 1.716%–2.521%，fan 0.009%–0.093%，canyon 0–0.011%，wide valley 0.201%–0.373%。
- go run ./cmd/geoworld-mcprofile-audit -count 4：默认 128-block 输出与 1dae70c 基线一致；Unknown=0，平均 confidence 0.691–0.764。
- go run ./cmd/geoworld-consistency-audit：731894205 / 10400 的全部当前诊断与 1dae70c 一致；最终连续海岸 / 河床问题仍为 UNRESOLVED。

下一步：做 6.1b 全仓残余尺度审计，重点检查 Phase 1 grid-frequency warp/noise、Landform 空间核和其它仍按节点数推导的默认值；审计后再决定是否正式进入 6.2。


## Phase 6.1b 第六切片实施结果（2026-09-30）

基线：074e8d7；开发分支：agent/world-model-foundation。

全仓残余尺度审计发现 Landform 仍把宏观节点数当成世界尺寸代理。本切片将山地与河谷空间核正式迁移到 Minecraft blocks；6.1b 仍继续做剩余审计。

已完成：
- [x] landform.Input / Model 接入 GridGeometry；InputFromWorld 传入 Phase 1 Geometry。
- [x] Legacy 手写 Landform Input 未提供 Geometry 时兼容解析为 128 blocks/node；Geometry 尺寸不匹配或非等距明确报错。
- [x] MountainDistance 从 D8 Cell 距离改为 blocks；D8 cardinal / diagonal cost 分别为 node spacing / sqrt(2)×node spacing。
- [x] MountainInfluenceCells → MountainInfluenceBlocks。
- [x] ValleyRadiusCells → ValleyRadiusBlocks。
- [x] ValleyDecayCells → ValleyDecayBlocks。
- [x] 默认空间核先在旧 128 blocks/node reference world 上解释原有“短边比例”规则，再映射到当前 GridGeometry；相同物理 Span 改 Width / Height 不再改变默认 blocks 半径。
- [x] Valley spread 用真实 block 距离判断半径和指数衰减。
- [x] 固定物理范围 128 → 64 blocks/node 回归覆盖 MountainDistance、MountainInfluence、ValleyWeight。
- [ ] Phase 1 grid-frequency warp/noise、Terrain 独立 CellSizeBlocks 等仍待残余尺度审计。

### 验证

- go test ./internal/landform -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-landform-audit：8 Seed / 240×150 通过；plain 37.807%–53.272%，hill 17.241%–27.328%，foothill 6.085%–9.554%，mountain 21.978%–31.630%，strong valley 0.222%–0.523%。
- go run ./cmd/geoworld-mcprofile-audit -count 4：默认 128-block 输出保持当前基线；Unknown=0，平均 confidence 0.691–0.764。
- go run ./cmd/geoworld-consistency-audit：731894205 / 10400 的全部当前诊断保持基线；最终连续海岸 / 河床问题仍为 UNRESOLVED。

下一步：继续 6.1b 全仓残余尺度审计，重点检查 Phase 1 warp/noise 与 Terrain CellSizeBlocks / GridGeometry 的关系。


## Phase 6.1b 第七切片实施结果（2026-09-30）

基线：92d40be；开发分支：agent/world-model-foundation。

残余尺度审计继续检查 Phase 1，确认 plate warp、continent seed 与三组低频扰动仍以当前宏观节点坐标定义。本切片将这些生成逻辑统一迁到旧 128 blocks/node reference-world 坐标；默认 128-block 世界保持逐值兼容。

已完成：
- [x] Plate center 使用物理 Span 对应的 128-block reference grid 定义；当前节点先映射到 reference 坐标再执行 nearest plate。
- [x] warpedPlatePoint 的 width / height / amplitude 使用 reference grid 尺寸；相同物理 Span 下改变节点数不再改变 warp 波长与幅度。
- [x] continent seed center / radius 与 edge bias 使用 reference-world 坐标。
- [x] continent potential、CrustThickness、Macro Elevation 三组固定节点频率改为 reference-world 频率。
- [x] EdgeOceanBufferFraction 默认宽度先在 reference grid 中解析，再换算当前网格 Cell 数。
- [x] 固定物理 Span 128 → 64 blocks/node 回归覆盖 Plate center、RawPlateID、warped PlateID、LandPotential、三类低频 noise 与 edge buffer 物理宽度。
- [ ] Terrain 仍允许独立 CellSizeBlocks / Origin，与上游 GridGeometry 可能脱节，下一切片处理。

### 验证

- go test ./internal/phase1 -count=1：通过。
- go test ./... -count=1：通过。
- go run ./cmd/geoworld-audit：32 Seed / 320×200 通过；edge-buffer-warnings=0。
- go run ./cmd/geoworld-mcprofile-audit -count 4：默认 128-block 输出保持当前基线；Unknown=0，平均 confidence 0.691–0.764。
- go run ./cmd/geoworld-consistency-audit：731894205 / 10400 全部当前诊断保持基线；最终连续海岸 / 河床问题仍为 UNRESOLVED。

下一步：Terrain Input / Model 直接携带 GridGeometry，统一 scalar grid 的 origin / spacing，并保留受控 legacy 兼容入口。


## Phase 6.1b 第八切片实施结果（2026-09-30）

基线：24d124b；开发分支：agent/world-model-foundation。

本切片解决 Terrain 独立 `CellSizeBlocks / Origin` 可能与上游世界 Geometry 脱节的问题，并完成 6.1b 最终残余尺度审计。

已完成：
- [x] `terrain.Input` / `terrain.Model` 接入 `GridGeometry`；正式 `InputFromModels` 继承 Erosion Geometry。
- [x] Terrain ScalarGrid 全部改用 `sampling.NewScalarGridWithGeometry`。
- [x] `CellSizeBlocks / OriginX / OriginZ` 只保留 legacy 兼容；Input 未提供 Geometry 时才允许用它们构造兼容 Geometry，默认 128 blocks/node。
- [x] Geometry 已提供时，legacy cell size / origin 与 Geometry 冲突会明确失败，禁止静默错位。
- [x] Terrain Render 按 Geometry Origin + Span 采样，非零 Origin 不再错误从 0 开始渲染。
- [x] Terrain Lab / Audit 的 `-cell-size` 从 Phase 1 设置整条世界 Span；Consistency audit 同样使用整条 Geometry 并通过 `NodePosition()` 采最终地形。
- [x] 固定相同物理范围 128 / 64 blocks-node、非零 Origin 回归覆盖 BaseElevation、RegionalDetail、FinalHeight。
- [x] 最终全仓残余尺度搜索已完成；剩余 `*Cells` 均属于 blocks→当前采样网格的运行时变量、128-block reference 翻译、测试/统计或 legacy 兼容字段，未发现新的固定节点生成尺度。

### 验证

- `go test ./internal/terrain -count=1`：通过。
- `go test ./... -count=1`：通过。
- `go run ./cmd/geoworld-terrain-audit`：4 Seed / 640×400 / 128 blocks 通过；最大 |regional detail| 0.03314–0.03689，chunk epsilon jump 1.12e-7–1.48e-7，macro slope mismatch 2.26e-6–6.12e-6。
- 默认 `geoworld-consistency-audit`：731894205 / 10400 所有当前诊断保持 6.1b 前基线；最终连续海岸 / 河床问题仍为 UNRESOLVED。
- 非默认 `cell-size=64` 的 Phase1→Terrain Consistency 路径可完整运行，MacroMaskDisagreements=0，验证命令层 Geometry 已统一。

## Phase 6.1b 收口

状态：**完成**。

当前已迁移并验证的空间尺度链：

```text
Phase 1 reference-world / coast / tectonics
→ Climate transport / distance / orographic gradient
→ Hydrology D8 slope / runoff area / lake area
→ Erosion slope / relief radius
→ Landform mountain / valley kernels
→ Geomorph upstream memory / alluvial-fan spreading
→ Terrain geometry / sampling / origin
```

收口边界：
- 物理尺度稳定不等于不同采样精度逐 Cell 完全相同；D8 河网、离散边界和分类仍允许合理的量化/拓扑差异。
- 6.1b 不处理最终连续 Terrain 改变海岸、湖面或河床逆坡，这些仍是 Phase 6.2 的独立验收问题。
- 后续新增任何 distance / radius / step / wavelength 参数都必须以 GridGeometry / Minecraft blocks 为正式语义，不能重新引入“Cell 就是世界单位”的隐式假设。

下一阶段：**Phase 6.2 — GenerateWorld / WorldModel 与最终水体一致性。**
