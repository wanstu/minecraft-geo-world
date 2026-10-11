# GeoWorld 露头矿体深度与客户端生成卡顿（2026-10-11）

## 实机问题
Seed `-563659577384754347`，用户在 X≈2855,Y217,Z-3289 看到铁矿大片铺在表面，但向下挖只是一层：虽然 2026-10-11 早些时候放宽了浅层矿脉椭球长度与宽度，`GeoWorldMineralVeins.populate` 仍通过 `isExposedRockFace()` 要求每个最终矿石体素本身必须暴露到空气。内部 STONE 因而被全部拒绝，这是直接根因。

HMCL `26.3/logs/latest.log` 20:28:18 前的新存档 t27：
- seed `-563659577384754347`，2304 区块累计均值：model=0.000ms、shoreline=6.504ms、biomes=42.330ms、voxel=12.895ms、minerals=3.039ms、geodes=0.010ms；
- `Can't keep up!` 两次：6893ms/137 ticks、10556ms/211 ticks；
- 没有 OOM 或 GeoWorld stacktrace；Mojang/Realms 401 网络异常暂不能认定与区块生成相关。
- 注意累计每区块平均数是不同区块/不同线程负载下的计时，不等于每个区块响应时延，且已包含与其他线程争用 CPU 的墙上时间。与此前 t26 的 biomes=48.505ms、minerals=0.776ms 只能作为方向参考。

## 修复
- 新版地表露头矿体继续使用 Seed 及地质条件确定的长轴/半径，但中心深入真实岩石表面约4格，专用椭球纵向半径约为横向半径的1.75倍，地表下允许填充最多13格。
- 先确认这一整根地表列顶端确实为原版 STONE、Mountain/Foothills、无水、无 shoreline override；然后矿脉内部凡实际为 STONE 的位置也可替换为矿石。**不要求每个矿石块自己暴露在空气中**。草、雪、沙、石上结构不覆盖；矿体大部分埋在石头里，露头是矿脉的入口。
- 去掉原有外部邻居 `GeoWorldShoreline.plan()` 逐矿块检查（大量 repeated model calls）。使用每区块 256 字节岩面有效性缓存、一个 384×256 bit 的体素去重 bitmap 防止连续球段对同一块反复进行 `getBlockState`。
- 地下8类主世界矿石和深板岩变种、紫水晶晶洞保持原计划，GeoWorld 以外普通世界不受影响。
- 客户端群系计算继续偏慢；最小等价优化：复用 `nearOcean(...96)` 的布尔结果，避免温带森林/草原蘑菇候选和 regionTags 都扫描同一圈8个模型邻域。保持每点生成群系不变，用三维 fingerprint 严格验证。

## 验证边界
- `verifyGeoWorldSurfaceOutcrops` 对实际 Seed 指定高山区域统计矿石表面与地下纵深，同一表面矿柱可延伸>=5格，且出现>=30个至少4层的矿柱；不以矿脉几何候选数冒充实际区块保存方块数。
- `fingerprintGeoWorldBiomeDecisions` 指纹强制为 123456=`9f3de994862521fb`，另一个 Seed `-563659577384754347`=`71bfd1e1968b4db6`，防止无意改变群系结果。
- 完整 NeoForge build/check 包括 56/56 主世界群系、8矿16变种、岸线和水体所有权。实机加载耗时必须等新 build 启动后再次读取日志确认，不能仅用 Java 单独基准宣称 FPS/加载时间改善。
