# GeoWorld Runtime Golden Vectors

本目录为跨语言 Runtime Reader / Sampler 提供固定对拍基线，不是普通 Lab 输出。

## 固定产物

- `testdata/runtime-v1.geoworld`：正式 `worldmodelio` V1 模型文件。
- `testdata/runtime-v1.json`：固定坐标与期望 Runtime 输出。

JSON 自身记录：

- golden format / version；
- World Model schema / generator version；
- 浮点比较容差；
- 固定 Seed 与 Geometry；
- 生成参数；
- 每个命名样点的坐标与完整 Runtime Sample。

当前 V1 样点覆盖：

- 四个有限世界角；
- 四条外围边；
- 负 X/Z 的 Minecraft 16-block Chunk 边界；
- 世界坐标原点；
- Macro Grid 精确节点与 Cell 内部；
- mixed coast；
- 带 receiver surface 的正式 River 下游段；
- 正式 Lake 节点中心。

## 跨语言验收

Java Reader / Sampler 应：

1. 读取 `runtime-v1.geoworld`，遵守 manifest 的 schema / generator version / checksum。
2. 对 `runtime-v1.json` 中每个 `x,z` 调用 Runtime 采样。
3. 枚举、布尔、来源节点和字符串字段必须完全一致。
4. 浮点字段按 JSON 的 `floatTolerance` 比较。
5. 不要用 Java 重新运行全球生成算法来“得到相近结果”；golden 的目标是验证**读取已保存模型并重建 Runtime Sampler**的一致性。

## 有意更新

只有模型格式或 Runtime 语义有意改变时才重新生成：

```
go run ./cmd/geoworld-golden-vectors -force
go test ./internal/goldenvectors -count=1
```

不要手工修改 JSON 数值来让测试通过。生成命令会从固定 Fixture 重新生成 `.geoworld` 和 JSON。
