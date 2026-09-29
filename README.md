# GeoWorld

GeoWorld 是一个面向 Minecraft Java Edition 的“地理规律驱动世界生成”项目。

目标不是用随机噪声拼接出看起来像大陆的地形，而是先建立可解释的世界模型：

```text
Seed
  ↓
板块划分与运动
  ↓
大陆 / 海盆 / 山脉 / 裂谷 / 火山带
  ↓
长期风场 + 洋流
  ↓
季风影响 + 温度 / 降水
  ↓
河流 / 湖泊 / 土壤湿度
  ↓
生态区 / Minecraft Biome
  ↓
Chunk 地形与方块
```

## 当前产品形态

GeoWorld 不限定为单一 Mod 或单一离线工具。

- **GeoWorld Core**：世界模型与算法规范。
- **GeoWorld Generator**：游戏外预生成、检查、可视化和导出。
- **GeoWorld Runtime**：Minecraft 游戏内按需生成 Chunk。
- **Loader Adapter**：NeoForge / Fabric / Forge 等运行时适配。

## V1 世界拓扑

Minecraft 坐标仍保持理论无限。

GeoWorld 只在中央有限区域生成完整地理世界：

- 东 / 西：逐步进入外洋，最终成为 **Infinite Storm Ocean**。
- 北 / 南：逐步进入极区，最终成为 **Infinite Frozen Ocean**。
- 四角：风暴洋与冰冻洋自然叠加。
- 主地理区之外不再生成新的大陆。

这样既保留 Minecraft 无限世界的技术模型，又让板块、洋流、季风和生态拥有有限且可控的模拟尺度。

## 当前阶段

当前先做产品规划与交互原型，不锁死具体 Loader，也不急于实现 Minecraft 接入。

原型入口：

- `prototype/index.html`
- `prototype/generator.html`
- `prototype/runtime.html`


## 项目文档

- `docs/STATUS.md`：当前进度、稳定基线、正在进行和下一步。
- `docs/DECISIONS.md`：关键设计决定及原因。
- `docs/ROADMAP.md`：V1 分阶段路线。
- `docs/PHASE1.md`：当前板块 / 大陆算法实验细节。
- `docs/ARCHITECTURE.md`：整体架构。
- `docs/PRODUCT.md`：产品边界。


## 查看静态原型

原型不需要启动 Web 服务，也不依赖 Node。

Windows 可直接双击：

`open-prototype.cmd`

或直接打开：

- `prototype/index.html`
- `prototype/generator.html`
- `prototype/runtime.html`
- `prototype/plates.html`

所有 CSS 与生成图片均使用相对路径，可通过 `file://` 直接加载。
