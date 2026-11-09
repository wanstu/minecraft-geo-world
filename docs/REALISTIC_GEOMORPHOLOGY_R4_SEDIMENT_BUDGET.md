# GeoWorld 真实地貌强化 — R4 输沙预算第一切片

更新时间：2026-11-09

状态：**只读 sediment budget 已实现，尚未改变 Terrain。**

## 1. 为什么先做预算而不是继续调 DepositionPotential

R0/R2 审计确认，旧 Geomorph 的 `SedimentSupply` 并不是实际输沙量。

旧逻辑：

```text
ReliefIndex / FluvialPower / TectonicSupport
→ 直接上游邻居一次平均
→ SedimentSupply
→ Floodplain / Alluvial / Fan 权重
→ DepositionPotential
```

主要缺口：

- Geomorph 原来拿不到 Erosion 的 `TotalErosion`。
- 没有实际产沙量。
- 没有沿完整 FlowTo 网络累计输沙。
- 没有 TransportCapacity。
- 没有 production = deposition + export 的守恒验收。

所以 R4 第一切片先建立事实层，不直接改变地形。

## 2. 新增事实

Geomorph Input 新接入：

```text
Erosion.TotalErosion
```

新增 Model 字段：

- `SedimentProduction`
- `SedimentFlux`
- `TransportCapacity`
- `SedimentDeposition`
- `SedimentExport`

现有：

- `SedimentSupply`
- `DepositionPotential`

暂时保持原语义。

因此当前 v1.1.19 World Model / Terrain 结果不因本切片发生变化，也不需要提升 generation rules。

## 3. Production

Production 使用 Erosion 阶段实际削掉的高度：

```text
production =
  max(TotalErosion, 0)
  × cellAreaScale
```

其中：

```text
cellAreaScale =
  cellArea / (128 × 128)
```

这保证改变 macro sampling resolution 时，不会因为单纯 cell 数变化把总产沙代理直接放大。

关键原则：

> Relief 很高、FluvialPower 很高，但 TotalErosion = 0 时，SedimentProduction 必须为 0。

这和旧 SedimentSupply 的启发式含义明确分开。

## 4. 沿 FlowTo 路由

使用 FlowTo DAG 的拓扑顺序：

```text
upstream production
+ incoming carried sediment
= available sediment

available
→ local deposition
→ downstream carried flux
→ terminal export
```

每一份 proxy mass 只允许：

- 在当前 cell 沉积；
- 继续向下游；
- 在终止出口 export。

不会凭空增加。

## 5. TransportCapacity

当前第一版 TransportCapacity 是 0..1 的**通过能力比例**，不是物理单位吨/年。

它由：

- FlowAccumulation
- RiverStrength
- FluvialPower
- local slope

共同决定。

高流量、高能量、较陡河段更容易继续搬运。

Lake 会显著降低 transport capacity。

## 6. Accommodation

低 capacity 不等于必须沉积。

沉积还需要本地空间：

- FloodplainWeight
- AlluvialPlainWeight
- AlluvialFanWeight
- low slope
- low relief

Lake 作为强沉积陷阱单独处理。

因此：

```text
deposition fraction
=
(1 - transport capacity)
× accommodation
```

实际实现保留少量背景储存，并限制湖泊也不能吞掉 100% 物质。

## 7. 单元验收

新增并通过：

### 无真实侵蚀不得凭空产沙

即使：

- ReliefIndex 高；
- FluvialPower 高；
- TectonicSupport 高；
- RiverStrength 高；

只要：

```text
TotalErosion = 0
```

则：

```text
Production = 0
Flux = 0
Deposition = 0
Export = 0
```

### FlowTo 质量守恒

合成河网中验证：

```text
sum(Production)
=
sum(Deposition)
+ sum(Export)
```

误差仅允许浮点误差。

### Lake sediment trap

相同上游输入下：

- Lake cell 的沉积增加。
- 流域最终 export 减少。

### 确定性

新增 sediment 字段在重复生成中逐 cell 完全一致；raw proxy 字段必须有限且非负，TransportCapacity 保持 0..1。

## 8. 标准 Seed 预算

Grid：

```text
320 × 200
cell = 128 blocks
rules = geoworld-rules-v1.1.19
```

### Seed 123456

```text
production = 59.3423
deposition = 48.9956
export     = 10.3467

deposited fraction = 0.8256
export fraction    = 0.1744

mass balance error = 3.3e-13
mean capacity      = 0.1945
lake share of deposition = 0.0094
```

同时最终河流 guard 保持：

```text
internal rise >= 0.25 block = 0
internal rise >= 1 block    = 0
internal confluence rise    = 0
mouth rise                  = 0
mouth narrowing             = 0
```

### 额外固定 Seed

```text
-4852173505788415804
  deposited = 0.7618
  export    = 0.2382
  balance error = 1.42e-13

-563659577384754347
  deposited = 0.8056
  export    = 0.1944
  balance error = 2.65e-13

4034658472718554561
  deposited = 0.7955
  export    = 0.2045
  balance error = 1.60e-14
```

当前 76%–83% 的 deposited fraction 包括山坡近源储存，并不等于“河漫滩沉积率”。

因此本阶段**不把这个比例直接接入 Terrain**。

## 9. 第二切片：hillslope / network 分层审计

在不改变 Terrain 的前提下，继续暴露：

- `SedimentAccommodation`
- hillslope production / deposition / export
- hillslope → network delivery
- network production / deposition / export
- network → hillslope transfer
- hillslope / network 各自 mass balance
- hillslope / network mean transport capacity
- hillslope / network mean accommodation
- network deposition-weighted accommodation
- low-accommodation network deposition fraction

网络定义暂时使用：

```text
RiverStrength > 0
or
LakeID > 0
```

这只是 R4 诊断分类，不改变正式 hydrology / terrain 语义。

### Seed 123456 / 320 × 200

```text
total production = 59.3423

hillslope production = 54.4735
network production   = 4.8688

hillslope → network delivery = 7.0263
hillslope delivery fraction  = 0.1290

network input fraction       = 0.2005
network deposited fraction   = 0.8429
network export fraction      = 0.1570

hillslope mean capacity      = 0.1852
network mean capacity        = 0.3612

hillslope mean accommodation = 0.2202
network mean accommodation   = 0.4148

network deposition-weighted accommodation
  = 0.3741

network deposition in accommodation < 0.15
  = 9.61% of network deposition

hillslope mass-balance error
  ≈ 1.96e-13

network mass-balance error
  ≈ 4.51e-15
```

### 结论

这组数据说明第一版 routed budget 的**守恒与分层路由是成立的**，但还不适合直接驱动 Terrain。

当前最明显的结构性风险：

1. hillslope production 只有约 12.9% 真正进入 drainage network；
2. 进入 network 后约 84.3% 又在网络内部沉积；
3. 当前 deposition 仍然在每个 cell 反复按 `(1-capacity) × accommodation` 计算；
4. 对长河而言，这会形成连续乘法衰减，容易让上游 sediment load 在到达湖口、河口前被过早耗尽。

因此下一步不能直接把 `SedimentDeposition` 接进 Terrain。

## 10. 下一切片：连续 channel load

下一步优先修正**输沙连续性**，目标是：

```text
hillslope production
→ near-source storage
→ channel delivery
→ channel carried load
→ capacity drop / accommodation-controlled deposition
→ lake / fan / mouth sink
→ terminal export
```

关键约束：

- incoming carried load 不能在每个普通河网 cell 被重复按同一比例“税收式”沉积；
- 稳定 capacity 的河段应允许 sediment load 连续通过；
- capacity 明显下降、坡降突变、湖泊入口、冲积空间增加时才应显著沉积；
- Lake 继续作为强 sediment trap；
- 全局、hillslope、network 三层都必须质量守恒；
- 在 routed facts 稳定前，旧 `SedimentSupply / DepositionPotential` 继续保持正式 Terrain 语义。

目标仍然是：

> 先建立可解释、守恒的物质链，再让它影响地貌；不能为了“看起来像沉积”继续堆权重。
