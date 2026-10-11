# 湖泊 / 海岸条件性缓坡：第一轮（2026-10-11）

## 用户反馈与边界

用户提供实机截图：一些湖海岸仍然像直立土墙或过陡斜坡，希望**在需要的地方**更平滑，但不将正常岩崖、峡谷海崖统一削平。与此同时，GeoWorld 的区块生成仍然偏慢，所有主世界原版群系、矿物和水体所有权不可回退。

## 新版 t28 基准日志

2026-10-11 20:46 左右的 HMCL v26.3 `latest.log`，Seed `-563659577384754347`、存档 `t28`，累计 1536 区块：

- `shoreline=5.623 ms`
- `biomes=39.687 ms`
- `voxel=10.386 ms`
- `minerals=0.167 ms`
- `geodes=0.007 ms`
- `Can't keep up` 两次：5250ms / 105 ticks 与 2776ms / 55 ticks。

这是改变岸线代码**之前**运行的游戏日志。不能用独立 Java 基准直接证明实际 Minecraft 加载速度。

## 修改

- `GeoWorldShoreline` 保留 `MAX_SHORE_DISTANCE=4` 的原始短沙滩、材质与河岸处理；湖/海的干燥内陆岸坡可延伸到 `MAX_GENTLE_BANK_DISTANCE=10`。
- 只在最近源水类型为 Lake/Ocean、原始地面高于该水位 4..32 格且不是真正的 Canyon / 大于24格的 Mountain 海崖时进行。天然岩石河岸并非一概豁免，中小岩岸也允许局部缓坡。**没有改变河流水面、湖海源水归属、生成新的水源或更改主世界其他机制。**
- 从水边到内陆按照 `smoothstep(d / 10)` 逐渐回归原始地表，且干岸地面不低于邻近水位及既有 Ocean 低岸稳定阈值；保留高差非常大的自然悬崖。
- 地表修正只改变 `GeoWorldShoreline.Column`，不写入 Engine 模型，原始 `groundHeight` 保持。新增 `gentleBankCap` 及阶梯单调性检查；`pointPlan` 与 `planChunk` 仍同源。
- 尺度扩展会增加附近模型采样数；不要因为独立小基准成功就宣称没有性能代价。需要进一步以客户端真实日志、相同可见区块范围对照判断是否需要优化岸线预计算。

## 验收与限制

2026-10-11 `build.cmd` 全部通过，并部署 HMCL `26.3`：

- 群系 56/56；群系指纹 `123456=9f3de994862521fb`
- 矿石、紫水晶、矿脉出露专项 PASS
- `verifyGeoWorldShoreline` 的缓坡曲线测试 PASS；窄沟、岸后小坑、海岸步阶、点位和分块计划差异均 0。
- `verifyGeoWorldWaterOwnership` 扫描 302848 列：unowned、unsafe、freeFall、runtimeLost、riverPeak、wetlandLeak 全 0。
- Java 独立模型侧 `benchmarkGeoWorldChunkPlanning`：shoreline 4.123ms、biomes 0.086ms、minerals 0.084ms。**不是客户端的整区块生成时间。**
- 当前 Seed 123456 的历史岸线审计窗口中 `gentleBankColumns=469`（含已有浅滩修整），`inlandGentleBankColumns=0`。这说明测试窗口不能作为图中高陡岸视觉改善的证据；需补高陡湖岸/海岸的实景验证坐标，必要时优化上游地形模型的宏观岸坡，而不是无限扩大 voxel 修正带宽。
- 已生成区块不会自动重做岸坡。测试使用新世界或未生成区块。

## 下一步

优先用实景坐标明确“普通湖海岸过陡”与“真实峡谷/高海崖”。如属于几十格乃至百格高差，改宏观地形分配，不宜再扩大当前 voxel 岸线 10 格修补带。并使用新版本的 `GeoWorld generated` 日志确认 shoreline 性能开销；`biomes` 仍是主要瓶颈。
