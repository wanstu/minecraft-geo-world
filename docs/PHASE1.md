# Phase 1 — 板块与大陆骨架

## 目标

验证 GeoWorld 最重要的第一条因果链：

```text
Seed
  ↓
板块中心
  ↓
板块划分
  ↓
板块类型 + 运动方向
  ↓
板块边界
  ↓
汇聚 / 张裂 / 转换
  ↓
大陆骨架 / 海盆 / 主要山系倾向
```

本阶段只追求“宏观结构可解释”，不追求最终 Minecraft 地貌细节。

## 当前实验模型

第一版实验使用二维有限平面。

- 网格：默认 320 × 200 个宏观 Cell
- 板块：默认 18
- 每个板块有：
  - ID
  - seed point
  - velocity vector
- Cell 通过最近板块中心归属板块
- 大陆 / 海洋地壳由独立的平滑“大陆势场”生成，并按目标海洋面积阈值切分
- 大陆势场在地图外围施加海洋偏置，给风暴洋 / 冰冻洋保留缓冲
- 相邻 Cell 属于不同板块时形成边界
- 根据两侧板块相对速度在边界法线方向的投影判断：
  - convergent
  - divergent
  - transform

这只是 Phase 1 Lab，不代表最终算法一定使用 Voronoi。

## 当前输出

`geoworld-lab` 输出：

- `plates.png`：板块 ID 分区
- `crust.png`：独立大陆 / 海洋地壳场
- `boundaries.png`：三类板块边界
- `tectonics.png`：地壳与板块边界叠加，用于检查大陆是否跨板块
- `features.png`：构造地貌分类，包括大陆碰撞山系、火山弧、海沟、洋中脊、大陆裂谷等
- `elevation.png`：基于地壳类型和构造地貌的粗略高程倾向

## 下一步算法改进

1. 让板块形状避免“标准 Voronoi 多边形感”。
2. 引入板块年龄与地壳厚度。
3. 区分大陆-大陆、大陆-海洋、海洋-海洋汇聚。
4. 对汇聚边界建立更合理的山系 / 海沟 / 岛弧模型。
5. 让大陆地壳可以跨多个相邻板块，而不是“一个板块等于一块大陆”。
6. 加入侵蚀前的基础高度场。
7. 明确主地理区与外围海洋缓冲区。

## 重要原则

板块和大陆不是同一个概念。

最终设计中：

- 一个大陆可以跨多个板块。
- 一个板块可以同时包含海洋和大陆地壳。
- 板块边界决定主要构造地貌，但不会直接等价于海岸线。

当前 Lab 已把 crust field 从 plate field 中拆开。下一步需要继续细化：大陆地壳年龄 / 厚度、板块边界两侧地壳组合，以及俯冲方向。


## 当前构造地貌规则

### 汇聚边界

- continental + continental → Collision Orogen
- continental 一侧面对 oceanic → Volcanic Arc
- oceanic 一侧面对 continental → Trench
- oceanic + oceanic → Oceanic Subduction（后续根据地壳年龄决定俯冲侧）

### 张裂边界

- oceanic + oceanic → Mid-Ocean Ridge
- 任一侧为 continental → Continental Rift

### 转换边界

- Transform Fault

这些规则仍是宏观近似，但已经让“板块边界类型”真正开始影响地貌类型，而不是只负责画一条边界线。


## Phase 1.2 完成记录

已实现：

### 海洋地壳年龄

- 海洋-海洋张裂边界作为年轻地壳源。
- 使用多源传播建立海洋地壳年龄场。
- 洋中脊附近年龄接近 0。
- 远离洋中脊年龄逐渐增大。
- 不连通小海盆使用稳定回退值，避免年龄场失效。

### 海洋-海洋俯冲

当两个海洋板块汇聚时：

- 更老的海洋地壳优先成为俯冲侧并形成 Trench。
- 更年轻的上覆板块侧形成 Island Arc。
- 年龄非常接近时使用稳定的 Plate ID 打破平局，保证同 Seed 确定性。

### 构造影响带

边界 Feature 只作为“源”。

影响通过距离衰减扩展到周围区域：

- Collision Orogen：宽强抬升
- Volcanic Arc：中等抬升
- Island Arc：海洋侧抬升
- Trench：窄强下沉
- Mid-Ocean Ridge：海洋侧弱抬升
- Continental Rift：大陆侧下沉
- Transform Fault：弱影响

当前高程已经改为：

```text
基础大陆 / 海盆高度
+ 海洋地壳年龄导致的海盆冷却下沉
+ 构造影响带
+ 少量局部噪声
= Phase 1 粗略高程
```

### 新增测试

`internal/phase1/generator_test.go`

验证：

- 确定性
- 海洋面积
- 海洋地壳年龄存在有效范围
- 构造影响范围大于边界源范围

### 新增图层

- `ocean-age.png`
- `subduction.png`
- `tectonic-influence.png`
