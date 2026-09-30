# GeoWorld 项目状态

更新时间：2026-09-30。当前状态以本文件为入口；历史逐阶段记录完整保存在 [PROGRESS.md](PROGRESS.md)。

## 当前目标与阶段
最近里程碑：能实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

**Phase 6.0、6.1a、6.1b 已完成。Phase 1 / Climate / Erosion / Hydrology / Landform / Geomorph / Terrain 的已知空间尺度与 Geometry 契约已完成迁移并通过最终残余审计；下一步正式进入 Phase 6.2：统一 GenerateWorld / WorldModel 与最终海岸、水体、河床一致性。**

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
- 基线原型引用检查：387 处本地引用，无缺失。
- 详细参数、当前数值和历史对照见 PHASE6.md。

## 仍未完成（不要误报完成）
1. 最终连续地形改变海岸：两个 Seed 的宏观/最终海陆变化节点为 806、608；最终海洋标签高于海平面的节点为 336、326。
2. 最终河床/水面校正未做：同口径河网逆坡连线为 72/908、126/1342。海陆修正不是河网修复，第二个 Seed 该计数还略有增加。
3. Phase 6.1b 已完成当前已知生成尺度迁移；但“物理尺度稳定”不等于不同采样精度逐 Cell 完全相同。D8 河网、离散边界/分类、量化采样仍允许合理的拓扑差异，后续新增任何空间参数也必须继续遵守 GridGeometry / blocks 规则。
4. 最终海岸/湖面/生态协调、外围拓扑与四角过渡仍待实现。宏观四邻接海洋与 D8 水文的细化协调属于后续水体约束。
5. phase1.Generate 仍保留旧 API 的默认值与 panic 方式；统一入口需要完整配置校验与错误返回。
6. WorldModel 版本化保存/重开、Java reader/sampler、实际游戏生成均未完成。
7. 静态原型生成/保存按钮未接入；prototype/generated 仍是先前实验快照，不能代表本轮修改后的实时结果。

## 下一步（6.2）
1. 建立正式 `GenerateWorld` / `WorldModel`，集中解析 Seed、Geometry、海平面和各阶段配置，消除当前 CLI 手工串联模块的重复调度。
2. 在统一 WorldModel 上实现最终连续海岸 / 外海 / 封闭低地 / 显式湖泊的一致水面语义。
3. 修复最终连续 Terrain 对河网造成的逆坡：明确河床纵剖面 / 水面约束，而不是隐藏或删除逆坡边。
4. 保留当前 Consistency audit 的未解决基线作为 6.2 验收门槛；只有 FinalSurfaceMaskChanges / FinalUpslopeEdges 被真实协调后才能宣告可玩世界一致性。
5. 6.2 后进入 6.3 WorldModel 版本化保存/重开；6.4 再做 Go→Java→一个 Loader 的早期游戏内验证。

## 继续工作入口
先读 STATUS -> PHASE6 -> DECISIONS；查历史读 PROGRESS 和 PHASE1–5。每个开发切片保留验证口径、结果、剩余风险并提交。
