# GeoWorld R8.2 Slice C — 特殊岛屿 / 雪地貌背景取证

日期：2026-11-17
规则基线：`geoworld-rules-v1.1.24`
种子：`-1818694639628404530`

本阶段只读取持久化 World Model 的 **SpecialFeatures 应用前** Ground / Water / Terrain / Climate 背景，不修改岛形、雪山形状或其他生成算法。

## 1. Ocean island

当前固定 seed 选中 8 个 Ocean island。对每个中心及 0.5R、1.0R、1.25R、1.5R、2.0R 环带采样后：

- 8/8 中心仍是 Ocean；
- 当前选中岛的中心相对外围海床中位高度差均只有 -1～1 格，没有一个达到 8 格；
- 深海型样本海深约 55～71 格，海床在数百格范围内近乎水平；
- 浅海型 Cherry island 海深约 13～14 格，但中心相对外围也只有 0～1 格高。

因此当前 Ocean island 不是“水下山体露出水面”，而是 SpecialFeatures 在原本海床上直接抬升出岛。

全 Ocean interior 候选池共有 31,843 个候选。在 384 格环带 32/32 仍为 Ocean 的前提下：

- 水下根部抬升 ≥4 格、中心深度 ≤48 格：760 个；
- 抬升 ≥8 格、中心深度 ≤48 格：281 个；
- 抬升 ≥12 格、中心深度 ≤48 格：11 个。

因此错误选址不是因为世界里没有合适的海底高地。

## 2. Lake island

当前选中两个 Lake island：

- `(-3136,-1088)`：中心湖底 Y57，160 格外围中位 Y59，root lift = -2；
- `(704,-2880)`：中心湖底 Y56，160 格外围中位 Y56，root lift = 0。

固定 seed 只有 4 个足够大的持久化湖组；按湖心附近检查，root lift 分别为 `0, -2, -5, -12`，没有任何明显水下高地。

结论：本 seed 不具备“湖底山头露出”为 Lake island 的地理条件。v0.1.7 应允许该 seed 不生成 Lake island，而不是为了数量强制生成。

## 3. Snow crater / basin 只读观察

两个 `volcanic_crater_snow` 的山地背景成立：

- 中心 Y231 / Y260；
- MountainWeight ≈ 1；
- 512 格 relief 约 164 / 90 格。

但年均温约 14.6°C / 8.5°C，说明 Snow 主题仍没有气候资格判断。

两个 `basin_snow` 的问题更严重：

- ID18：中心 Y93，Mountain/Foothill 权重 0，512 格 relief 15 格，年均温约 -3.3°C；
- ID19：中心 Y115，Mountain/Foothill 权重 0，512 格 relief **9 格**，年均温约 **15.0°C**。

这与此前“没有雪山盆地地理条件”的视觉反馈一致。

候选池并不缺少更合理背景：例如 `temp≤5°C + 512 relief≥50 + basin depth≥4` 的陆地候选有 452 个。

Snow 的筛选属于 Slice D；本 Slice C 不同时修改它。

## 4. Slice C 实施边界

Slice C 只处理 island siting：

- Ocean island 必须有可量化的 submerged local high / root；
- moat 环带仍必须是 Ocean；
- 不通过时继续 deterministic candidate 序列；
- Lake island 若没有真实湖底高点则允许 0 个；
- 不修改 RiverGeometry、Waterbed、海面、湖面或岛屿 Apply 几何。

Snow basin / crater 留给 Slice D 单独处理。

证据：

- `docs/acceptance/v017_r82_slicec_background_baseline.json`
- `docs/acceptance/v017_r82_selected_feature_background.json`
