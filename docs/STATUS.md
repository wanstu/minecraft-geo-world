# GeoWorld 项目状态

更新时间：2026-09-30。当前状态以本文件为入口；历史逐阶段记录完整保存在 [PROGRESS.md](PROGRESS.md)。

## 当前目标与阶段
最近里程碑：能实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

**Phase 6.0、6.1a 已完成；Phase 6.1b 进行中。Phase 1 与 Climate 两个空间尺度切片已实现并验证，下一步进入 Erosion / Hydrology / Geomorph。**

## 仓库与基线
- 仓库：D:\projects\minecraft-geo-world
- LADM Environment：env_59c743bd89627fe7
- 当前开发分支：agent/world-model-foundation
- master / Phase 5.2 基线：4a5f685 — feat: add minecraft generation profiles and phase 6 plan
- 上一生态基线：661544d
- Phase 6.1a：7de68a4 — feat: establish world geometry and shared macro surface。
- Phase 6.1b 第一切片：1439bcb — feat: migrate phase1 spatial scales to world units。
- Phase 6.1b 第二切片：Climate physical scale migration（提交号以 git log 获取）。
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

## 本轮验证
- go test ./... -count=1：全部通过（包括新的连通性、负坐标、分辨率/范围、Phase 1 物理尺度和全链路回归）。
- 新增固定物理范围多分辨率回归：128 → 64 blocks/node 时，同一物理位置的 CoastDistance、构造影响、OceanDistance、沿岸温度衰减、SST 传播、水汽与降水结果保持一致。
- 4 Seed / 640×400 Profile audit：通过；Unknown=0；平均置信度范围 0.691–0.764，低置信度 0.086%–0.216%。
- 2 Seed / 640×400 consistency audit：宏观海洋高于海平面=0；跨模块海陆标签不一致=0。
- 基线原型引用检查：387 处本地引用，无缺失。
- 详细参数、当前数值和历史对照见 PHASE6.md。

## 仍未完成（不要误报完成）
1. 最终连续地形改变海岸：两个 Seed 的宏观/最终海陆变化节点为 806、608；最终海洋标签高于海平面的节点为 336、326。
2. 最终河床/水面校正未做：同口径河网逆坡连线为 72/908、126/1342。海陆修正不是河网修复，第二个 Seed 该计数还略有增加。
3. Phase 1 距离型尺度与 Climate 海温/水汽/地形抬升尺度已迁移；Erosion 的 slope/relief、Hydrology / Geomorph 的距离核，以及 Phase 1 部分 grid-frequency warp/noise 仍未全部迁移。坐标契约不等于已实现全算法精度无关。
4. 最终海岸/湖面/生态协调、外围拓扑与四角过渡仍待实现。宏观四邻接海洋与 D8 水文的细化协调属于后续水体约束。
5. phase1.Generate 仍保留旧 API 的默认值与 panic 方式；统一入口需要完整配置校验与错误返回。
6. WorldModel 版本化保存/重开、Java reader/sampler、实际游戏生成均未完成。
7. 静态原型生成/保存按钮未接入；prototype/generated 仍是先前实验快照，不能代表本轮修改后的实时结果。

## 下一步（6.1b）
1. Phase 1 与 Climate 的第一批物理尺度迁移已完成；继续把 GridGeometry 传入 erosion / hydrology / geomorph 的正式输入契约。
2. 下一切片优先迁移 Erosion slope / relief，再迁移 Hydrology / Geomorph 的距离核与河谷扩散。
3. 每一批都保留固定物理范围下的多分辨率检查；允许离散化误差，但作用距离、坡度量纲和传播距离不能随网格精度系统漂移。
4. 然后进入 6.2 的 GenerateWorld 与最终水体约束，再做 6.3 保存/重开。
5. 6.4 提前验证 Go -> Java -> 一个 Loader，完整界面与批量导出随后扩展。

## 继续工作入口
先读 STATUS -> PHASE6 -> DECISIONS；查历史读 PROGRESS 和 PHASE1–5。每个开发切片保留验证口径、结果、剩余风险并提交。
