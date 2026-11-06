# GeoWorld Java Runtime Reader

这是 GeoWorld World Model V1 的纯 Java reader/sampler。

当前边界：

- 不依赖 Fabric / NeoForge / Forge。
- 不依赖 Jackson / Gson / Gradle / Maven。
- 仅使用 Java 标准库。
- 以 Java 21 作为最低编译目标。
- 读取 .geoworld V1（gzip+json）。
- 校验 manifest、schema/generator version、payload bytes 和 SHA-256。
- 按 Go encoding/json 规则解码持久化的 []uint8 Base64 字段。
- 恢复 CoastTerrain、River/Lake Waterbed、WorldSample 和 Runtime Profile。
- 对任意世界坐标 x,z 提供最终 Runtime Sample。

## Golden 验证

在仓库根目录运行：

    java\runtime-reader\verify-golden.cmd

该命令会：

1. 用 javac --release 21 编译 reader；
2. 加载 internal/goldenvectors/testdata/runtime-v1.geoworld；
3. 读取 runtime-v1.json；
4. 对 18 个 cross-language golden cases 逐字段比较；
5. 浮点字段使用 golden 文件中的 floatTolerance，离散字段要求完全一致。

当前 golden 覆盖：

- 四角 / 四边；
- 负 X/Z Minecraft Chunk 边界；
- 世界坐标原点；
- Macro node / Macro cell interior；
- mixed coast；
- River backwater；
- Lake center；
- 有限模型之外的远洋、极地海洋与稀疏海岛。

验证全部已提交的新旧模型：

    pwsh -NoProfile -File java/runtime-reader/verify-golden.ps1 -All

`-All` 会编译一次 reader，然后对 `testdata/runtime-*.json` 及其同名 `.geoworld` 逐组验证。当前为 6 个模型、108 个样点；任一模型失败立即返回非零退出码。CI 在适配器构建和发布前执行此检查。指定 `-Model` / `-Golden` 的单模型验证方式保持可用，不能与 `-All` 同时使用。

## 设计目的

这一层不是 Minecraft Loader，也不是世界存档 writer。

它的职责只有一个：

    Go World Model V1
           ↓
    Java V1 Reader
           ↓
    Runtime sample(x,z)

下一层的 Minecraft 存档生成器或 Loader 只依赖这个 Java Runtime 接口，不复制 Go 全球生成算法。
