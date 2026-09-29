# GeoWorld 项目状态

更新时间：2026-09-29。当前状态以本文件为入口；历史逐阶段记录完整保存在 [PROGRESS.md](PROGRESS.md)。

## 当前目标
实现能实际生成、保存、重开，且地形、水体、群系一致的最小世界模型。

当前阶段：Phase 6.0 基线收口，随后开始 Phase 6.1a（统一坐标/尺度定义与宏观海陆语义）。

## 仓库与基线
- 仓库：D:\projects\minecraft-geo-world
- LADM Environment：env_59c743bd89627fe7
- 分支：master
- 最近已提交：661544d — feat: add long term ecology regions
- Phase 5.2 Minecraft Generation Profile：代码与审计已完成，正在收口提交。
- 先前稳定提交与全部阶段记录见 PROGRESS.md；不丢弃既往实验结果。

## 已完成
- Phase 1–5 算法第一版：构造/宏观高程、气候、季风、水文、侵蚀、地貌、连续地形、生态和 Minecraft 语义 Profile。
- Core 与 Loader Registry 分离；最终地形按绝对世界坐标采样。
- 2026-09-29 重跑 go test ./... -count=1 通过。
- 4 Seed / 640×400 Profile audit 通过，Unknown=0，平均置信度 0.692–0.755。
- 静态原型参数与生成/保存按钮仍未接入真实生成器。

## 当前已知问题
1. LandMask 仍在多个 adapter 中直接取自地壳类型，与实际淹水/出露不一致。
2. 旧河网与最终连续地形有逆向高差；河床、水位与湖面约束未完成。
3. 世界范围与网格精度尚未统一，仍有固定 Cell 尺度。
4. 统一 WorldModel、版本化保存/重开、Java 采样和游戏内生成未完成。
5. Phase 1–5 的比例与连续性审计不能视为完整世界验收。

诊断数据、口径和完整验收要求见 [PHASE6.md](PHASE6.md)。

## 本轮实施顺序
1. 固化 Phase 5.2 与调整后的规划。
2. 实现世界度量/海平面契约，分离地壳与宏观陆海判定，迁移各 adapter。
3. 完成针对性与集成回归，更新本文件并提交。
4. 继续迁移旧空间参数，然后统一生成入口与最终水文/海岸一致性。
5. 最小 Generator 存取闭环后，提前验证 Go -> Java -> 单 Loader。

## 工作边界
暂不增加新的地貌类别或孤立 Lab 页面；暂不同时做多个 Loader。宏观海陆修正不等于已经完成最终连续海岸与河湖校正。每个提交明确写出剩余事项。

## 继续工作入口
- ROADMAP.md：主路线。
- PHASE6.md：具体步骤、评审数据、验收标准。
- DECISIONS.md：不可丢失的设计理由。
- ARCHITECTURE.md：模型/运行时关系。
- PROGRESS.md、PHASE1.md–PHASE5.md：历史算法与验收记录。
