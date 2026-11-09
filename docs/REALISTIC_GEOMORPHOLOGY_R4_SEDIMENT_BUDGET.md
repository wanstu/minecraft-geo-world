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

## 10. 第三切片：连续 channel load

第一版 routed budget 的问题是：

```text
deposition fraction
=
(1 - transport capacity)
× accommodation
```

它对当前 cell 的**全部 available load**重复执行。

所以即使一条河的 capacity 长距离保持稳定，只要每格有少量 deposition fraction，carried load 也会沿程指数衰减。

### 10.1 定向失败样例

新增 40 个 macro cell 的稳定 channel：

- 无 Lake；
- 无明显冲积空间；
- RiverStrength / FluvialPower / slope 基本稳定；
- sediment 只在最上游产生。

旧算法：

```text
source production = 0.1000
outlet export     = 0.0437
retained fraction = 43.7%
```

即使没有明确 sediment sink，仍有超过一半 carried load 因“经过了很多 cell”而消失在沿途沉积中。

### 10.2 新路由规则

现在区分：

**Local production**

刚刚在当前 cell 侵蚀产生的 sediment 允许发生 near-source storage：

```text
local deposition
=
local production
× local storage fraction
```

**Incoming carried load**

已经在 FlowTo 网络中运输的 sediment 不再重复支付同一份本地 storage。

它只有在：

```text
upstream transport capacity
>
current transport capacity
```

并且当前存在 accommodation 时才发生明显沉积：

```text
capacity drop
× accommodation
→ incoming deposition
```

为了在多上游汇入时保持确定性，路由同时携带：

```text
pending sediment mass
pending mass-weighted upstream capacity
```

当前 cell 得到的 upstream capacity 是实际 incoming load 的质量加权值。

### 10.3 Lake

Lake 继续对完整：

```text
incoming load + local production
```

执行强 sediment trap。

仍保留部分 through-flow，因此它是可解释的 routing proxy，而不是“湖泊永久吞掉 100% sediment”。

### 10.4 定向回归

新增并通过：

**Stable channel continuity**

稳定 capacity、无明显 sink 的长河必须把至少 70% 的 source load 送达 outlet。

旧算法该样例只有 43.7%。

**Capacity-drop deposition**

相同上游 source 下：

- stable channel 的对照 cell 不应凭空沉积 incoming load；
- 当 slope / fluvial power 明显下降，同时 floodplain / alluvial accommodation 增加时，该位置必须形成 routed deposition；
- outlet load 相应下降。

**Lake trap**

原有 Lake sediment trap 回归继续通过。

**Mass conservation**

全局、hillslope、network 三层继续守恒。

## 11. 标准 Seed 结果

### Seed 123456

第一版 budget → continuous-load budget：

```text
total deposited fraction:
  82.56% → 27.99%

total export fraction:
  17.44% → 72.01%

hillslope → network delivery:
  12.90% → 39.11%

network deposited fraction:
  84.29% → 20.13%

network export fraction:
  15.70% → 79.86%

lake share of total deposition:
  0.94% → 23.04%

network deposition-weighted accommodation:
  0.374 → 0.830

network deposition in accommodation < 0.15:
  9.61% → 1.36%
```

同时：

```text
global mass-balance error    ≈ floating-point zero
hillslope mass-balance error ≈ floating-point zero
network mass-balance error   ≈ floating-point zero

internal rise >= 0.25 block = 0
internal rise >= 1 block    = 0
internal confluence rise    = 0
mouth rise                  = 0
mouth narrowing             = 0
```

### 额外固定 Seed

```text
Seed -4852173505788415804

hillslope delivery = 34.61%
network deposited = 12.70%
network export    = 87.30%
low-accommodation network deposition = 5.01%
guards = clean


Seed -563659577384754347

hillslope delivery = 39.35%
network deposited = 9.97%
network export    = 89.97%
low-accommodation network deposition = 6.25%

该 Seed 原本已有一个 <1 block 内部局部 rise；
本切片没有新增 >=1 block、汇流或河口问题。


Seed 4034658472718554561

hillslope delivery = 47.32%
network deposited = 22.43%
network export    = 77.57%
low-accommodation network deposition = 2.74%
guards = clean
```

四个标准 Seed 都表现出同一结构：

- carried load 能连续到达下游；
- network deposition 不再由“经过 cell 数量”主导；
- 沉积明显向高 accommodation / Lake 集中；
- 全局与分层质量守恒；
- 最终河流几何没有被这些只读 sediment facts 改变。

## 12. 当前版本边界

这批新 sediment facts 仍然只在：

- `internal/geomorph`
- `internal/geomorphaudit`

内部使用。

目前没有被：

- Terrain
- Reconcile
- Waterbed
- Runtime

消费。

因此本切片**仍不改变最终世界几何，也不提升 generation rules version**。

旧的：

- `SedimentSupply`
- `DepositionPotential`

继续承担 v1.1.19 正式 Terrain 语义。

## 13. 下一步

下一切片开始把 routed sediment 按**真实沉积位置**继续拆分：

1. mountain-front / alluvial-fan deposition；
2. floodplain deposition；
3. lake-inlet deposition；
4. terminal river / mouth delivered load；
5. 后续再与 Waterbed delta / estuary 条件关联。

在这些位置级事实稳定前，不直接用新 budget 改 Terrain。

目标仍然是：

> 先建立可解释、守恒的物质链，再让它影响地貌；不能为了“看起来像沉积”继续堆权重。


## 14. 第四切片：沉积位置与 receiver load

连续 channel load 稳定后，R4 继续把**山坡储存**和**河网沉积**严格分开。

沉积分区现在是：

```text
total deposition
=
hillslope near-source storage
+ network fan deposition
+ network floodplain/alluvial-plain deposition
+ network lake deposition
+ other network deposition
```

其中 fan / floodplain / lake 只对 drainage-network cell 分类。

这避免把山坡近源储存误读成“冲积平原沉积”。

### Seed 123456 / 320 × 200

```text
total deposition        = 16.6092

hillslope storage       = 11.3410
network deposition      = 5.2682

network partition:

alluvial-fan deposition = 0.0132   (0.25%)
floodplain deposition   = 0.6211   (11.79%)
lake deposition         = 3.8261   (72.63%)
other network           = 0.8079   (15.33%)

partition error          ≈ 4.8e-14
```

### Lake inlet

```text
lake inlet delivered load = 4.3905
lake inlet transitions     = 32
```

这是进入 Lake 前的 carried load，不等同于 Lake 内最终 deposition。

### Ocean receiver

所有直接进入 Ocean cell 的 terminal export 继续分成：

- network → ocean；
- hillslope → ocean。

标准 Seed：

```text
network → ocean load        = 20.9011
network → ocean transitions = 64

hillslope → ocean load        = 21.8320
hillslope → ocean transitions = 1442
```

只有前者可以作为**河口 delivered sediment load**。

后者是沿海坡面 / 小尺度直接入海的 sediment，不应拿去驱动 Waterbed delta。

64 个 network → ocean transition 与该标准世界的 OceanMouth 量级一致，说明 routed budget 已经能提供稳定的河口来沙事实。

### 当前结论

目前：

- Lake sediment trap 已经形成明确主导沉积；
- Floodplain / alluvial plain 已经获得一部分 routed deposition；
- 河口已经有可直接消费的 network-delivered load；
- **Alluvial fan routed deposition 仅占 network deposition 的约 0.25%，明显偏弱。**

因此下一切片优先审计并修正：

```text
mountain sediment source
→ foothill slope break
→ channel capacity drop
→ fan-core deposition
→ lateral fan footprint
```

在 fan deposition 有真实 upstream load 与质量守恒证据前，不用新 sediment budget 直接抬升 Terrain。


## 15. 第五切片：山前冲积扇 competence loss

位置分类后，Seed `123456` 的 network fan deposition 只占约 `0.25%`。

第一反应不能是“把 fan 权重乘大”，所以先审计 fan core 自身。

### 15.1 根因证据

标准世界原始 fan-core 统计：

```text
fan core cells                 = 12
network fan core cells         = 12
fan-core available load        = 11.7351
fan-core deposition            = 0.0623
fan-core deposition fraction   = 0.53%

mean scalar capacity drop      = 0.0097
load-weighted capacity drop    = 0.0073
```

结论：

- fan core 并不缺；
- 12 个 fan core 全部落在 drainage network；
- routed sediment load 也非常充足；
- 真正缺的是 transport model 对山前坡折的 **competence loss** 响应。

原因是当前 scalar `TransportCapacity` 同时包含：

- FlowAccumulation；
- RiverStrength；
- FluvialPower；
- slope。

河流从陡峭山地进入山前时，slope 会骤降，但 discharge / accumulation 同时增大，最终 scalar capacity 可能几乎不降。

因此真实的山前坡折被数学上“抵消”了。

### 15.2 定向失败回归

新增：

```text
TestAlluvialFanCoreDepositsRoutedLoadAcrossMountainFrontSlopeBreak
```

合成场景：

- 上游为山地河；
- 真实 TotalErosion 产生 sediment；
- routed load 到达山前；
- target 有强 slope break；
- target 被 Geomorph 明确认定为 AlluvialFanCore；
- 但增长的 FlowAccumulation 让 scalar capacity 没有下降。

修复前：

```text
fanCore     = 0.690
incoming    = 0.09486
deposition  = 0
fraction    = 0%
```

即：

> 模型已经知道“这里是冲积扇核心”，也有真实来沙，却完全没有 routed deposition。

### 15.3 SedimentCompetenceLoss

新增只读事实：

```text
SedimentCompetenceLoss
```

普通河段：

```text
competence loss
=
max(upstream transport capacity - current capacity, 0)
```

山前 fan core 额外提供：

```text
fan competence loss
=
0.28 × AlluvialFanCore
```

最终：

```text
SedimentCompetenceLoss
=
max(capacity drop, fan competence loss)
```

然后仍然由：

```text
incoming carried load
× competence loss
× accommodation
```

决定 routed deposition。

因此：

- 不凭空增加 sediment；
- 没有 upstream load 时不会制造 fan deposition；
- 没有 accommodation 时不会大量沉积；
- 只是在已被独立 Geomorph 证据确认的山前坡折补上 bed-load competence loss。

### 15.4 Seed 123456

修复前 → 修复后：

```text
fan-core deposition fraction:
  0.53% → 1.42%

network fan deposition share:
  0.25% → 0.61%

network floodplain share:
  11.79% → 16.12%

network lake share:
  72.63% → 66.66%

network export:
  79.86% → 79.30%
```

修复后：

```text
fan-core available load              = 11.4893
fan-core deposition                  = 0.1626
mean scalar capacity drop            = 0.0097
mean competence loss                 = 0.0251
load-weighted scalar capacity drop   = 0.0074
load-weighted competence loss        = 0.0231
```

仍然保持：

```text
internal rise >= 0.25 block = 0
internal rise >= 1 block    = 0
internal confluence rise    = 0
mouth rise                  = 0
mouth narrowing             = 0
```

### 15.5 额外固定 Seed

```text
Seed -4852173505788415804

fan cores                    = 21
fan-core deposition fraction = 1.90%
network fan share            = 1.51%
network export               = 86.20%
guards                       = clean


Seed -563659577384754347

fan cores                    = 18
fan-core deposition fraction = 1.83%
network fan share            = 0.77%
network export               = 89.25%

原有一个 <1 block 内部局部 rise；
没有新增 >=1 block / 汇流 / 河口问题。


Seed 4034658472718554561

fan cores                    = 92
fan-core deposition fraction = 2.57%
network fan share            = 4.68%
network export               = 71.82%
guards                       = clean
```

fan 最多的 Seed 也仍有超过 70% 的 network load 继续向下游，因此 `0.28` 没有把山前变成 sediment black hole。

### 15.6 当前结论

这一切片解决的是**因果缺口**，不是追求某个固定 fan 百分比。

现在已经形成：

```text
mountain erosion
→ routed sediment
→ mountain-front slope-break semantics
→ competence loss
→ accommodation-controlled fan deposition
```

同时 stable-channel continuity、普通 capacity-drop、Lake trap 与质量守恒回归均保持通过。

这些 routed sediment facts 仍未接入 Terrain / Waterbed，因此仍不改变最终世界几何，也不提升 generation rules version。
