# GeoWorld 事实字段契约表（FACT_REGISTRY）

更新时间：2026-11-07
适用：`geoworld-query-v1`，`GeoWorldQueryService.NaturalField` 共 **86** 个常量（本表全覆盖）。
约束来源：`DECISIONS.md` D-048 / D-049 / D-050；总体定位见 `FOUNDATION_API.md`。

本表是 Query 字段语义的唯一权威说明。字段增删、单位、取值域、缺失条件变化，必须同步修改本表。

---

## 0. 通用契约

1. **坐标量化**：查询坐标先按 `Resolution`（`BLOCK_1/4/16`）量化，结果里的 `sampleX/sampleZ` 是量化后坐标。
2. **缺失语义**：不可用的字段只出现在 `unavailableFields`，`values` 中没有这个键。调用方不得把缺失当成 0。
3. **版本追溯**：每个结果都带 `worldProfileVersion / generationRulesVersion / worldModelVersion`。旧模型缺字段时依赖规则 2，不能假设字段一定存在。
4. **高度单位**：`ELEVATION`、`WATER_SURFACE_ELEVATION`、`WATER_DEPTH` 都是归一化高度，不是方块 Y。26.3 映射见 D-044：0.0→Y=-64，0.5→Y=63，1.0→Y=320。
5. **评分方向**：`0..1` 评分默认值越高越有利；本表单独标出的**负向**字段例外。
6. **验收列**：`V` 表示该字段已纳入 QUERY-02～04 的字符串域、缺失/有限值与数值范围检查。当前 86 字段均在 6 个模型的采样中出现；这不代表每个启发式评分已做玩法语义校准。

---

## 1. 地形与生态（离散标签相邻可跳变，D-042）

| 字段 | 类型 / 单位 | 可用性 | 说明 | 验收 |
|---|---|---|---|---|
| `ELEVATION` | double，归一化 | 总是可用 | 最终地面高度 | V |
| `SLOPE` | double，归一化比值 | 总是可用 | 按 Resolution 尺度估算的坡度，不是角度 | V |
| `ASPECT_DEGREES` | double，0..360° | 总是可用 | 坡向；平地上的值不保证有意义，建议配合 `SLOPE` 过滤 | V |
| `ECO_REGION` | String | 总是可用 | 取值见 A.1 | V |
| `BIOME_FAMILY` | String | 总是可用 | 与 `ECO_REGION` 同一取值域；是 GeoWorld 生态家族，**不是 Minecraft 群系 ID**（D-027） | V |
| `TERRAIN_VARIANT` | String | 总是可用 | 取值见 A.2 | V |
| `SURFACE_PROFILE` | String | 总是可用 | 取值见 A.3 | V |
| `VEGETATION_PROFILE` | String | 总是可用 | 取值见 A.4 | V |

## 2. 水体

| 字段 | 类型 / 单位 | 可用性 | 说明 | 验收 |
|---|---|---|---|---|
| `WATER_KIND` | String | 总是可用 | `None / Ocean / Lake / River` | V |
| `WATER_SURFACE_ELEVATION` | double，归一化 | 仅水体处 | 水面未定义时缺失 | V |
| `WATER_DEPTH` | double，归一化 | 总是可用 | 无水时为 0，这里的 0 不表示缺失 | V |
| `OCEAN_ELIGIBILITY` | double，0..1 | 总是可用 | 外海连通资格；判断是否为海洋应以 `WATER_KIND` 为准 | V |
| `RIVER_WEIGHT` | double，0..1 | 总是可用 | 河道横向权重，河心最高并向岸边递减（D-041）；只适合做“离河心多近”的相对量 | V |

## 3. 气候（长期统计量，不代表实时天气，D-005）

气候数据不可用时，下面 5 个字段一起缺失。

| 字段 | 类型 / 单位 | 说明 | 验收 |
|---|---|---|---|
| `ANNUAL_MEAN_TEMP_C` | double，°C | 年均温，已包含海拔递减和极地修正 | V |
| `ANNUAL_PRECIP_MM` | double，mm/年 | 年降水 | V |
| `MONSOON_INDEX` | double，0..1 | 季风强度；海洋为 0，极地衰减 | V |
| `WIND_X` | double，无量纲向量分量 | **年均**盛行风 X 分量（`+X` 为东），模长约 ≤1 × `PrevailingWindStrength`（默认 1） | V |
| `WIND_Z` | double，无量纲向量分量 | 年均盛行风 Z 分量 | V |

夏季风和冬季风（`SummerWind*` / `WinterWind*`）只在 Go 模型中存在，**没有通过 Query 暴露**。季节类模组如果需要，应按 D-049 新增字段，不能借用 `WIND_X/Z`。

## 4. 水文

| 字段 | 类型 / 单位 | 可用性 | 说明 | 验收 |
|---|---|---|---|---|
| `WATERSHED_ID` | int | 仅 id > 0 时可用 | 流域 ID；同一模型内稳定，跨 `generationRulesVersion` 不保证一致 | V |
| `FLOW_ACCUMULATION` | double，相对量 | 有水文数据时可用 | 参考单元等效径流的累积值；只用于同一世界内比较大小，没有物理单位 | V |
| `RIVER_ORDER` | int | 有水文数据时可用 | 宏观网格上的 Strahler 河流等级；0 表示非河道 | V |
| `NATURAL_FLOW_DIRECTION` | Map `{x,z}` | x、z 都有值时可用 | 自然流向分量 | V |

## 5. 自然要素距离（单位都是 block）

| 字段 | 可用性 | 说明 | 验收 |
|---|---|---|---|
| `RIVER_ID` | 模型内有河流时可用 | Integer，按本模型河段索引从 1 开始编号；**只在同一模型内有效**，不能作为跨版本的持久标识 | V |
| `DISTANCE_TO_RIVER` | 同上 | 到最近河岸的距离，位于河内时为 0 | V |
| `DISTANCE_TO_LAKE` | 模型内有湖时可用 | 到湖外缘的距离，位于湖内时为 0 | V |
| `DISTANCE_TO_COAST` | 模型内可用 | 到最近海岸线段的距离 | V |
| `DISTANCE_TO_OCEAN` | 模型内可用；模型外只有海洋点可用 | 海洋上为 0，其余位置等于海岸距离 | V |

模型外（外围海洋区）的 `RIVER_ID / DISTANCE_TO_RIVER / LAKE / COAST` 一律缺失。

## 6. 地质与矿业

| 字段 | 类型 | 说明 | 验收 |
|---|---|---|---|
| `GEOLOGICAL_REGION` | String | `<family>:<hex>`，见 A.5 | V |
| `ORE_PROVINCE` | String | `ore-<0..7>-<hex>`；中间数字是矿省强度档位，**不是矿种** | V |
| `MINING_SUITABILITY` | 0..1 | 综合采矿适宜度 | V |
| `SURFACE_MINING_SUITABILITY` | 0..1 | 地表采矿适宜度，对应约 Y=80 的剖面 | V |
| `SHALLOW_MINING_SUITABILITY` | 0..1 | 浅层采矿适宜度，对应约 Y=28 的剖面 | V |
| `DEEP_MINING_SUITABILITY` | 0..1 | 深层采矿适宜度，对应约 Y=-40 的剖面 | V |
| `INDUSTRIAL_MINERAL_VALUE` | 0..1 | 工业价值 | V |
| `MINERAL_RESOURCE_DIVERSITY` | 0..1 | 资源多样性 | V |
| `DOMINANT_MINERAL_RESOURCE` | String | 主导矿物家族名（小写），无矿物时为 `none` | V |
| `MINING_DISTRICT_CLASS` | String | 见 A.6 | V |
| `MINERAL_RESOURCE_PROFILE` | Map<String,Double> | 键是矿物家族，值为 0..1 | V |

矿业字段的口径：

- 描述的是**区域资源经济属性**，不表示这个坐标正下方有什么矿。
- 非陆地或有水的位置得分为 0，`MINING_DISTRICT_CLASS=none`。
- 当前 Query v1 的资源评分来自 `GeoWorldMineralAffinity.MINERALS`：`coal / iron / copper / gold / lapis / redstone / diamond / emerald`。生成引擎的 `GeoWorldMineralContext` 可包含模组矿物，但现有这组查询尚未消费该上下文，不能宣称此 Map 已覆盖模组矿物；调用方仍应遍历 Map 的键。

## 7. 宜居与航运（0..1，总是可用）

| 字段 | 说明 | 验收 |
|---|---|---|
| `SETTLEMENT_BUILDABILITY` | 可建造性 | V |
| `FRESH_WATER_ACCESS` | 淡水可达性 | V |
| `FLOOD_RISK` | 洪水风险，**负向** | V |
| `AGRICULTURE_SUITABILITY` | 农业适宜度 | V |
| `CLIMATE_COMFORT` | 气候舒适度 | V |
| `TRANSPORT_ACCESS` | 交通可达性 | V |
| `RESOURCE_ACCESS` | 资源可达性 | V |
| `SETTLEMENT_SUITABILITY` | 综合定居适宜度 | V |
| `SPAWN_SUITABILITY` | 出生适宜度，仅作参考；与 D-047 的 Spawn Planner 无关 | V |
| `NEAREST_RIVER_HALF_WIDTH` | 最近河段半宽，单位 block；附近没有河时缺失 | V |
| `NEAREST_RIVER_GRADIENT` | 最近河段纵坡，归一化高差 / 河段长度的相对量；附近没有河时缺失 | V |
| `WATERWAY_NAVIGABILITY` | 航道可航性 | V |
| `PORT_SUITABILITY` | 港口适宜度 | V |
| `NAVIGATION_BOTTLENECK_RISK` | 航运瓶颈风险，**负向** | V |
| `WATERWAY_CLASS` | String，见 A.7 | V |
| `PORT_TYPE` | String，见 A.8 | V |

## 8. 文明、区域优势与交通走廊（0..1 或标签，总是可用）

| 字段 | 类型 / 单位 | 说明 | 验收 |
|---|---|---|---|
| `CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY` | double，0..1 | 农业聚落潜力 | V |
| `CIVILIZATION_PORT_CITY_SUITABILITY` | double，0..1 | 港城潜力 | V |
| `CIVILIZATION_MINING_TOWN_SUITABILITY` | double，0..1 | 矿镇潜力 | V |
| `CIVILIZATION_TRANSPORT_NODE_SUITABILITY` | double，0..1 | 交通节点潜力 | V |
| `CIVILIZATION_NODE_SUITABILITY` | double，0..1 | 综合节点潜力 | V |
| `CIVILIZATION_SPECIALIZATION_STRENGTH` | double，0..1 | 文明节点专业化程度 | V |
| `CIVILIZATION_NODE_CLASS` | String | 文明节点分类，见 A.9 | V |
| `REGIONAL_BUILDING_ADVANTAGE` | double，0..1 | 建设区域优势 | V |
| `REGIONAL_AGRICULTURE_ADVANTAGE` | double，0..1 | 农业区域优势 | V |
| `REGIONAL_PORT_ADVANTAGE` | double，0..1 | 港口区域优势 | V |
| `REGIONAL_MINING_ADVANTAGE` | double，0..1 | 矿业区域优势 | V |
| `REGIONAL_TRANSPORT_ADVANTAGE` | double，0..1 | 交通区域优势 | V |
| `REGIONAL_SPECIALIZATION_STRENGTH` | double，0..1 | 区域专业化程度 | V |
| `REGIONAL_ADVANTAGE_OVERLAP_RISK` | double，0..1 | 优势重叠风险，**负向** | V |
| `REGIONAL_HIGH_ADVANTAGE_COUNT` | int，0..5 | 高优势项数 | V |
| `REGIONAL_DOMINANT_ADVANTAGE` | String | 主导优势，见 A.10 | V |
| `REGIONAL_ADVANTAGE_CLASS` | String | 区域优势分类，见 A.10 | V |
| `REGIONAL_REGION_KEY` | String | 连通优势区域标识；无区域时为 `none` | V |
| `REGIONAL_REGION_CELL_COUNT` | int | 区域宏观单元数；无区域时为 0 | V |
| `REGIONAL_REGION_AREA_BLOCKS2` | double，block² | 区域面积；无区域时为 0 | V |
| `REGIONAL_REGION_CENTROID_X` | double，世界 block 坐标 | 区域质心 X；无区域时为 0 | V |
| `REGIONAL_REGION_CENTROID_Z` | double，世界 block 坐标 | 区域质心 Z；无区域时为 0 | V |
| `NATURAL_CORRIDOR_SUITABILITY` | double，0..1 | 6 类走廊强度中的最大值 | V |
| `RIVER_VALLEY_CORRIDOR` | double，0..1 | 河谷走廊强度 | V |
| `HILL_PASS_CORRIDOR` | double，0..1 | 丘陵隘口走廊强度 | V |
| `MOUNTAIN_PASS_CORRIDOR` | double，0..1 | 山口走廊强度 | V |
| `PLATEAU_ACCESS_CORRIDOR` | double，0..1 | 高原入口走廊强度 | V |
| `BASIN_GATEWAY_CORRIDOR` | double，0..1 | 盆地门户走廊强度 | V |
| `LOW_DIVIDE_CORRIDOR` | double，0..1 | 低分水岭走廊强度 | V |
| `STRAIT_ISTHMUS_POTENTIAL` | double，0..1 | 海峡 / 地峡潜力 | V |
| `TRANSPORT_CHOKEPOINT_STRENGTH` | double，0..1 | 咽喉强度，取隘口、门户、分水岭、海峡中的最大值 | V |
| `NATURAL_CORRIDOR_CLASS` | String | 自然走廊分类，见 A.11 | V |

`REGIONAL_REGION_*` 是区域级事实，同一非空区域内的各点返回相同的单元数、面积与质心。
这一组都是**推断性评分**（启发式），可以作为玩法的倾向信号，不能当作地理真值。

---

## 附录 A：权威取值域

任何字符串字段返回本附录之外的值，都属于契约违规。

- **A.1 ECO_REGION / BIOME_FAMILY**：`Unknown, Warm Ocean, Temperate Ocean, Cold Ocean, Frozen Ocean, Ice Sheet, Tundra, Alpine, Boreal Forest, Temperate Forest, Temperate Grassland, Temperate Desert, Mediterranean Shrubland, Subtropical Forest, Hot Desert, Savanna, Tropical Monsoon Forest, Tropical Rainforest, Wetland`
- **A.2 TERRAIN_VARIANT**：`Unknown, Ocean, Plain, Rolling Plain, Hills, Foothills, Mountain, Plateau, Floodplain, Alluvial Plain, Alluvial Fan, Canyon`
- **A.3 SURFACE_PROFILE**：`Unknown, Ocean Sand, Ocean Gravel, Frozen Ocean, Ice / Snow, Tundra Soil, Alpine Rock, Alpine Snow, Forest Loam, Podzol, Grass Loam, Dry Grass Soil, Shrub Soil, Desert Sand, Desert Gravel, Alluvial Silt, Wet Mud, Peat`
- **A.4 VEGETATION_PROFILE**：`Unknown, None, Tundra Sparse, Alpine Sparse, Conifer Forest, Mixed Forest, Broadleaf Forest, Dense Rainforest, Monsoon Forest, Grassland, Dry Grassland, Shrubland, Savanna, Desert Sparse, Wetland`
- **A.5 GEOLOGICAL_REGION family**：`oceanic-crust`（Ocean）、`orogenic-belt`（Mountain/Foothills）、`uplifted-block`（Plateau/Hills）、`exposed-crust`（Canyon）、`sedimentary-basin`（Floodplain/Alluvial *）、`continental-interior`（其余）
- **A.6 MINING_DISTRICT_CLASS**：`none, low_potential, surface_district, shallow_district, deep_district, mixed_district`
- **A.7 WATERWAY_CLASS**：`open_sea, shallow_sea, lake, shallow_lake, major_river, river, local_waterway, non_navigable, none`
- **A.8 PORT_TYPE**：`none, sea, lake, river`
- **A.9 CIVILIZATION_NODE_CLASS**：`none, agricultural_settlement, port_city, mining_town, transport_node, low_potential, multi_role_hub`
- **A.10 REGIONAL_DOMINANT_ADVANTAGE**：`none, building, agriculture, port, mining, transport`；**REGIONAL_ADVANTAGE_CLASS**：`none, building_region, agriculture_region, port_region, mining_region, transport_region, low_potential, multi_advantage_region`
- **A.11 NATURAL_CORRIDOR_CLASS**：`none, river_valley, hill_pass, mountain_pass, plateau_access, basin_gateway, low_divide, low_potential`

命名风格的历史遗留：A.1–A.4 是带空格的标题式写法，A.6–A.11 是蛇形小写。按 D-049，不允许原地统一；需要统一时新增 `_V2` 字段。

---

## 附录 B：计算依赖（影响查询成本）

```text
MINING            ← 只依赖 sample
CLIMATE / HYDROLOGY / NATURAL_FEATURES / NAVIGATION_FEATURES ← 各自独立的采样组
SETTLEMENT        ← CLIMATE + NATURAL_FEATURES + NAVIGATION_FEATURES
NAVIGATION_SCORE  ← SETTLEMENT + NAVIGATION_FEATURES
CIVILIZATION / REGIONAL_ADVANTAGE ← SETTLEMENT + NAVIGATION_SCORE + MINING
REGIONAL_REGION   ← 区域聚合（跨宏观节点，成本最高）
TRANSPORT_GEOGRAPHY ← SETTLEMENT + NAVIGATION_SCORE + HYDROLOGY
```

调用建议：同一组字段放在一次请求里提交；不要逐 tick 单字段查询；查询 `REGIONAL_REGION_*` 时用 `BLOCK_16`。

---

## 附录 C：自动验收证据与剩余发布条件

2026-11-07 已通过：QUERY-01～10 与现有 24 项 MODEL/RUNTIME/PROFILE/PARAM 验收，合计 34/34 项，覆盖 6 个已提交模型。

- 每个模型：32×32 网格 + 18 个 golden 特殊点，共 1,042 点；全模型合计 6,252 点。
- QUERY-01：86 个字段逐行登记，与枚举一一对应。
- QUERY-02～04：所有字段的取值域、缺失分区、有限值及数值范围；本表所有 `V` 均由这些检查支撑。
- QUERY-05：同区域标识的单元数、面积、质心一致，每个模型至少 8 个区域被重复采样。
- QUERY-06：1/4/16 方块分辨率的同格偏移完全相等，覆盖负坐标。
- QUERY-07～08：模型版本与模型外缺失语义。
- QUERY-09：旧 golden 无洪泛地形节点，使用每模型两个受控持久化测试点验证负向洪水风险；不宣称生成器的真实洪泛平原已被这些模型覆盖。

按代码事实修正的文档偏差：展开合并字段行；补全水域 `none` 与 `open_sea / shallow_sea / lake / shallow_lake` 分类；区分外围海岛和海洋；注明当前 Query 矿业评分仍为 8 类原版矿物。没有改变查询返回值或世界生成。

QUERY-10 已在全部 6 个模型上验证门面与直接查询等价，`supports` 与版本信息也一致；门面生命周期黑盒检查已通过。两个 Loader 的实机切换仍是发布前置条件。D-048 封控保持，不能据本表宣称 Query API 已对外开放。
