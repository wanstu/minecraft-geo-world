# GeoWorld

GeoWorld is a Minecraft Java Edition world-generation project built around **geographic rules instead of isolated random terrain noise**.

Rather than treating each chunk as an independent terrain sample, GeoWorld first builds a deterministic world model from the seed, then lets Minecraft generate chunks from that shared model.

```text
Seed
  ↓
Plate layout and tectonic movement
  ↓
Continents / ocean basins / mountain belts / rifts / volcanic regions
  ↓
Long-term wind fields + ocean influence
  ↓
Temperature / precipitation / monsoon effects
  ↓
Rivers / lakes / coastlines / soil moisture
  ↓
Ecological regions / Minecraft biomes
  ↓
Terrain, surface materials, minerals and structures
  ↓
Minecraft chunks
```

## Highlights

- **Large-scale geographic continuity** — continents, mountain systems, basins, coasts and drainage networks are derived from one shared model.
- **Hydrology-aware terrain** — rivers, lakes, estuaries and coastlines are modeled as connected geographic systems instead of unrelated decorations.
- **Biome and surface realization** — terrain shape, climate, water and ecology jointly influence Minecraft biome and surface selection.
- **Special landscapes** — GeoWorld can model features such as island biomes, basins, sink-like depressions and other constrained special terrain without treating them as arbitrary pasted structures.
- **Mining districts** — mineral generation can be organized around regional mining areas instead of distributing every large ore feature uniformly across the world.
- **Settlement and transport analysis** — the model exposes suitability layers for settlement, navigation, transport geography, mining and regional advantages.
- **Interactive WebUI** — inspect generated world models, switch analysis layers, zoom the map and inspect locations from a browser.
- **On-demand analysis** — expensive WebUI layers are calculated only when requested instead of precomputing the entire world during Minecraft startup.
- **Loader/version separation** — shared world-generation logic is kept separate from Minecraft-version and Fabric/NeoForge adapters.
- **No hard dependency on a specific content mod** — GeoWorld is designed to integrate through Minecraft-level concepts instead of requiring one particular third-party ore or biome mod.

## Supported Versions

The current CI release matrix builds the following targets:

| Minecraft | Fabric | NeoForge |
| --- | --- | --- |
| 26.3 | ✅ | ✅ |
| 26.2 | ✅ | ✅ |
| 26.1.2 | ✅ | ✅ |
| 1.21.1 | ✅ | ✅ |
| 1.20.1 | ✅ | — |

Minecraft 1.20.1 currently has a Fabric build only. Forge 1.20.1 is intentionally not part of the release matrix.

Always use the JAR that exactly matches both your Minecraft version and loader.

## World Model

Minecraft coordinates remain technically unbounded, but GeoWorld keeps the main simulated geographic world finite and controlled.

The central modeled region contains the complete geographic system. Outside that region, terrain gradually transitions into long-range ocean and polar boundary zones instead of starting unrelated new continents indefinitely.

This gives plate tectonics, climate, hydrology, ecology and civilization-oriented analysis a stable global scale while preserving Minecraft's chunk-based runtime model.

## WebUI and Tools

GeoWorld includes browser-based tools for inspecting a world model.

Depending on the build and integration point, the WebUI can expose:

- terrain, water and biome previews;
- structures and special landscapes;
- mining districts and mineral-oriented layers;
- settlement suitability;
- navigation and transport geography;
- regional advantages;
- civilization-network analysis;
- location inspection and runtime information.

Heavy analysis layers are lazy: opening Minecraft or binding a world does not automatically calculate every full-map cache.

The standalone tester can be launched from the repository with:

```text
go run ./cmd/geoworld-tester
```

The WebUI normally uses a local dynamic/bound port provided by the running tool or Minecraft integration.

## Installation

1. Install the Fabric or NeoForge loader matching your Minecraft version.
2. Download the GeoWorld JAR built for that exact Minecraft/loader combination.
3. Put the JAR into the Minecraft `mods` directory.
4. Create a new world using GeoWorld's available world-generation entry for that build.

World generation changes are most meaningful on newly generated chunks. Back up important worlds before testing development builds.

## Project Structure

The repository separates the geographic model from Minecraft runtime adaptation:

- `internal/*` — Go world-model, climate, hydrology, ecology, mining, settlement, transport and analysis logic.
- `cmd/geoworld-*` — generators, auditors, acceptance tools and the interactive tester/WebUI.
- `acceptance/*` — automated acceptance rules and evidence generation.
- `java/minecraft-common` — Minecraft-independent shared Java integration logic.
- `java/minecraft-v*-common` — Minecraft-version compatibility layers.
- `java/adapters/*-fabric` — Fabric adapters.
- `java/adapters/*-neoforge` — NeoForge adapters.
- `java/neoforge-mod` — the current Minecraft 26.3 NeoForge module.
- `java/runtime-reader` — Java-side access to GeoWorld model/runtime data.
- `docs/*` — architecture, decisions, progress, performance and acceptance documentation.

## Development and Verification

Core Go verification:

```text
go test ./...
```

Minecraft adapters are built independently for their target Minecraft version and loader. GitHub Actions is the source of truth for the complete supported build matrix.

Examples:

```text
cd java/adapters/1.21.1-neoforge
gradlew.bat build
```

```text
cd java/adapters/26.3-fabric
gradlew.bat build
```

The repository also contains targeted audit and acceptance commands under `cmd/geoworld-*`.

## Documentation

Useful project documents include:

- `docs/ARCHITECTURE.md` — architecture and major subsystem boundaries.
- `docs/DECISIONS.md` — important design decisions and rationale.
- `docs/STATUS.md` — current development status.
- `docs/PROGRESS.md` — detailed development and acceptance history.
- `docs/ROADMAP.md` — longer-term roadmap.
- `docs/CHUNK_PERFORMANCE.md` — chunk-generation performance work.
- `docs/MINERALS_AND_BIOME_COVERAGE.md` — mineral and biome coverage notes.
- `docs/REALISTIC_GEOMORPHOLOGY.md` — realistic erosion, sedimentation, hydrology, terrain-transition and real-world morphology acceptance plan.

GeoWorld is under active development. World-model formats and generation behavior may still change between development versions.

## License

This project is licensed under the **BSD 3-Clause License**. See [LICENSE](LICENSE).

---

# 中文说明

GeoWorld 是一个面向 Minecraft Java Edition 的世界生成项目，核心目标是使用**可解释的地理规律**来生成世界，而不是让每个区块依靠彼此独立的随机噪声拼接地形。

GeoWorld 会先根据 Seed 建立一套确定性的世界模型，再让 Minecraft 的区块生成从同一个世界模型中读取结果。

```text
Seed
  ↓
板块划分与板块运动
  ↓
大陆 / 海盆 / 山脉 / 裂谷 / 火山区域
  ↓
长期风场 + 海洋影响
  ↓
温度 / 降水 / 季风影响
  ↓
河流 / 湖泊 / 海岸线 / 土壤湿度
  ↓
生态区域 / Minecraft 群系
  ↓
地形、地表、矿物与结构
  ↓
Minecraft 区块
```

## 主要特性

- **大尺度地理连续性**：大陆、山系、盆地、海岸和流域来自同一套世界模型，而不是各区块单独随机。
- **水文参与地形生成**：河流、湖泊、入海口和海岸线作为相互关联的地理系统处理。
- **群系与地表实现**：地形、气候、水体和生态共同参与 Minecraft 群系及地表材质的选择。
- **特殊地形**：支持岛屿群系、盆地、天坑式低地等受约束的特殊地形，不把它们简单作为一块独立地形硬贴进原世界。
- **矿区概念**：矿物可以围绕区域级矿区组织，而不是把大型矿脉平均随机撒在所有地方。
- **聚落与交通分析**：世界模型提供聚落、航运、交通地理、矿业和区域优势等适宜度分析。
- **交互式 WebUI**：可在浏览器中查看世界模型、切换分析图层、缩放地图并检查具体位置。
- **按需分析**：重型 WebUI 图层只在真正访问时计算，不会在 Minecraft 启动或世界绑定后自动把整张地图全部预热。
- **版本与 Loader 解耦**：公共世界生成逻辑与 Minecraft 版本、Fabric、NeoForge 适配层分离。
- **不硬依赖特定内容模组**：GeoWorld 的设计目标是基于 Minecraft 层面的概念进行兼容，而不是写死依赖某一个矿物或群系模组。

## 支持版本

当前 CI 发布矩阵覆盖以下目标：

| Minecraft | Fabric | NeoForge |
| --- | --- | --- |
| 26.3 | ✅ | ✅ |
| 26.2 | ✅ | ✅ |
| 26.1.2 | ✅ | ✅ |
| 1.21.1 | ✅ | ✅ |
| 1.20.1 | ✅ | — |

Minecraft 1.20.1 当前仅提供 Fabric 构建，发布矩阵中不提供 Forge 1.20.1。

安装时必须选择与 **Minecraft 版本和 Loader 都完全一致** 的 JAR。

## 世界模型

Minecraft 坐标在技术上仍然保持无限。

GeoWorld 会在中央有限范围内生成完整的主地理世界，并在主地理区之外逐步过渡到远洋和极地区域，而不是无限重复生成彼此无关的新大陆。

这样可以让板块、气候、水文、生态以及文明建设相关分析拥有稳定、可控的全局尺度，同时仍然兼容 Minecraft 的 Chunk 运行时生成方式。

## WebUI 与工具

GeoWorld 提供浏览器工具用于查看和分析世界模型。

根据具体版本和 Loader 集成情况，WebUI 可以提供：

- 地形、水体和群系预览；
- 结构与特殊地形；
- 矿区和矿物相关图层；
- 聚落适宜度；
- 航运与交通地理；
- 区域优势；
- 文明网络分析；
- 坐标位置和运行时信息查看。

耗时较高的分析图层采用按需计算机制：进入 Minecraft 或绑定世界时不会自动计算所有全图缓存。

仓库中的独立测试器可以直接运行：

```text
go run ./cmd/geoworld-tester
```

WebUI 通常使用运行中的工具或 Minecraft 集成所提供的本地动态/已绑定端口。

## 安装

1. 安装与 Minecraft 版本匹配的 Fabric 或 NeoForge。
2. 下载与 Minecraft 版本、Loader 完全对应的 GeoWorld JAR。
3. 将 JAR 放入 Minecraft 的 `mods` 目录。
4. 使用该构建提供的 GeoWorld 世界生成入口创建新世界。

世界生成类改动主要作用于新生成的区块。测试开发版本前，建议先备份重要世界。

## 项目结构

GeoWorld 将地理模型与 Minecraft 运行时适配拆开维护：

- `internal/*`：Go 实现的世界模型、气候、水文、生态、矿业、聚落、交通与分析逻辑。
- `cmd/geoworld-*`：生成器、审计工具、验收工具以及交互测试器 / WebUI。
- `acceptance/*`：自动验收规则与验收证据生成。
- `java/minecraft-common`：Minecraft 公共 Java 集成逻辑。
- `java/minecraft-v*-common`：不同 Minecraft 版本的兼容层。
- `java/adapters/*-fabric`：Fabric 适配层。
- `java/adapters/*-neoforge`：NeoForge 适配层。
- `java/neoforge-mod`：当前 Minecraft 26.3 NeoForge 模块。
- `java/runtime-reader`：Java 侧读取 GeoWorld 世界模型和运行时数据的公共能力。
- `docs/*`：架构、设计决定、进度、性能与验收文档。

## 开发与验证

Go Core 的基础验证：

```text
go test ./...
```

不同 Minecraft / Loader 适配器分别构建，完整支持矩阵以 GitHub Actions CI 为准。

例如：

```text
cd java/adapters/1.21.1-neoforge
gradlew.bat build
```

```text
cd java/adapters/26.3-fabric
gradlew.bat build
```

仓库的 `cmd/geoworld-*` 下还包含多种针对地形、水文、生态、性能和一致性的专项审计 / 验收命令。

## 项目文档

常用文档包括：

- `docs/ARCHITECTURE.md`：整体架构与子系统边界。
- `docs/DECISIONS.md`：重要设计决定及原因。
- `docs/STATUS.md`：当前开发状态。
- `docs/PROGRESS.md`：完整开发与验收历史。
- `docs/ROADMAP.md`：后续路线。
- `docs/CHUNK_PERFORMANCE.md`：区块生成性能优化记录。
- `docs/MINERALS_AND_BIOME_COVERAGE.md`：矿物和群系覆盖说明。
- `docs/REALISTIC_GEOMORPHOLOGY.md`：真实侵蚀、沉积、水文一致性、地貌过渡与真实地貌对照验收计划。

GeoWorld 仍在持续开发中，开发版本之间的世界模型格式和生成结果仍可能发生变化。

## 许可证

本项目采用 **BSD 3-Clause License** 开源。详见 [LICENSE](LICENSE)。
