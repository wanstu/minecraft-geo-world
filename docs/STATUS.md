# GeoWorld 项目状态

更新时间：2026-09-29。当前状态以本文件为入口；历史逐阶段记录完整保存在 [PROGRESS.md](PROGRESS.md)。

## 当前目标与阶段
最近里程碑：能实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

**Phase 6.0 已完成；Phase 6.1a 已完成首版并验证。下一步是 6.1b：统一旧算法的空间尺度。**

## 仓库与基线
- 仓库：D:\projects\minecraft-geo-world
- LADM Environment：env_59c743bd89627fe7
- 当前开发分支：agent/world-model-foundation
- master / Phase 5.2 基线：4a5f685 — feat: add minecraft generation profiles and phase 6 plan
- 上一生态基线：661544d
- 本次功能提交：feat: establish world geometry and shared macro surface（以 git log 获取提交号）。
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

## 本轮验证
- go test ./... -count=1：全部通过（包括新的连通性、负坐标、分辨率/范围和全链路回归）。
- 4 Seed / 640×400 Profile audit：通过；Unknown=0；平均置信度范围 0.691–0.764，低置信度 0.086%–0.216%。
- 2 Seed / 640×400 consistency audit：宏观海洋高于海平面=0；跨模块海陆标签不一致=0。
- 基线原型引用检查：387 处本地引用，无缺失。
- 详细参数、当前数值和历史对照见 PHASE6.md。

## 仍未完成（不要误报完成）
1. 最终连续地形改变海岸：两个 Seed 的宏观/最终海陆变化节点为 806、608；最终海洋标签高于海平面的节点为 336、326。
2. 最终河床/水面校正未做：同口径河网逆坡连线为 72/908、126/1342。海陆修正不是河网修复，第二个 Seed 该计数还略有增加。
3. 固定 Cell 的构造带、海温衰减、坡度等尚未全部迁移；坐标契约不等于已实现全算法精度无关。
4. 最终海岸/湖面/生态协调、外围拓扑与四角过渡仍待实现。宏观四邻接海洋与 D8 水文的细化协调属于后续水体约束。
5. phase1.Generate 仍保留旧 API 的默认值与 panic 方式；统一入口需要完整配置校验与错误返回。
6. WorldModel 版本化保存/重开、Java reader/sampler、实际游戏生成均未完成。
7. 静态原型生成/保存按钮未接入；prototype/generated 仍是先前实验快照，不能代表本轮修改后的实时结果。

## 下一步（6.1b）
1. 把固定世界范围/网格精度显式传入模拟配置；不能只改某个 Lab 的输出尺寸。
2. 优先迁移 Phase 1 构造带宽与海岸尺度，再迁移气候衰减/传输、坡度和水文距离。
3. 用固定物理范围下的多分辨率检查衡量作用距离、海岸与地貌变化；保留离散化误差，不把所有图强行做成相同。
4. 然后进入 6.2 的 GenerateWorld 与最终水体约束，再做 6.3 保存/重开。
5. 6.4 提前验证 Go -> Java -> 一个 Loader，完整界面与批量导出随后扩展。

## 继续工作入口
先读 STATUS -> PHASE6 -> DECISIONS；查历史读 PROGRESS 和 PHASE1–5。每个开发切片保留验证口径、结果、剩余风险并提交。
