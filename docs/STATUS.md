# GeoWorld 项目状态

更新时间：2026-09-30。当前状态以本文件为入口；历史逐阶段记录完整保存在 [PROGRESS.md](PROGRESS.md)。

## 当前目标与阶段
最近里程碑：能实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

**Phase 6.0、6.1a、6.1b 已完成；Phase 6.2 进行中。第一切片已建立统一 GenerateWorld / WorldModel；第二切片建立 Terrain/Water reconciliation 约束层；第三切片已把海岸 broad-surface correction 真正接入连续 `CoastTerrain`，两个正式 Seed 的 applied coast mismatch 均为 0，Chunk/Macro 连续性通过护栏。河/湖床 overlay 尚未应用，因此 raw 806/608 海陆变化与 72/126 逆坡仍保留，coast-only 河网仍有 58/124 条逆坡。下一步应用 RiverBedTarget / LakeBedCeiling。**

## 仓库与基线
- 仓库：D:\projects\minecraft-geo-world
- LADM Environment：env_59c743bd89627fe7
- 当前开发分支：agent/world-model-foundation
- master / Phase 5.2 基线：4a5f685 — feat: add minecraft generation profiles and phase 6 plan
- 上一生态基线：661544d
- Phase 6.1a：7de68a4 — feat: establish world geometry and shared macro surface。
- Phase 6.1b 第一切片：1439bcb — feat: migrate phase1 spatial scales to world units。
- Phase 6.1b 第二切片：6a3ecf0 — feat: migrate climate spatial scales to world units。
- Phase 6.1b 第三切片：6b3b23d — feat: migrate erosion spatial scales to world units。
- Phase 6.1b 第四切片：905c49f — feat: migrate hydrology spatial scales to world units。
- Phase 6.1b 第五切片：3f8b69a — feat: migrate geomorph spatial scales to world units。
- Phase 6.1b 第六切片：10e7ed7 — feat: migrate landform spatial scales to world units。
- Phase 6.1b 第七切片：444c3c3 — feat: migrate phase1 reference world scales。
- Phase 6.1b 第八切片：1a5fb4d — feat: bind terrain sampling to world geometry。
- Phase 6.2 第一切片：0a0b1f2 — feat: add unified world generation pipeline。
- Phase 6.2 第二切片：5b53ba9 — feat: add terrain water reconciliation constraints。
- Phase 6.2 第三切片：10da905 — feat: apply continuous coast terrain correction。
- 工作区的完成状态在提交后检查；继续前先核对 git status，不能仅凭本文假定 clean。

## 已完成
- Phase 1–5 算法第一版与 Profile 基线已提交。
- 评审、完整路线及验收标准已落入 PHASE6 / ROADMAP / DECISIONS / ARCHITECTURE。
- 原 STATUS 中的所有阶段记录全文保存为 PROGRESS.md。
- internal/world：GridGeometry，独立定义世界原点、范围和网格节点数；验证有限数值、尺寸溢出及可表示的间距。
- ScalarGrid 接入几何契约；当前旧采样器只支持相同 X/Z 间距，不支持时明确拒绝。
- 宏观 Surface 从高程和外海连通性生成，与 Crust 分离；四邻接、严格低于海平面才连通；封闭低地保留。
- climate -> hydrology -> erosion -> landform / geomorph -> ecology -> mcprofile 共享宏观海陆结果。
- 各阶段默认海平面使用统一常量；完整可配置海平面的调度约束仍待统一 Generator。
- 新增可重复执行的 geoworld-consistency-audit；明确区分已通过的宏观约束与未完成的最终地形约束。
- Phase 1 CLI 分开报告海洋地壳比例与实际宏观海洋比例。
- Phase 1 Config 现在显式携带 Origin / SpanXBlocks / SpanZBlocks；零 Span 仅作为兼容入口解析为旧默认 128 blocks/node。
- Phase 1 World 保存已解析 GridGeometry；当前仍明确要求 X/Z 等距，避免静默拉伸。
- CoastDistance 已从 Cell 单位迁到 blocks；大陆内部、陆架、陆坡、海盆、高原/海岸阈值同步按参考物理尺度换算。
- 构造影响带由固定 Cell 半径改为物理半径；不同宏观分辨率按 blocks 重新采样影响范围。
- `world.DefaultMacroSpacingBlocks = 128` 成为旧 Cell 默认值迁移的共享参考尺度；不是新的固定网格大小。
- Climate Input / Model 现在携带 GridGeometry；Phase 1 adapter 传入真实 Geometry，旧手写 Input 在未提供 Geometry 时才回退到 128 blocks/node。
- `CoastalTempDecayCells` / `OceanDistanceCells` 已迁为 `CoastalTempDecayBlocks` / `OceanDistanceBlocks`。
- SST 平流/扩散、水汽平流/扩散以 128 blocks 的参考物理步长解析；默认 MoistureIterations 按物理 Span 推导，不随节点数变化。
- Orographic Lift 的高程梯度已按实际节点间距校正，避免提高宏观精度就系统放大坡度。
- Erosion Input / Model 已接入 GridGeometry；正式 adapter 传入真实 Geometry，legacy 手写输入继续仅作 128 blocks/node 兼容。
- Erosion Slope 中心差分按真实节点间距校正后映射到旧 128-block reference slope；SlopeScale 默认语义不变。
- `ReliefRadiusCells` 已迁为 `ReliefRadiusBlocks`；默认继续表达约世界短边 1% 的物理观察窗口，并保留 128～4096 blocks 的旧夹取范围。
- Hydrology Input / Model 已接入 GridGeometry；正式 adapter 传入真实 Geometry，legacy 手写输入未提供 Geometry 时才回退到 128 blocks/node。
- Priority Flood 的 `FillEpsilon` 按实际节点间距缩放，同时保留旧 D8 每 hop 同 epsilon 的兼容语义；相同物理尺度的路径不会因为节点增多而累计更多人工抬升。
- D8 Flow Direction / downstream slope 已按真实节点间距归一回旧 128-block reference slope 量纲。
- 本地径流贡献按 Cell 物理面积缩放，避免相同流域仅因加密网格就产生约 4 倍累积流量。
- `MinLakeCells` 已迁为 `MinLakeAreaBlocks2`；Lake metadata 新增 `AreaBlocks2`，最小显式湖泊门槛不再随分辨率缩小。
- Geomorph Input / Model 已接入 GridGeometry；正式 adapter 继承 Hydrology Geometry，legacy 手写输入仍仅作 128 blocks/node 兼容。
- `UpstreamSteepMemory` / `UpstreamMountainMemory` 的每-hop 衰减按节点间距解析为 `baseDecay^(spacing/128)`；默认 128-block 时仍是 0.78 / 0.82。
- 冲积扇旧 `2..7 Cell` 扩散范围已解释为 256..896 blocks，并在当前 GridGeometry 上重新采样；前向/横向衰减也使用 128-block reference 坐标。
- Landform Input / Model 已接入 GridGeometry；MountainDistance 改为 blocks。
- `MountainInfluenceCells / ValleyRadiusCells / ValleyDecayCells` 已迁为 blocks；默认值按旧 128 blocks/node reference world 解析，不再直接依赖 Width / Height 节点数。
- Valley spread 以真实 block 距离判断半径与衰减，MountainInfluence 直接用 block 距离指数衰减。
- Phase 1 Plate center / plate warp / continent seed / edge bias 已统一在旧 128-block reference-world 坐标中定义；当前节点只负责采样。
- Phase 1 大陆潜力、CrustThickness 与 Macro Elevation 的固定节点频率已迁为 reference-world 频率；默认 128-block 数值保持不变。
- Edge ocean buffer 默认宽度先解释为 reference-world 物理宽度，再换算当前网格 Cell 数。
- Terrain Input / Model 现在直接携带上游 GridGeometry；正式 adapter 继承 Erosion Geometry，ScalarGrid 全部通过 `NewScalarGridWithGeometry` 创建。
- Terrain 的 `CellSizeBlocks / OriginX / OriginZ` 仅保留为 legacy 手写输入兼容；正式 Geometry 存在时若 legacy 配置冲突会明确报错，不允许静默错位。
- Terrain Render 使用 Geometry 的 Origin + Span，非零世界原点不再被错误当成 0 起点。
- Terrain Lab / Audit / Consistency 的 `cell-size` 现在从 Phase 1 定义整条世界 Geometry，不再只改变最终 Terrain 采样层。
- 新增 `internal/generator`：正式 `Config -> GenerateWorld -> WorldModel`，集中串联 Phase1→Climate→Hydrology→Erosion→Landform→Geomorph→Ecology→MCProfile→Terrain。
- 顶层 Seed / GridGeometry / SeaLevelNormalized 成为正式单一真相源；阶段 SeaLevel、Terrain seed/legacy geometry 若与顶层冲突直接拒绝。
- Phase 1 新增 `GenerateChecked` error API；旧 `Generate` 只保留为兼容 panic wrapper。
- `WorldModel.Config` 保存各阶段**实际解析后的 Config**，为 6.3 版本化持久化准备。
- `GenerateWorld` 支持固定阶段顺序的进度回调与 context 取消；当前取消粒度为阶段边界，不冒充阶段内部抢占。
- `geoworld-consistency-audit` 已改用统一 `GenerateWorld`，不再手工复制整条流水线。
- 新增 `internal/reconcile`，把最终 Terrain 与宏观 Surface / Hydrology 的冲突转成显式约束，而不是直接粗暴 clamp 整块地形。
- 海岸约束输出 `CoastTargetElevation`：宏观海洋控制点保持低于海平面，非封闭宏观陆地控制点保持不被最终 Terrain 淹没；封闭低地仍保留原语义。
- 水体元数据统一标记 Ocean / Lake / River；Ocean 水面=共享 SeaLevel，显式 Lake 使用 Hydrology Lake.SurfaceElevation，并生成 LakeBedCeiling。
- 河流约束输出独立 `RiverBedTarget`，仅雕刻河床纵剖面，不把河床 target 冒充成宽域 Terrain control point；多支流按有向拓扑传播，禁止通过抬高河床修逆坡。
- Reconciliation 作为 GenerateWorld 第 10 阶段并写入 WorldModel；ResolvedConfig 同步保存 reconciliation 参数。
- `sampling.SampleSmoothstepBilinear` 作为 correction overlay 插值：命中宏观控制点、保持凸包、不产生 overshoot，并使 correction 在宏观 Cell 边界的一阶导趋零。
- `terrain.WithHeightCorrection` 以不可变浅拷贝方式把 `CoastCorrection` 挂到连续 Terrain；Raw `Terrain` 继续保留用于历史/诊断，`WorldModel.CoastTerrain` 表示已应用 broad coast correction 的连续地形。
- GenerateWorld 在构建 CoastTerrain 后重新执行 `EvaluateCoastApplication`，并基于应用后的 broad terrain 重算 RiverBedTarget，避免河床 target 仍依赖旧 raw terrain。
- Consistency Audit 已增加 applied coast mismatch、节点命中误差、Chunk boundary jump 与 Macro boundary slope mismatch 护栏。

## 本轮验证
- go test ./... -count=1：全部通过（包括新的连通性、负坐标、分辨率/范围、Phase 1 物理尺度和全链路回归）。
- 新增固定物理范围多分辨率回归：128 → 64 blocks/node 时，同一物理位置的 CoastDistance、构造影响、OceanDistance、沿岸温度衰减、SST 传播、水汽/降水、Erosion Slope、HillslopeErosion、LocalRelief、Priority-Flood epsilon 累积、Hydrology slope、径流面积量纲、Geomorph 上游记忆/冲积扇扩散、Landform 山地距离/山麓影响/河谷扩散保持一致；最小湖泊阈值按物理面积验证。
- 4 Seed / 640×400 Profile audit：通过；Unknown=0；平均置信度范围 0.691–0.764，低置信度 0.086%–0.216%。
- 2 Seed / 640×400 consistency audit：宏观海洋高于海平面=0；跨模块海陆标签不一致=0。
- 8 Seed / 240×150 Erosion audit：通过；平均侵蚀 0.00200–0.00321，最大侵蚀 0.01480–0.02392，强侵蚀占比最大 0.314%。
- 8 Seed / 240×150 Hydrology audit：通过；河流陆地占比 2.212%–3.374%，填洼 0.615%–7.212%，最大 Strahler 2–3，显式湖泊 0–5。
- 4 Seed / 640×400 Geomorph audit：通过；河漫滩 0.237%–0.519%，冲积平原 1.716%–2.521%，冲积扇 0.009%–0.093%，峡谷 0–0.011%，宽谷底 0.201%–0.373%。
- 8 Seed / 240×150 Landform audit：通过；平原 37.807%–53.272%，丘陵 17.241%–27.328%，山麓 6.085%–9.554%，山地 21.978%–31.630%，强河谷 0.222%–0.523%。
- 4 Seed / 640×400 Terrain audit：通过；`|regional detail|` 0.03314–0.03689，chunk epsilon jump 1.12e-7–1.48e-7，macro slope mismatch 2.26e-6–6.12e-6。
- Terrain 固定物理范围 128 / 64 blocks-node 回归：非零 Origin 下 BaseElevation / RegionalDetail / FinalHeight 在对应世界坐标一致；legacy Geometry 冲突明确拒绝。
- 非默认 `cell-size=64` 的完整 Consistency 命令可贯穿 Phase1→Terrain，跨模块 LandMask disagreement=0，证明命令层 Geometry 不再脱节。
- 6.1b 最终全仓残余搜索已完成：剩余 `*Cells` 均为 blocks→当前网格的运行时采样变量、旧 128 reference 翻译、测试/统计或 Terrain legacy 兼容字段，未发现新的固定节点生成尺度。
- `internal/generator` 回归：统一流水线与原手工流水线的 Phase1 Elevation、Hydrology FlowTo、MC Profile 及固定点 Terrain 结果一致；统一 SeaLevel=0.47 可贯穿所有相关阶段；非法 Geometry/共享配置冲突返回 error；当前包含 reconcile 的 10 阶段进度顺序与取消通过。
- Consistency Audit 改用 GenerateWorld 后，731894205 / 10400 的全部默认 128-block 诊断与旧基线逐项一致。
- Reconciliation 纯单元测试覆盖：全海洋高 Terrain 的海岸 carve、全陆地低 Terrain 的海岸 raise、河床单调向下且绝不抬床、湖面/湖床 ceiling 元数据。
- Seed 731894205：原始 FinalSurfaceMaskChanges=806、FinalUpslopeEdges=72；constraint 只需调整 665 个海岸控制点（raise 329 / carve 336）即 SurfaceMismatchAfter=0；河床 target 雕刻 98 个节点即 TargetUpslope=0；发现 1 个 LakeBedViolation。
- Seed 10400：原始 FinalSurfaceMaskChanges=608、FinalUpslopeEdges=126；constraint 调整 510 个海岸控制点（raise 184 / carve 326）即 SurfaceMismatchAfter=0；河床 target 雕刻 162 个节点即 TargetUpslope=0；发现 7 个 LakeBedViolation。
- Consistency 默认门槛要求 constraint 自身 SurfaceMismatchAfter=0、TargetUpslopeRiverEdges=0，并要求 applied coast mismatch=0、节点误差≤1e-10、Chunk jump≤1e-5、Macro slope mismatch≤1e-4。
- Seed 731894205：CoastTerrain applied mismatch=0、节点误差=0、Chunk jump=5.59045e-8、Macro slope mismatch=6.55886e-6；coast-only river upslope=58，重算 river target 需 carve 110 个节点，LakeBedViolation=1。
- Seed 10400：CoastTerrain applied mismatch=0、节点误差=0、Chunk jump=6.29519e-8、Macro slope mismatch=6.74817e-6；coast-only river upslope=124，重算 river target 需 carve 171 个节点，LakeBedViolation=7。
- Raw baseline 806/608 FinalSurfaceMaskChanges 与 72/126 FinalUpslopeEdges 仍保留；`-strict-final` 仍应失败，因为河/湖床 overlay 尚未应用。
- 基线原型引用检查：387 处本地引用，无缺失。
- 详细参数、当前数值和历史对照见 PHASE6.md。

## 仍未完成（不要误报完成）
1. Raw 连续 Terrain 仍保留历史海岸问题：两个 Seed 的宏观/最终海陆变化节点为 806、608；但 CoastTerrain 已把宏观控制点应用后 mismatch 真正降为 0。下一步还需更细的水边界/Runtime 采样语义，不能只凭节点控制点宣告完整海岸完成。
2. 河/湖床 overlay 尚未应用：raw 河网逆坡为 72/908、126/1342；coast-only 后为 58、124。RiverBedTarget 已基于 CoastTerrain 重算并可达 target upslope=0，但仍只是待应用的局部床面约束。
3. Phase 6.1b 已完成当前已知生成尺度迁移；但“物理尺度稳定”不等于不同采样精度逐 Cell 完全相同。D8 河网、离散边界/分类、量化采样仍允许合理的拓扑差异，后续新增任何空间参数也必须继续遵守 GridGeometry / blocks 规则。
4. 最终海岸/湖面/生态协调、外围拓扑与四角过渡仍待实现。宏观四邻接海洋与 D8 水文的细化协调属于后续水体约束。
5. `phase1.Generate` 仍保留 legacy panic wrapper，但正式统一入口已改用 `phase1.GenerateChecked` 返回 error；其它旧 Lab/CLI 尚未全部迁到统一 Generator。
6. WorldModel 版本化保存/重开、Java reader/sampler、实际游戏生成均未完成。
7. 静态原型生成/保存按钮未接入；prototype/generated 仍是先前实验快照，不能代表本轮修改后的实时结果。

## 下一步（6.2）
1. 统一 `GenerateWorld` / `WorldModel` 调度第一切片已完成；继续把正式 Lab/审计逐步收敛到该入口。
2. `CoastTargetElevation` 已通过 smoothstep correction 真正接入连续 CoastTerrain，并通过节点命中、Chunk boundary、Macro slope 护栏。
3. 下一切片把基于 CoastTerrain 重算后的 `RiverBedTarget` 与 `LakeBedCeiling` 作为独立 carve-only 水道/湖床 overlay 接入 Terrain，而不是修改整块 broad terrain；随后重新计算 applied river/lake 一致性。
4. 保留当前 Consistency audit 的未解决基线作为 6.2 验收门槛；只有 FinalSurfaceMaskChanges / FinalUpslopeEdges 被真实协调后才能宣告可玩世界一致性。
5. 6.2 后进入 6.3 WorldModel 版本化保存/重开；6.4 再做 Go→Java→一个 Loader 的早期游戏内验证。

## 继续工作入口
先读 STATUS -> PHASE6 -> DECISIONS；查历史读 PROGRESS 和 PHASE1–5。每个开发切片保留验证口径、结果、剩余风险并提交。
