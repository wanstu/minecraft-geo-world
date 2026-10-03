package geoworld.neoforge.client;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.neoforge.worldgen.GeoWorldBiomeSource;
import geoworld.runtime.GeoWorldV1;

import net.minecraft.client.gui.GuiGraphicsExtractor;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.layouts.HeaderAndFooterLayout;
import net.minecraft.client.gui.layouts.LinearLayout;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.locale.Language;
import net.minecraft.network.chat.CommonComponents;
import net.minecraft.network.chat.Component;

import java.util.Locale;

public final class GeoWorldPreviewScreen extends Screen {
    private static final Component TITLE =
        Component.translatable("geoworld.preview.title");

    private static final int MAX_COLUMNS = 160;
    private static final int MAX_ROWS = 100;
    private static final double INITIAL_LOCAL_SPAN_X = 1600.0;
    private static final double INITIAL_LOCAL_SPAN_Z = 1000.0;
    private static final double MIN_VIEW_SPAN_X = 128.0;
    private static final double MIN_VIEW_SPAN_Z = 80.0;
    private static final double ZOOM_STEP = 0.78;

    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    private final Screen parent;
    private final GeoWorldV1 model;
    private final Double focusX;
    private final Double focusZ;

    private HeaderAndFooterLayout layout;
    private Button viewModeButton;

    private boolean localView;
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

    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        this(parent, generated.model(), null, null);
    }

    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldV1 model,
        Double focusX,
        Double focusZ
    ) {
        super(TITLE);
        this.parent = parent;
        this.model = model;
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

        footer.addChild(
            Button.builder(
                Component.translatable("geoworld.preview.reset_view"),
                button -> this.resetCurrentView()
            )
                .width(120)
                .build()
        );

        footer.addChild(
            Button.builder(CommonComponents.GUI_BACK, button -> this.onClose())
                .width(120)
                .build()
        );

        this.layout.visitWidgets(this::addRenderableWidget);
        this.repositionElements();
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
        GeoWorldV1.Geometry geometry = this.model.manifest().geometry();
        this.localView = false;
        this.viewOriginX = geometry.originX();
        this.viewOriginZ = geometry.originZ();
        this.viewSpanX = geometry.spanXBlocks();
        this.viewSpanZ = geometry.spanZBlocks();
        this.rebuildSamples();
    }

    private void resetLocalView() {
        GeoWorldV1.Geometry geometry = this.model.manifest().geometry();
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
                    this.model.sample(x, z);
                this.colors[row * this.columns + column] =
                    color(this.model, x, z, sample);
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

        int availableWidth = Math.max(120, this.width - 32);
        int availableHeight = Math.max(
            80,
            this.height - header - footer - 34
        );

        double aspect = this.viewSpanX / this.viewSpanZ;
        if (!Double.isFinite(aspect) || aspect <= 0.0) {
            aspect = 1.0;
        }

        int widthFromHeight = (int)Math.round(availableHeight * aspect);
        if (widthFromHeight <= availableWidth) {
            this.mapHeight = availableHeight;
            this.mapWidth = Math.max(1, widthFromHeight);
        } else {
            this.mapWidth = availableWidth;
            this.mapHeight = Math.max(
                1,
                (int)Math.round(availableWidth / aspect)
            );
        }

        this.mapLeft = (this.width - this.mapWidth) / 2;
        this.mapTop = header + 5;
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
        this.renderSummary(graphics);

        if (this.isInsideMap(mouseX, mouseY)) {
            this.renderHoverInfo(graphics, mouseX, mouseY);
        }
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

        graphics.fill(
            markerX - 5,
            markerY - 1,
            markerX + 6,
            markerY + 2,
            0xFFFF3B30
        );
        graphics.fill(
            markerX - 1,
            markerY - 5,
            markerX + 2,
            markerY + 6,
            0xFFFF3B30
        );
    }

    private void renderSummary(GuiGraphicsExtractor graphics) {
        GeoWorldV1.Manifest manifest = this.model.manifest();
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

    private void renderHoverInfo(
        GuiGraphicsExtractor graphics,
        int mouseX,
        int mouseY
    ) {
        double fx =
            (double)(mouseX - this.mapLeft) / this.mapWidth;
        double fz =
            (double)(mouseY - this.mapTop) / this.mapHeight;
        double worldX =
            this.viewOriginX + clamp(fx, 0.0, 1.0) * this.viewSpanX;
        double worldZ =
            this.viewOriginZ + clamp(fz, 0.0, 1.0) * this.viewSpanZ;

        GeoWorldV1.RuntimeSample sample =
            this.model.sample(worldX, worldZ);
        String minecraftBiome =
            GeoWorldBiomeSource.minecraftBiomeId(
                this.model,
                worldX,
                worldZ
            );

        int groundBoundary = HEIGHTS.boundaryY(sample.groundHeight());
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
            int waterBoundary = HEIGHTS.boundaryY(
                sample.waterSurfaceElevation()
            );
            int depthBlocks = Math.max(
                0,
                waterBoundary - groundBoundary
            );
            line2 = Component.translatable(
                "geoworld.preview.hover_water",
                localizedValue(sample.waterKind()),
                waterBoundary - 1,
                depthBlocks
            );
        } else {
            line2 = Component.translatable(
                "geoworld.preview.hover_dry"
            );
        }

        Component line3 = Component.translatable(
            "geoworld.preview.hover_profile",
            localizedValue(sample.biomeFamily()),
            localizedValue(sample.terrainVariant()),
            localizedValue(sample.surfaceProfile())
        );
        Component line4 = Component.translatable(
            "geoworld.preview.hover_minecraft",
            localizedMinecraftBiome(minecraftBiome),
            localizedValue(sample.vegetationProfile())
        );

        Component[] lines = {line1, line2, line3, line4};
        int boxWidth = 0;
        for (Component line : lines) {
            boxWidth = Math.max(boxWidth, this.font.width(line));
        }
        int boxHeight = lines.length * 11 + 7;

        int x = mouseX + 12;
        int y = mouseY + 12;
        if (x + boxWidth + 10 > this.width) {
            x = mouseX - boxWidth - 14;
        }
        if (y + boxHeight + 4 > this.height) {
            y = mouseY - boxHeight - 8;
        }

        graphics.fill(
            x - 4,
            y - 4,
            x + boxWidth + 5,
            y + boxHeight,
            0xE0101418
        );

        for (int i = 0; i < lines.length; i++) {
            graphics.text(
                this.font,
                lines[i],
                x,
                y + i * 11,
                i == 0 ? 0xFFFFFFFF : 0xFFD7D7D7
            );
        }
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
        GeoWorldV1.Geometry geometry = this.model.manifest().geometry();

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
        this.minecraft.gui.setScreen(this.parent);
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
            default -> landColor(model, x, z, sample);
        };
    }

    private static int oceanColor(GeoWorldV1.RuntimeSample sample) {
        double depth = Math.max(0.0, sample.waterDepth());
        if (depth > 0.45) return 0xFF173F73;
        if (depth > 0.20) return 0xFF235A8E;
        return 0xFF3477A7;
    }

    private static int landColor(
        GeoWorldV1 model,
        double x,
        double z,
        GeoWorldV1.RuntimeSample sample
    ) {
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

        String minecraftBiome =
            GeoWorldBiomeSource.minecraftBiomeId(model, x, z);
        if ("minecraft:jungle".equals(minecraftBiome)) {
            return 0xFF2D7546;
        }
        if ("minecraft:sparse_jungle".equals(minecraftBiome)) {
            return 0xFF43845A;
        }
        if ("minecraft:forest".equals(minecraftBiome)) {
            return 0xFF4E8150;
        }
        if ("minecraft:birch_forest".equals(minecraftBiome)) {
            return 0xFF67935A;
        }

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
