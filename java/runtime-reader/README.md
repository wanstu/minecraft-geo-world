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

## Query API

`geoworld-query-v1` 已于 2026-11-07 完成字段、门面、双目标完整检查与实机世界切换验收。GeoWorld 提供锁定自然模型的事实与倾向评分，玩法由使用它的模组决定：

| 使用场景 | 可读取的事实 | 由调用方决定 |
|---|---|---|
| 季节 / 天气 | `ANNUAL_PRECIP_MM`、`MONSOON_INDEX`、`WIND_X / WIND_Z` | 季节变化与降雨 |
| 定居点 | `SETTLEMENT_SUITABILITY`、`FRESH_WATER_ACCESS`、`FLOOD_RISK` | 建村、交易和奖励 |
| 矿工玩法 | `ORE_PROVINCE`、`MINING_DISTRICT_CLASS`、`DOMINANT_MINERAL_RESOURCE` | 矿区玩法与奖励 |

### 入口与兼容性

在编译期依赖对应 Minecraft / Loader 的 GeoWorld 模组 JAR。当前没有独立 API JAR、Maven 坐标或 SPI 分发层。

- 唯一读取入口：`geoworld.runtime.api.GeoWorldFacade`。目前 `apiVersion()` 和 `minSupportedApiVersion()` 均返回 `geoworld-query-v1`。
- `currentWorld()` 返回 `Optional<WorldFacts>`；`WorldFacts` 只提供 `queryNatural(QueryRequest)`、`supports(NaturalField)` 与 `worldProfileVersion() / generationRulesVersion() / worldModelVersion()`。
- `geoworld.runtime.api.*` 承诺契约稳定。v1 同时保留 `GeoWorldQueryService` 的四个嵌套值类型作为稳定契约：`NaturalField`、`Resolution`、`QueryRequest`、`NaturalResult`。可以引用这四个类型；服务实现、模型本体和绑定钩子属于内部实现。
- 兼容性按字段行为判断。单位、取值域、缺失条件、字段名称或量化语义发生破坏性变化时必须提升 API 版本，详见 [D-049](../../docs/DECISIONS.md)。

### 可用范围与缺失值

门面只在**逻辑服务端**提供当前动态 GeoWorld **主世界**的自然事实。普通世界、静态 QA Preview、尚未就绪或已经关闭的世界返回 empty。下界与末地没有独立的 GeoWorld 事实模型；v1 没有客户端调用或联网同步 API。

`currentWorld()` 可在任意线程调用，只读取原子快照，不采样、不阻塞。返回的 `WorldFacts` 是一个锁定模型的不可变视图；切换世界后旧视图仍代表旧模型，需要当前世界时应重新取得视图。`queryNatural` 可以并发调用，结果映射不可变；查询只读取模型，不加载/生成区块，不修改存档或玩法状态。

`supports(field)` 是该模型的能力探测，在世界绑定时缓存；它不保证某个坐标有该字段。每次查询仍须检查 `unavailableFields()`。缺失字段不会用 null 或 0 补齐；旧世界保留自己的模型与生成规则版本。相同锁定模型与相同请求的结果保持一致，不能仅凭 Seed 假定不同配置或生成规则下的世界相同。

下面的 Java 21 示例只请求年降水量，并处理未就绪、模型不支持和坐标缺失三种情况：

```java
import geoworld.runtime.api.GeoWorldFacade;
import geoworld.runtime.api.WorldFacts;
import geoworld.runtime.GeoWorldQueryService.NaturalField;
import geoworld.runtime.GeoWorldQueryService.NaturalResult;
import geoworld.runtime.GeoWorldQueryService.QueryRequest;
import geoworld.runtime.GeoWorldQueryService.Resolution;
import java.util.OptionalDouble;
import java.util.Set;

public final class GeoWorldQueryExample {
    public static OptionalDouble annualPrecipMm(double x, double z) {
        WorldFacts facts = GeoWorldFacade.currentWorld().orElse(null);
        NaturalField field = NaturalField.ANNUAL_PRECIP_MM;
        if (facts == null || !facts.supports(field)) {
            return OptionalDouble.empty();
        }
        NaturalResult result = facts.queryNatural(
            new QueryRequest(x, z, Resolution.BLOCK_4, Set.of(field))
        );
        if (result.unavailableFields().contains(field)) {
            return OptionalDouble.empty();
        }
        return OptionalDouble.of(((Number) result.values().get(field)).doubleValue());
    }
}
```

### 解释结果与查询成本

- `ELEVATION` 等高程值是归一化模型高程，不能直接当作 Minecraft 方块 Y；风场和降水是年均 / 年度气候背景，不能当作当前天气。
- 定居、航运、矿业等分数是自然条件的启发式倾向，不保证现实区块可建造、有指定矿块或已有玩法事件。
- 当前 Query 矿业评分仍使用 8 类原版矿物家族；运行时生成可以包含模组矿物，二者范围不同。字段单位、取值域与坐标缺失条件以 [FACT_REGISTRY](../../docs/FACT_REGISTRY.md) 为准。
- 按需请求字段，同组字段合并一次查询并复用结果。避免每 tick 请求全字段；区域聚合等昂贵计算应在工作线程执行，`REGIONAL_REGION_*` 使用 `BLOCK_16`。
- `queryRegion / queryNearest / queryScan` 尚未实现。Local Reality / Dynamic State 只保留内部 Provider 槽位，v1 门面不提供具体玩法或实时区块状态。

验收范围、自动检查和实机证据见 [QUERY_API_ACCEPTANCE](../../docs/QUERY_API_ACCEPTANCE.md)。
