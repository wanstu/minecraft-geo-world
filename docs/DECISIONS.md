# GeoWorld 关键设计决定

> 记录“为什么这样做”。这些决定不是临时聊天结论，后续修改时应明确更新原因。

## D-001：GeoWorld 同时是工具与 Runtime

状态：已确定

GeoWorld Core / World Model 是核心。

外部形态：

- Generator：离线生成、分析、导出。
- Runtime：Minecraft 按需生成。

原因：世界模拟和 Minecraft 接入属于不同问题，不应绑死。

---

## D-002：不绑定单一 Mod Loader

状态：已确定

Loader 只是 Minecraft Adapter。

目标结构：

```text
GeoWorld Model
      ↓
Minecraft Common
 ┌────┼────┐
Neo Fabric Forge
```

---

## D-003：V1 不做球形世界

状态：已确定

Minecraft 保持无限平面坐标。

中央有限区域承担完整地理模拟；外围使用无限特殊海洋。

原因：

- 避免球面投影、极点、首尾接缝复杂度。
- 板块 / 气候模拟仍然可以在有限平面中成立。
- 保持 Minecraft 原始坐标和区块模型。

---

## D-004：外围世界规则

状态：已确定

- 东 / 西：Infinite Storm Ocean。
- 北 / 南：Infinite Frozen Ocean。
- 四角：两种环境自然叠加。
- 外围不是硬 World Border，玩家仍能继续探索。
- 主地理区边缘必须留海洋缓冲，避免大陆直接撞“世界尽头”。

---

## D-005：GeoWorld 生成气候，不模拟实时天气

状态：已确定

GeoWorld 可以计算夏 / 冬甚至 12 个月的气候统计值，但 V1 只输出长期气候和生态。

不负责：

- 四季玩法
- 当前月份
- 每日天气
- 实时区域风暴

未来独立季节 / 天气 Mod 可读取 GeoWorld API。

---

## D-006：世界生成配置创建后不可修改

状态：已确定

影响以下内容的参数只能在创建新世界时设置：

- Seed
- 世界地理规模
- 板块参数
- 海陆比例
- 气候模型
- 水文模型
- 世界外围规则

创建后写入世界自己的 GeoWorld metadata。

已有世界继续生成新区块时必须读取该世界自己的配置，而不是全局默认值。

原因：防止旧区块 / 新区块发生不可逆割裂。

---

## D-007：大陆与板块必须解耦

状态：已实现

错误模型：

```text
一个板块 = 一块大陆 / 一片海洋
```

正确方向：

- 大陆可以跨多个板块。
- 板块可以同时包含海陆地壳。
- 海岸线不等于板块边界。

---

## D-008：噪声不能决定宏观地理

状态：已确定

噪声可以用于：

- 海岸细节
- 山体细节
- 局部高度变化
- 方块级自然感

噪声不负责：

- 大陆位置
- 山系主轴
- 海沟
- 洋中脊
- 主河流骨架
- 气候区

宏观结构必须来自可解释的地理因果链。

---

## D-009：板块构造应该产生“带”，不是一格线

状态：已实现第一版

真实造山带、裂谷、海沟、岛弧都有宽度。

因此边界分类只作为构造源，后续要生成影响场：

```text
boundary source
      ↓
distance falloff
      ↓
tectonic influence band
      ↓
elevation / trench / ridge / volcanism
```

---

## D-010：海洋地壳年龄决定海洋俯冲倾向

状态：已实现

洋中脊生成年轻海洋地壳。

远离洋中脊：

- 地壳逐渐变老。
- 温度降低。
- 密度增加。

海洋-海洋汇聚时，优先让较老的一侧俯冲。

这将用于决定：

- 海沟侧
- 岛弧侧
- 俯冲方向


---

## D-011：构造边界是“源”，地貌必须形成有宽度的影响带

状态：已实现第一版

不再直接把边界 Cell 当成山脉 / 海沟本身。

流程：

```text
Plate Boundary
      ↓
Tectonic Feature Source
      ↓
Distance Falloff
      ↓
Tectonic Influence Band
      ↓
Elevation Field
```

不同构造类型使用不同半径和幅度：

- Collision Orogen：宽、强抬升
- Volcanic Arc：中等宽度抬升
- Island Arc：海洋侧局部抬升
- Trench：窄而强的下沉
- Mid-Ocean Ridge：较宽弱抬升
- Continental Rift：较宽下沉
- Transform Fault：窄且弱

带宽和幅度已经升级为由相对板块速度、地壳厚度、地壳年龄等共同决定；后续仍会继续校准物理关系。


---

## D-012：板块自然化采用连续坐标扭曲，不做逐 Cell 随机扰动

状态：已实现第一版

原始 Voronoi 的优点是：

- 板块连续
- 拓扑简单
- 计算便宜
- 易于解释

缺点是边界过于笔直、几何感明显。

V1 自然化方案：

```text
World Coordinate
      ↓
Low-frequency Smooth Domain Warp
      ↓
Warped Coordinate
      ↓
Nearest Plate Seed
```

不采用：

```text
每个 Cell 独立随机修改 plate id
```

原因是后者容易产生：

- 孤立 Cell
- 板块碎片
- 锯齿边界
- 不稳定的构造带

原始 `RawPlateID` 保留用于调试与算法比较，实际构造计算使用自然化后的 `PlateID`。


---

## D-013：构造强度不能只由“边界类型”决定

状态：已实现第一版

同样是大陆碰撞，慢速轻微挤压与高速强烈碰撞不应产生一样的山脉。

因此 GeoWorld 使用：

```text
Relative Plate Velocity
        ↓
Boundary Strength
        +
Crust Thickness / Ocean Age
        ↓
Tectonic Intensity
        ↓
Landform Width + Elevation Amplitude
```

这让地貌类型和地貌强度分离：

- Feature 决定“是什么”。
- Intensity 决定“有多强”。


---

## D-014：基础海陆高程与构造影响必须分层

状态：已实现第一版

不能把“大陆架、深海盆地”和“山脉、海沟”混成一个公式直接生成。

GeoWorld 采用：

```text
Base Geography
  ├─ Coast Distance
  ├─ Continental Crust Thickness
  └─ Ocean Crust Age
          ↓
     Base Elevation
          +
Tectonic Influence
          ↓
   Final Macro Elevation
```

原因：

- 大陆架 / 大陆坡主要是大陆边缘几何，不应依赖是否正好有板块边界。
- 山脉 / 海沟属于构造异常，应叠加在基础地形之上。
- 后续气候、水文和 Minecraft 细化可以分别读取 BaseElevation、TectonicInfluence 和最终 Elevation。


---

## D-015：丘陵属于 Relief / 侵蚀结果，不属于宏观随机噪声

状态：已确定

GeoWorld 不采用：

```text
noise > threshold → hills
```

丘陵应来自中等尺度起伏：

```text
Tectonic Background
+ Distance to Mountain Belt
+ River Incision
+ Precipitation
+ Terrain Age
+ Erosion Resistance Approximation
        ↓
Relief
        ↓
Hills / Foothills / Valleys / Rolling Plains
```

因此 Phase 1 只建立宏观高程骨架。

最终丘陵放到气候与水文之后的侵蚀 / Relief 阶段生成。

Elevation 与 Relief 必须分离：

- Elevation：这里整体有多高。
- Relief：周围一定尺度内起伏有多强。

高海拔 + 低 Relief 可以是高原；中等海拔 + 中 Relief 才可能是丘陵。

---

## D-016：Minecraft Chunk 不是地形数学边界

状态：已确定

最终地形必须定义为连续世界坐标函数：

```text
height = H(worldX, worldZ)
```

Chunk 只负责采样其覆盖的世界坐标。

禁止：

```text
generateChunk(cx, cz) → 独立随机地形
```

原因：

- 独立 Chunk 生成容易产生接缝。
- 世界坐标采样天然保证相邻 Chunk 使用同一函数。
- 相同坐标无论从哪个 Chunk 请求，都得到相同结果。

最终高度建议分层：

```text
BaseElevation
+ TectonicRelief
+ ErosionRelief
+ RegionalDetail
+ LocalDetail
= H(worldX, worldZ)
```

宏观 Grid 只是采样点，不是方块高度台阶。

从宏观场到 Minecraft 坐标时必须使用连续插值；初期可采用双线性，正式方案优先考虑受限 Hermite / 单调 cubic 等不会产生明显 overshoot 的平滑方法。

---

## D-017：地貌边界使用连续权重混合

状态：已确定

平原、丘陵、山麓、山地等不能按分类边界直接切换地形函数。

应使用连续权重：

```text
Terrain =
  Plain * plainWeight
+ Hills * hillsWeight
+ Mountain * mountainWeight
```

权重随位置连续变化且总和归一化。

这样可以避免：

- 群系墙
- 地形突然变陡
- 宏观 Cell 边缘台阶
- Chunk 边界视觉割裂


---

## D-018：外围海洋缓冲是硬约束，不只是概率偏置

状态：已实现

仅靠“越靠近边缘越不容易生成大陆”的软偏置，极端 Seed 仍可能让陆地碰到模拟边界。

因此 V1 同时采用：

```text
Hard Ocean Guard Band
+
Soft Ocean Edge Bias
```

默认 `EdgeOceanBufferFraction = 0.04`。

硬缓冲保证外围一定为海洋；软偏置保证缓冲区内侧的大陆轮廓仍然自然退入海洋，而不是沿矩形边界被切断。

大陆 Cell 数从合法候选区中精确选取，因此目标 OceanFraction 不因硬缓冲而漂移。
