package geoworld.minecraft.v26common.client;

import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.preview.GeoWorldPreviewModel;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;

import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphicsExtractor;
import net.minecraft.client.input.MouseButtonEvent;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.layouts.HeaderAndFooterLayout;
import net.minecraft.client.gui.layouts.LinearLayout;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.locale.Language;
import net.minecraft.network.chat.CommonComponents;
import net.minecraft.network.chat.Component;
import net.minecraft.world.level.levelgen.Heightmap;

import java.util.Locale;

public class GeoWorldV26PreviewScreen extends Screen {
    @FunctionalInterface
    public interface BiomeIdResolver {
        String biomeId(GeoWorldV1 model, double blockX, double blockZ);
    }
    private enum PreviewLayer {
        BIOME,
        CONTINENT,
        TERRAIN,
        TEMPERATURE,
        PRECIPITATION,
        MONSOON
    }

    private static final Component TITLE =
        Component.translatable("geoworld.preview.title");

    private static final int MAX_COLUMNS = 160;
    private static final int MAX_ROWS = 100;
    private static final double INITIAL_LOCAL_SPAN_X = 1600.0;
    private static final double INITIAL_LOCAL_SPAN_Z = 1000.0;
    private static final double MIN_VIEW_SPAN_X = 128.0;
    private static final double MIN_VIEW_SPAN_Z = 80.0;
    private static final double ZOOM_STEP = 0.78;
    private static final long DRAG_REBUILD_INTERVAL_NANOS = 50_000_000L;
    // Supported Minecraft 26.x MouseButtonInfo uses 1 for the primary/left button.
    private static final int PRIMARY_MOUSE_BUTTON = 1;


    private final Screen parent;
    private final GeoWorldPreviewModel previewModel;
    private final HeightMapping heights;
    private final BiomeIdResolver biomeIdResolver;
    private final GeoWorldV26ScreenNavigator screenNavigator;
    private final Double focusX;
    private final Double focusZ;

    private HeaderAndFooterLayout layout;
    private Button viewModeButton;
    private Button layerButton;
    private Button teleportButton;

    private PreviewLayer previewLayer = PreviewLayer.BIOME;
    private boolean localView;
    private boolean draggingMap;
    private boolean mapPressActive;
    private boolean mapPressMoved;
    private double mapPressTravel;
    private boolean samplesDirty;
    private long lastSampleRebuildNanos;
    private Double selectedX;
    private Double selectedZ;
    private double viewOriginX;
    private double viewOriginZ;
    private double viewSpanX;
    private double viewSpanZ;

    private int columns;
    private int rows;
    private int[] colors;

    private int landSamples;
    private int oceanSamples;
    private int lakeSamples;
    private int riverSamples;

    private int mapLeft;
    private int mapTop;
    private int mapWidth;
    private int mapHeight;

    public GeoWorldV26PreviewScreen(
        Screen parent,
        GeoWorldEngineBridge.GeneratedModel generated,
        HeightMapping heights,
        BiomeIdResolver biomeIdResolver,
        GeoWorldV26ScreenNavigator screenNavigator
    ) {
        this(
            parent,
            generated.model(),
            null,
            null,
            heights,
            biomeIdResolver,
            screenNavigator
        );
    }

    public GeoWorldV26PreviewScreen(
        Screen parent,
        GeoWorldV1 model,
        Double focusX,
        Double focusZ,
        HeightMapping heights,
        BiomeIdResolver biomeIdResolver,
        GeoWorldV26ScreenNavigator screenNavigator
    ) {
        super(TITLE);
        this.parent = parent;
        this.previewModel = new GeoWorldPreviewModel(model);
        this.heights = java.util.Objects.requireNonNull(heights, "heights");
        this.biomeIdResolver = java.util.Objects.requireNonNull(
            biomeIdResolver,
            "biomeIdResolver"
        );
        this.screenNavigator = java.util.Objects.requireNonNull(
            screenNavigator,
            "screenNavigator"
        );
        this.focusX = focusX;
        this.focusZ = focusZ;

        if (focusX != null && focusZ != null) {
            this.resetLocalView();
        } else {
            this.resetGlobalView();
        }
    }

    @Override
    protected void init() {
        this.layout = new HeaderAndFooterLayout(this, 27, 31);
        this.layout.addTitleHeader(this.title, this.font);

        LinearLayout footer = this.layout.addToFooter(
            LinearLayout.horizontal().spacing(8)
        );

        if (this.focusX != null && this.focusZ != null) {
            this.viewModeButton = footer.addChild(
                Button.builder(
                    this.viewModeLabel(),
                    button -> this.toggleViewMode()
                )
                    .width(120)
                    .build()
            );
        }

        this.layerButton = footer.addChild(
            Button.builder(
                this.layerLabel(),
                button -> this.cycleLayer()
            )
                .width(112)
                .build()
        );

        footer.addChild(
            Button.builder(
                Component.translatable("geoworld.preview.reset_view"),
                button -> this.resetCurrentView()
            )
                .width(112)
                .build()
        );

        if (this.focusX != null && this.focusZ != null) {
            this.teleportButton = footer.addChild(
                Button.builder(
                    Component.translatable("geoworld.preview.teleport"),
                    button -> this.teleportToSelection()
                )
                    .width(112)
                    .build()
            );
            this.teleportButton.active = false;
        }

        footer.addChild(
            Button.builder(CommonComponents.GUI_BACK, button -> this.onClose())
                .width(112)
                .build()
        );

        this.layout.visitWidgets(this::addRenderableWidget);
        this.repositionElements();
    }

    private void selectMapPoint(double mouseX, double mouseY) {
        double fx = clamp(
            (mouseX - this.mapLeft) / this.mapWidth,
            0.0,
            1.0
        );
        double fz = clamp(
            (mouseY - this.mapTop) / this.mapHeight,
            0.0,
            1.0
        );
        this.selectedX = this.viewOriginX + fx * this.viewSpanX;
        this.selectedZ = this.viewOriginZ + fz * this.viewSpanZ;
        if (this.teleportButton != null) {
            this.teleportButton.active = true;
        }
    }

    private void teleportToSelection() {
        if (this.selectedX == null || this.selectedZ == null) {
            return;
        }

        Minecraft minecraft = Minecraft.getInstance();
        if (minecraft.player == null) {
            return;
        }
        var server = minecraft.getSingleplayerServer();
        if (server == null) {
            return;
        }

        int blockX = (int)Math.floor(this.selectedX);
        int blockZ = (int)Math.floor(this.selectedZ);
        var playerId = minecraft.player.getUUID();

        server.execute(() -> {
            var serverPlayer = server.getPlayerList().getPlayer(playerId);
            if (serverPlayer == null) {
                return;
            }
            var level = serverPlayer.level();

            // Force-load/generate the target chunk before querying its real
            // surface. Teleport uses the actual generated world rather than
            // only trusting the preview model's predicted Y.
            level.getChunk(blockX >> 4, blockZ >> 4);
            int y = level.getHeight(
                Heightmap.Types.WORLD_SURFACE,
                blockX,
                blockZ
            );

            serverPlayer.teleportTo(
                blockX + 0.5,
                y,
                blockZ + 0.5
            );
            serverPlayer.sendSystemMessage(
                Component.translatable(
                    "geoworld.preview.teleport_done",
                    blockX,
                    y,
                    blockZ
                )
            );

            minecraft.execute(() -> this.screenNavigator.setScreen(null));
        });
    }

    private void panByPixels(double dx, double dy) {
        if (this.mapWidth <= 0 || this.mapHeight <= 0) {
            return;
        }

        GeoWorldV1.Geometry geometry = this.previewModel.manifest().geometry();
        double deltaX = -dx / this.mapWidth * this.viewSpanX;
        double deltaZ = -dy / this.mapHeight * this.viewSpanZ;

        this.viewOriginX = clampViewOrigin(
            this.viewOriginX + deltaX,
            geometry.originX(),
            geometry.spanXBlocks(),
            this.viewSpanX
        );
        this.viewOriginZ = clampViewOrigin(
            this.viewOriginZ + deltaZ,
            geometry.originZ(),
            geometry.spanZBlocks(),
            this.viewSpanZ
        );

        this.samplesDirty = true;
        long now = System.nanoTime();
        if (
            now - this.lastSampleRebuildNanos >=
            DRAG_REBUILD_INTERVAL_NANOS
        ) {
            this.rebuildSamples();
        }
    }

    private void toggleViewMode() {
        if (this.focusX == null || this.focusZ == null) {
            this.resetGlobalView();
            return;
        }
        if (this.localView) {
            this.resetGlobalView();
        } else {
            this.resetLocalView();
        }
        this.refreshViewModeButton();
    }

    private void resetCurrentView() {
        if (
            this.localView &&
            this.focusX != null &&
            this.focusZ != null
        ) {
            this.resetLocalView();
        } else {
            this.resetGlobalView();
        }
        this.refreshViewModeButton();
    }

    private void resetGlobalView() {
        GeoWorldV1.Geometry geometry = this.previewModel.manifest().geometry();
        this.localView = false;
        this.viewOriginX = geometry.originX();
        this.viewOriginZ = geometry.originZ();
        this.viewSpanX = geometry.spanXBlocks();
        this.viewSpanZ = geometry.spanZBlocks();
        this.rebuildSamples();
    }

    private void resetLocalView() {
        GeoWorldV1.Geometry geometry = this.previewModel.manifest().geometry();
        this.localView = true;

        this.viewSpanX = Math.min(
            geometry.spanXBlocks(),
            INITIAL_LOCAL_SPAN_X
        );
        this.viewSpanZ = Math.min(
            geometry.spanZBlocks(),
            INITIAL_LOCAL_SPAN_Z
        );

        double centerX = this.focusX != null
            ? this.focusX
            : geometry.originX() + geometry.spanXBlocks() * 0.5;
        double centerZ = this.focusZ != null
            ? this.focusZ
            : geometry.originZ() + geometry.spanZBlocks() * 0.5;

        this.viewOriginX = clampViewOrigin(
            centerX - this.viewSpanX * 0.5,
            geometry.originX(),
            geometry.spanXBlocks(),
            this.viewSpanX
        );
        this.viewOriginZ = clampViewOrigin(
            centerZ - this.viewSpanZ * 0.5,
            geometry.originZ(),
            geometry.spanZBlocks(),
            this.viewSpanZ
        );
        this.rebuildSamples();
    }

    private Component layerLabel() {
        String key = switch (this.previewLayer) {
            case BIOME -> "geoworld.preview.layer.biome";
            case CONTINENT -> "geoworld.preview.layer.continent";
            case TERRAIN -> "geoworld.preview.layer.terrain";
            case TEMPERATURE -> "geoworld.preview.layer.temperature";
            case PRECIPITATION -> "geoworld.preview.layer.precipitation";
            case MONSOON -> "geoworld.preview.layer.monsoon";
        };
        return Component.translatable(
            "geoworld.preview.layer_button",
            Component.translatable(key)
        );
    }

    private void cycleLayer() {
        PreviewLayer[] values = PreviewLayer.values();
        this.previewLayer = values[
            (this.previewLayer.ordinal() + 1) % values.length
        ];
        this.rebuildSamples();
        if (this.layerButton != null) {
            this.layerButton.setMessage(this.layerLabel());
        }
    }

    private Component viewModeLabel() {
        return Component.translatable(
            this.localView
                ? "geoworld.preview.show_global"
                : "geoworld.preview.show_local"
        );
    }

    private void refreshViewModeButton() {
        if (this.viewModeButton != null) {
            this.viewModeButton.setMessage(this.viewModeLabel());
        }
    }

    private void rebuildSamples() {
        double aspect = this.viewSpanX / this.viewSpanZ;
        if (!Double.isFinite(aspect) || aspect <= 0.0) {
            aspect = 1.0;
        }

        if (aspect >= 1.0) {
            this.columns = MAX_COLUMNS;
            this.rows = Math.max(
                32,
                Math.min(
                    MAX_ROWS,
                    (int)Math.round(MAX_COLUMNS / aspect)
                )
            );
        } else {
            this.rows = MAX_ROWS;
            this.columns = Math.max(
                32,
                Math.min(
                    MAX_COLUMNS,
                    (int)Math.round(MAX_ROWS * aspect)
                )
            );
        }

        this.samplesDirty = false;
        this.lastSampleRebuildNanos = System.nanoTime();
        this.landSamples = 0;
        this.oceanSamples = 0;
        this.lakeSamples = 0;
        this.riverSamples = 0;
        this.colors = new int[Math.multiplyExact(this.columns, this.rows)];

        for (int row = 0; row < this.rows; row++) {
            double fz = this.rows == 1
                ? 0.5
                : (double)row / (this.rows - 1);
            double z = this.viewOriginZ + fz * this.viewSpanZ;

            for (int column = 0; column < this.columns; column++) {
                double fx = this.columns == 1
                    ? 0.5
                    : (double)column / (this.columns - 1);
                double x = this.viewOriginX + fx * this.viewSpanX;

                GeoWorldV1.RuntimeSample sample =
                    this.previewModel.sample(x, z);
                this.colors[row * this.columns + column] =
                    this.colorForLayer(x, z, sample);
                this.count(sample);
            }
        }
    }

    private void updateMapRect() {
        int header = this.layout != null
            ? this.layout.getHeaderHeight()
            : 27;
        int footer = this.layout != null
            ? this.layout.getFooterHeight()
            : 31;

        int availableHeight = Math.max(
            80,
            this.height - header - footer - 34
        );
        int mapAreaLeft = 12;
        int mapAreaWidth = Math.max(120, this.width - 24);

        double aspect = this.viewSpanX / this.viewSpanZ;
        if (!Double.isFinite(aspect) || aspect <= 0.0) {
            aspect = 1.0;
        }

        int widthFromHeight = (int)Math.round(availableHeight * aspect);
        if (widthFromHeight <= mapAreaWidth) {
            this.mapHeight = availableHeight;
            this.mapWidth = Math.max(1, widthFromHeight);
        } else {
            this.mapWidth = mapAreaWidth;
            this.mapHeight = Math.max(
                1,
                (int)Math.round(mapAreaWidth / aspect)
            );
        }

        this.mapLeft =
            mapAreaLeft + (mapAreaWidth - this.mapWidth) / 2;
        this.mapTop =
            header + 5 + (availableHeight - this.mapHeight) / 2;
    }

    @Override
    public void extractRenderState(
        GuiGraphicsExtractor graphics,
        int mouseX,
        int mouseY,
        float partialTick
    ) {
        super.extractRenderState(graphics, mouseX, mouseY, partialTick);
        this.updateMapRect();

        graphics.fill(
            this.mapLeft - 3,
            this.mapTop - 3,
            this.mapLeft + this.mapWidth + 3,
            this.mapTop + this.mapHeight + 3,
            0xFF101820
        );

        for (int row = 0; row < this.rows; row++) {
            int y0 =
                this.mapTop + row * this.mapHeight / this.rows;
            int y1 =
                this.mapTop + (row + 1) * this.mapHeight / this.rows;

            int runStart = 0;
            int runColor = this.colors[row * this.columns];
            for (int column = 1; column <= this.columns; column++) {
                int nextColor = column < this.columns
                    ? this.colors[row * this.columns + column]
                    : Integer.MIN_VALUE;
                if (column == this.columns || nextColor != runColor) {
                    int x0 =
                        this.mapLeft +
                        runStart * this.mapWidth / this.columns;
                    int x1 =
                        this.mapLeft +
                        column * this.mapWidth / this.columns;
                    graphics.fill(x0, y0, x1, y1, runColor);
                    if (column < this.columns) {
                        runStart = column;
                        runColor = nextColor;
                    }
                }
            }
        }

        this.renderFocusMarker(graphics);
        this.renderSelectionMarker(graphics);
        this.renderSummary(graphics);
        this.renderInfoPanel(graphics, mouseX, mouseY);
    }

    private void renderFocusMarker(GuiGraphicsExtractor graphics) {
        if (
            this.focusX == null ||
            this.focusZ == null ||
            this.viewSpanX <= 0.0 ||
            this.viewSpanZ <= 0.0 ||
            this.focusX < this.viewOriginX ||
            this.focusX > this.viewOriginX + this.viewSpanX ||
            this.focusZ < this.viewOriginZ ||
            this.focusZ > this.viewOriginZ + this.viewSpanZ
        ) {
            return;
        }

        double fx =
            (this.focusX - this.viewOriginX) / this.viewSpanX;
        double fz =
            (this.focusZ - this.viewOriginZ) / this.viewSpanZ;
        int markerX =
            this.mapLeft + (int)Math.round(fx * this.mapWidth);
        int markerY =
            this.mapTop + (int)Math.round(fz * this.mapHeight);

        float yaw = 0.0F;
        Minecraft minecraft = Minecraft.getInstance();
        if (minecraft.player != null) {
            yaw = minecraft.player.getYRot();
        }
        double radians = Math.toRadians(yaw);
        // Minecraft yaw 0 faces +Z. Map +Z points down the screen.
        double dirX = -Math.sin(radians);
        double dirY = Math.cos(radians);
        double sideX = dirY;
        double sideY = -dirX;

        int tipX = markerX + (int)Math.round(dirX * 8.0);
        int tipY = markerY + (int)Math.round(dirY * 8.0);
        int leftX = markerX - (int)Math.round(dirX * 2.0)
            + (int)Math.round(sideX * 4.0);
        int leftY = markerY - (int)Math.round(dirY * 2.0)
            + (int)Math.round(sideY * 4.0);
        int rightX = markerX - (int)Math.round(dirX * 2.0)
            - (int)Math.round(sideX * 4.0);
        int rightY = markerY - (int)Math.round(dirY * 2.0)
            - (int)Math.round(sideY * 4.0);
        int tailX = markerX - (int)Math.round(dirX * 5.0);
        int tailY = markerY - (int)Math.round(dirY * 5.0);

        int color = 0xFFFF3B30;
        this.drawMapLine(graphics, tipX, tipY, leftX, leftY, color);
        this.drawMapLine(graphics, tipX, tipY, rightX, rightY, color);
        this.drawMapLine(graphics, leftX, leftY, tailX, tailY, color);
        this.drawMapLine(graphics, rightX, rightY, tailX, tailY, color);
        this.drawMapLine(graphics, tailX, tailY, markerX, markerY, color);
    }

    private void renderSelectionMarker(GuiGraphicsExtractor graphics) {
        if (
            this.selectedX == null ||
            this.selectedZ == null ||
            this.selectedX < this.viewOriginX ||
            this.selectedX > this.viewOriginX + this.viewSpanX ||
            this.selectedZ < this.viewOriginZ ||
            this.selectedZ > this.viewOriginZ + this.viewSpanZ
        ) {
            return;
        }

        double fx =
            (this.selectedX - this.viewOriginX) / this.viewSpanX;
        double fz =
            (this.selectedZ - this.viewOriginZ) / this.viewSpanZ;
        int markerX =
            this.mapLeft + (int)Math.round(fx * this.mapWidth);
        int markerY =
            this.mapTop + (int)Math.round(fz * this.mapHeight);

        int color = 0xFFFFD54F;
        graphics.fill(markerX, markerY - 4, markerX + 1, markerY + 5, color);
        graphics.fill(markerX - 1, markerY - 3, markerX + 2, markerY + 4, color);
        graphics.fill(markerX - 2, markerY - 2, markerX + 3, markerY + 3, color);
        graphics.fill(markerX - 3, markerY - 1, markerX + 4, markerY + 2, color);
    }

    private void drawMapLine(
        GuiGraphicsExtractor graphics,
        int x0,
        int y0,
        int x1,
        int y1,
        int color
    ) {
        int dx = x1 - x0;
        int dy = y1 - y0;
        int steps = Math.max(Math.abs(dx), Math.abs(dy));
        if (steps == 0) {
            graphics.fill(x0, y0, x0 + 1, y0 + 1, color);
            return;
        }
        for (int i = 0; i <= steps; i++) {
            int x = x0 + (int)Math.round((double)dx * i / steps);
            int y = y0 + (int)Math.round((double)dy * i / steps);
            graphics.fill(x - 1, y - 1, x + 2, y + 2, color);
        }
    }

    private void renderSummary(GuiGraphicsExtractor graphics) {
        GeoWorldV1.Manifest manifest = this.previewModel.manifest();
        double sampleScale = Math.max(
            this.columns > 1
                ? this.viewSpanX / (this.columns - 1)
                : this.viewSpanX,
            this.rows > 1
                ? this.viewSpanZ / (this.rows - 1)
                : this.viewSpanZ
        );
        double zoom = manifest.geometry().spanXBlocks() / this.viewSpanX;

        Component summary = Component.translatable(
            "geoworld.preview.summary",
            manifest.seed(),
            String.format(Locale.ROOT, "%.1f", zoom),
            Math.round(this.viewSpanX),
            Math.round(this.viewSpanZ),
            String.format(Locale.ROOT, "%.1f", sampleScale)
        );

        int textY = this.mapTop + this.mapHeight + 6;
        graphics.text(
            this.font,
            summary,
            Math.max(6, (this.width - this.font.width(summary)) / 2),
            textY,
            0xFFFFFFFF
        );

        Component controls = Component.translatable(
            "geoworld.preview.controls"
        );
        graphics.text(
            this.font,
            controls,
            Math.max(
                6,
                (this.width - this.font.width(controls)) / 2
            ),
            textY + 11,
            0xFFB8B8B8
        );
    }

    private void renderInfoPanel(
        GuiGraphicsExtractor graphics,
        int mouseX,
        int mouseY
    ) {
        if (!this.isInsideMap(mouseX, mouseY) || this.draggingMap) {
            return;
        }

        double fx =
            (double)(mouseX - this.mapLeft) / this.mapWidth;
        double fz =
            (double)(mouseY - this.mapTop) / this.mapHeight;
        double worldX =
            this.viewOriginX + clamp(fx, 0.0, 1.0) * this.viewSpanX;
        double worldZ =
            this.viewOriginZ + clamp(fz, 0.0, 1.0) * this.viewSpanZ;

        GeoWorldV1.RuntimeSample sample =
            this.previewModel.sample(worldX, worldZ);
        String minecraftBiome =
            this.biomeIdResolver.biomeId(
                this.previewModel.lockedModel(),
                worldX,
                worldZ
            );

        int groundBoundary = this.heights.boundaryY(sample.groundHeight());
        int groundY = groundBoundary - 1;
        Component line1 = Component.translatable(
            "geoworld.preview.hover_position",
            Math.round(worldX),
            Math.round(worldZ),
            groundY
        );

        Component line2;
        if (
            sample.waterSurfaceDefined() &&
            sample.waterSurfaceElevation() != null
        ) {
            int waterBoundary = "River".equals(sample.waterKind())
                ? Math.max(
                    groundBoundary + 1,
                    this.heights.boundaryYCeil(sample.waterSurfaceElevation())
                )
                : this.heights.boundaryY(sample.waterSurfaceElevation());
            line2 = Component.translatable(
                "geoworld.preview.hover_water",
                localizedValue(sample.waterKind()),
                waterBoundary - 1,
                Math.max(0, waterBoundary - groundBoundary)
            );
        } else {
            line2 = Component.translatable(
                "geoworld.preview.hover_dry"
            );
        }

        GeoWorldV1.ClimateSample climate =
            this.previewModel.sampleClimate(worldX, worldZ);
        Component climateLine = climate.available()
            ? Component.translatable(
                "geoworld.preview.hover_climate",
                String.format(Locale.ROOT, "%.1f", climate.annualMeanTempC()),
                Math.round(climate.annualPrecipMM()),
                String.format(Locale.ROOT, "%.2f", climate.monsoonIndex())
            )
            : Component.translatable(
                "geoworld.preview.hover_climate_unavailable"
            );

        java.util.List<Component> lines = new java.util.ArrayList<>();
        lines.add(Component.translatable("geoworld.preview.info_title"));
        lines.add(line1);
        lines.add(line2);
        lines.add(Component.translatable(
            "geoworld.preview.hover_profile",
            localizedValue(sample.biomeFamily()),
            localizedValue(sample.terrainVariant()),
            localizedValue(sample.surfaceProfile())
        ));
        lines.add(Component.translatable(
            "geoworld.preview.hover_minecraft",
            localizedMinecraftBiome(minecraftBiome),
            localizedValue(sample.vegetationProfile())
        ));
        lines.add(climateLine);

        if (this.selectedX != null && this.selectedZ != null) {
            lines.add(Component.translatable(
                "geoworld.preview.selected",
                Math.round(this.selectedX),
                Math.round(this.selectedZ)
            ));
        }

        int wrapWidth = Math.max(180, Math.min(360, this.width / 2));
        java.util.ArrayList<net.minecraft.util.FormattedCharSequence> tooltip =
            new java.util.ArrayList<>();
        for (Component line : lines) {
            tooltip.addAll(this.font.split(line, wrapWidth));
        }
        graphics.setTooltipForNextFrame(
            this.font,
            tooltip,
            mouseX,
            mouseY
        );
    }
    @Override
    public boolean mouseClicked(
        MouseButtonEvent event,
        boolean doubleClick
    ) {
        this.updateMapRect();
        if (
            event.button() == PRIMARY_MOUSE_BUTTON &&
            this.isInsideMap(event.x(), event.y())
        ) {
            this.mapPressActive = true;
            this.mapPressMoved = false;
            this.mapPressTravel = 0.0;
            this.draggingMap = false;
            return true;
        }
        return super.mouseClicked(event, doubleClick);
    }

    @Override
    public boolean mouseDragged(
        MouseButtonEvent event,
        double dx,
        double dy
    ) {
        if (this.mapPressActive && event.button() == PRIMARY_MOUSE_BUTTON) {
            this.mapPressTravel += Math.hypot(dx, dy);
            if (this.mapPressTravel >= 3.0) {
                this.mapPressMoved = true;
                this.draggingMap = true;
            }
            if (this.draggingMap) {
                this.panByPixels(dx, dy);
            }
            return true;
        }
        return super.mouseDragged(event, dx, dy);
    }

    @Override
    public boolean mouseReleased(MouseButtonEvent event) {
        if (event.button() == PRIMARY_MOUSE_BUTTON && this.mapPressActive) {
            boolean wasClick =
                !this.mapPressMoved &&
                this.isInsideMap(event.x(), event.y());
            this.mapPressActive = false;
            this.draggingMap = false;
            if (wasClick) {
                this.selectMapPoint(event.x(), event.y());
            } else if (this.samplesDirty) {
                this.rebuildSamples();
            }
            return true;
        }
        return super.mouseReleased(event);
    }

    @Override
    public boolean mouseScrolled(
        double mouseX,
        double mouseY,
        double scrollX,
        double scrollY
    ) {
        this.updateMapRect();
        if (
            scrollY != 0.0 &&
            this.isInsideMap(mouseX, mouseY)
        ) {
            this.zoomAt(mouseX, mouseY, scrollY);
            return true;
        }
        return super.mouseScrolled(
            mouseX,
            mouseY,
            scrollX,
            scrollY
        );
    }

    private void zoomAt(
        double mouseX,
        double mouseY,
        double scrollY
    ) {
        GeoWorldV1.Geometry geometry = this.previewModel.manifest().geometry();

        double anchorXFraction = clamp(
            (mouseX - this.mapLeft) / this.mapWidth,
            0.0,
            1.0
        );
        double anchorZFraction = clamp(
            (mouseY - this.mapTop) / this.mapHeight,
            0.0,
            1.0
        );
        double anchorWorldX =
            this.viewOriginX +
            anchorXFraction * this.viewSpanX;
        double anchorWorldZ =
            this.viewOriginZ +
            anchorZFraction * this.viewSpanZ;

        double factor = Math.pow(ZOOM_STEP, scrollY);
        double newSpanX = clamp(
            this.viewSpanX * factor,
            Math.min(MIN_VIEW_SPAN_X, geometry.spanXBlocks()),
            geometry.spanXBlocks()
        );
        double newSpanZ = clamp(
            this.viewSpanZ * factor,
            Math.min(MIN_VIEW_SPAN_Z, geometry.spanZBlocks()),
            geometry.spanZBlocks()
        );

        this.viewOriginX = clampViewOrigin(
            anchorWorldX - anchorXFraction * newSpanX,
            geometry.originX(),
            geometry.spanXBlocks(),
            newSpanX
        );
        this.viewOriginZ = clampViewOrigin(
            anchorWorldZ - anchorZFraction * newSpanZ,
            geometry.originZ(),
            geometry.spanZBlocks(),
            newSpanZ
        );
        this.viewSpanX = newSpanX;
        this.viewSpanZ = newSpanZ;

        boolean isGlobal =
            Math.abs(this.viewSpanX - geometry.spanXBlocks()) < 1e-6 &&
            Math.abs(this.viewSpanZ - geometry.spanZBlocks()) < 1e-6;
        this.localView = !isGlobal;

        this.rebuildSamples();
        this.refreshViewModeButton();
    }

    private boolean isInsideMap(double x, double y) {
        return x >= this.mapLeft &&
            x < this.mapLeft + this.mapWidth &&
            y >= this.mapTop &&
            y < this.mapTop + this.mapHeight;
    }

    @Override
    protected void repositionElements() {
        if (this.layout != null) {
            this.layout.arrangeElements();
        }
        this.updateMapRect();
    }

    @Override
    public void onClose() {
        this.screenNavigator.setScreen(this.parent);
    }

    private static Component localizedValue(String raw) {
        if (raw == null || raw.isBlank()) {
            return Component.translatable("geoworld.value.unknown");
        }

        String slug = raw
            .toLowerCase(Locale.ROOT)
            .replaceAll("[^a-z0-9]+", "_")
            .replaceAll("^_+|_+$", "");
        String key = "geoworld.value." + slug;
        if (Language.getInstance().has(key)) {
            return Component.translatable(key);
        }
        return Component.literal(raw);
    }

    private static Component localizedMinecraftBiome(String biomeId) {
        if (biomeId == null || biomeId.isBlank()) {
            return Component.translatable("geoworld.value.unknown");
        }

        int separator = biomeId.indexOf(':');
        String namespace = separator >= 0
            ? biomeId.substring(0, separator)
            : "minecraft";
        String path = separator >= 0
            ? biomeId.substring(separator + 1)
            : biomeId;
        String key = "biome." + namespace + "." +
            path.replace('/', '.');
        if (Language.getInstance().has(key)) {
            return Component.translatable(key);
        }
        return Component.literal(biomeId);
    }

    private void count(GeoWorldV1.RuntimeSample sample) {
        switch (sample.waterKind()) {
            case "Ocean" -> this.oceanSamples++;
            case "Lake" -> this.lakeSamples++;
            case "River" -> this.riverSamples++;
            default -> {
                if (sample.sourceLand()) {
                    this.landSamples++;
                }
            }
        }
    }

    private int colorForLayer(
        double x,
        double z,
        GeoWorldV1.RuntimeSample sample
    ) {
        return switch (this.previewLayer) {
            case BIOME -> color(this.previewModel.lockedModel(), x, z, sample);
            case CONTINENT -> continentColor(sample);
            case TERRAIN -> terrainLayerColor(sample);
            case TEMPERATURE -> climateTemperatureColor(
                this.previewModel.sampleClimate(x, z)
            );
            case PRECIPITATION -> climatePrecipitationColor(
                this.previewModel.sampleClimate(x, z)
            );
            case MONSOON -> climateMonsoonColor(
                this.previewModel.sampleClimate(x, z)
            );
        };
    }

    private static int continentColor(GeoWorldV1.RuntimeSample sample) {
        return switch (sample.waterKind()) {
            case "Ocean" -> oceanColor(sample);
            case "Lake" -> 0xFF49A6D8;
            case "River" -> 0xFF67BDEA;
            default -> {
                double h = clamp(sample.groundHeight(), 0.0, 1.0);
                int low = 0xFF5E8E4C;
                int high = 0xFFB2A57C;
                yield mixColor(low, high, clamp((h - 0.48) / 0.35, 0, 1));
            }
        };
    }

    private static int terrainLayerColor(GeoWorldV1.RuntimeSample sample) {
        if ("Ocean".equals(sample.waterKind())) {
            return oceanColor(sample);
        }
        if ("Lake".equals(sample.waterKind())) {
            return 0xFF3F9FD0;
        }
        if ("River".equals(sample.waterKind())) {
            return 0xFF55B9E8;
        }
        double h = clamp(sample.groundHeight(), 0.0, 1.0);
        if (h < 0.50) {
            return mixColor(0xFF506E46, 0xFF7FA05A, h / 0.50);
        }
        if (h < 0.72) {
            return mixColor(
                0xFF7FA05A,
                0xFF9C8C69,
                (h - 0.50) / 0.22
            );
        }
        return mixColor(
            0xFF9C8C69,
            0xFFE7E8E4,
            clamp((h - 0.72) / 0.28, 0, 1)
        );
    }

    private static int climateTemperatureColor(
        GeoWorldV1.ClimateSample climate
    ) {
        if (!climate.available()) {
            return 0xFF555A60;
        }
        double t = clamp(
            (climate.annualMeanTempC() + 30.0) / 65.0,
            0.0,
            1.0
        );
        if (t < 0.5) {
            return mixColor(0xFF315A9E, 0xFFF0D96A, t * 2.0);
        }
        return mixColor(
            0xFFF0D96A,
            0xFFB73B2D,
            (t - 0.5) * 2.0
        );
    }

    private static int climatePrecipitationColor(
        GeoWorldV1.ClimateSample climate
    ) {
        if (!climate.available()) {
            return 0xFF555A60;
        }
        double t = clamp(climate.annualPrecipMM() / 3200.0, 0.0, 1.0);
        if (t < 0.5) {
            return mixColor(0xFFD0B567, 0xFF5E9A59, t * 2.0);
        }
        return mixColor(
            0xFF5E9A59,
            0xFF397EB2,
            (t - 0.5) * 2.0
        );
    }

    private static int climateMonsoonColor(
        GeoWorldV1.ClimateSample climate
    ) {
        if (!climate.available()) {
            return 0xFF555A60;
        }
        double m = clamp(climate.monsoonIndex(), -1.0, 1.0);
        if (m < 0.0) {
            return mixColor(0xFFB7694E, 0xFFDDD4B4, m + 1.0);
        }
        return mixColor(0xFFDDD4B4, 0xFF477FB8, m);
    }

    private static int mixColor(int a, int b, double t) {
        t = clamp(t, 0.0, 1.0);
        int ar = (a >> 16) & 0xFF;
        int ag = (a >> 8) & 0xFF;
        int ab = a & 0xFF;
        int br = (b >> 16) & 0xFF;
        int bg = (b >> 8) & 0xFF;
        int bb = b & 0xFF;
        int r = (int)Math.round(ar + (br - ar) * t);
        int g = (int)Math.round(ag + (bg - ag) * t);
        int bl = (int)Math.round(ab + (bb - ab) * t);
        return 0xFF000000 | (r << 16) | (g << 8) | bl;
    }

    private static int color(
        GeoWorldV1 model,
        double x,
        double z,
        GeoWorldV1.RuntimeSample sample
    ) {
        return switch (sample.waterKind()) {
            case "Ocean" -> oceanColor(sample);
            case "Lake" -> 0xFF3F9FD0;
            case "River" -> 0xFF4AAFE6;
            default -> landColor(sample);
        };
    }

    private static int oceanColor(GeoWorldV1.RuntimeSample sample) {
        double depth = Math.max(0.0, sample.waterDepth());
        if (depth > 0.45) return 0xFF173F73;
        if (depth > 0.20) return 0xFF235A8E;
        return 0xFF3477A7;
    }

    private static int landColor(GeoWorldV1.RuntimeSample sample) {
        String surface = sample.surfaceProfile();
        if (surface != null) {
            String surfaceValue = surface.toLowerCase(Locale.ROOT);
            if (
                surfaceValue.contains("snow") ||
                surfaceValue.contains("ice")
            ) {
                return 0xFFDCE5E2;
            }
            if (surfaceValue.contains("alpine rock")) {
                return 0xFF96958D;
            }
        }

        // The overview represents GeoWorld's ecological family layer.
        // Resolving the final Minecraft biome for every preview pixel performs
        // neighbourhood matching and dominates preview latency. The exact
        // Minecraft biome is still resolved for the hovered/selected point.
        String biome = sample.biomeFamily();
        if (biome == null) {
            return heightColor(sample.groundHeight());
        }

        String value = biome.toLowerCase(Locale.ROOT);
        if (value.contains("desert")) return 0xFFD8C27A;
        if (value.contains("savanna")) return 0xFFB6B65D;
        if (value.contains("tundra")) return 0xFFB8C4B5;
        if (
            value.contains("taiga") ||
            value.contains("conifer")
        ) {
            return 0xFF496B55;
        }
        if (
            value.contains("rainforest") ||
            value.contains("tropical")
        ) {
            return 0xFF2D7546;
        }
        if (value.contains("forest")) return 0xFF4E8150;
        if (value.contains("wetland")) return 0xFF5F8C69;
        if (value.contains("grass")) return 0xFF79A85B;
        if (value.contains("shrub")) return 0xFF839260;
        if (
            value.contains("snow") ||
            value.contains("ice")
        ) {
            return 0xFFDCE5E2;
        }
        return heightColor(sample.groundHeight());
    }

    private static double clampViewOrigin(
        double origin,
        double geometryOrigin,
        double geometrySpan,
        double viewSpan
    ) {
        double max = geometryOrigin + geometrySpan - viewSpan;
        if (max < geometryOrigin) {
            return geometryOrigin;
        }
        return clamp(origin, geometryOrigin, max);
    }

    private static double clamp(
        double value,
        double min,
        double max
    ) {
        if (max < min) {
            return min;
        }
        return Math.max(min, Math.min(max, value));
    }

    private static int heightColor(double height) {
        if (height > 0.82) return 0xFF9B9A91;
        if (height > 0.68) return 0xFF7C8563;
        if (height > 0.52) return 0xFF6E9856;
        return 0xFF83A963;
    }
}
