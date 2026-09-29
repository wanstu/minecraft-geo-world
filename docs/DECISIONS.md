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

状态：进行中

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

后续会把带宽和幅度从固定常量升级为由相对板块速度、地壳厚度、地壳年龄等决定。
