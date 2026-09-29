# GeoWorld 连续地形采样

## 目标

Minecraft Chunk 不能成为地形数学边界。

最终 Runtime 应只做：

```text
for each block (worldX, worldZ):
    height = H(worldX, worldZ)
```

无论该坐标由哪个 Chunk 请求，结果都必须一致。

## 当前第一版

实现位置：

`internal/sampling/grid.go`

当前使用：

**Bilinear Interpolation / 双线性插值**

优点：

- 连续。
- 不会 overshoot。
- 计算便宜。
- 与 Chunk 无关。
- 非常适合第一版 Runtime 和算法验证。

当前接口：

```text
ScalarGrid
├─ Width / Height
├─ CellSize
├─ OriginX / OriginZ
├─ Values
└─ SampleBilinear(worldX, worldZ)
```

宏观 Grid Node 只是采样点，不代表一整块 Minecraft 区域使用同一个高度。

## 当前自动测试

验证：

1. 宏观 Grid 节点采样值精确一致。
2. 双线性插值不超出四个相邻节点的数值范围。
3. 穿过宏观 Cell 边界时连续。
4. 穿过 Minecraft 16-block Chunk 边界时没有特殊跳变。
5. 同一个世界坐标不会因为“从哪个 Chunk 查询”而改变。

## 后续升级

双线性虽然连续，但一阶坡度在宏观 Cell 边界可能改变。

正式地形细化阶段可以升级为：

- monotonic cubic
- constrained Hermite
- 其他无 overshoot 的 C1 平滑插值

要求：

- 保持同一个 `Sample(worldX, worldZ)` 抽象。
- 不允许出现超出邻域高程范围的插值尖峰。
- 不允许使用 Chunk-local 随机状态。
- 地貌混合权重也必须通过世界坐标连续采样。

## 最终高度层级

建议最终 Runtime 高度：

```text
H(x,z)
 =
MacroBaseElevation
+ TectonicRelief
+ ErosionRelief
+ RegionalDetail
+ LocalDetail
```

其中 LocalDetail 可以使用连续噪声，但只负责小尺度自然感，不负责大陆、山系或丘陵的宏观存在性。
