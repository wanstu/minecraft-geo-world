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
import net.minecraft.network.chat.CommonComponents;
import net.minecraft.network.chat.Component;

import java.util.Locale;

public final class GeoWorldPreviewScreen extends Screen {
    private static final Component TITLE =
        Component.translatable("geoworld.preview.title");
    private static final int MAX_COLUMNS = 128;
    private static final int MAX_ROWS = 80;
    private static final double LOCAL_BLOCKS_PER_SAMPLE = 8.0;
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
        this.localView = focusX != null && focusZ != null;
        this.buildPreview();
    }

    @Override
    protected void init() {
        this.layout = new HeaderAndFooterLayout(this, 29, 33);
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
                    .width(150)
                    .build()
            );
        }
        footer.addChild(
            Button.builder(CommonComponents.GUI_BACK, button -> this.onClose())
                .width(150)
                .build()
        );
        this.layout.visitWidgets(this::addRenderableWidget);
        this.repositionElements();
    }

    private void toggleViewMode() {
        if (this.focusX == null || this.focusZ == null) {
            return;
        }
        this.localView = !this.localView;
        this.buildPreview();
        if (this.viewModeButton != null) {
            this.viewModeButton.setMessage(this.viewModeLabel());
        }
    }

    private Component viewModeLabel() {
        return Component.translatable(
            this.localView
                ? "geoworld.preview.show_global"
                : "geoworld.preview.show_local"
        );
    }

    private void buildPreview() {
        GeoWorldV1.Geometry geometry =
            this.model.manifest().geometry();

        if (
            this.localView &&
            this.focusX != null &&
            this.focusZ != null
        ) {
            this.columns = MAX_COLUMNS;
            this.rows = MAX_ROWS;
            this.viewSpanX = Math.min(
                geometry.spanXBlocks(),
                (this.columns - 1) * LOCAL_BLOCKS_PER_SAMPLE
            );
            this.viewSpanZ = Math.min(
                geometry.spanZBlocks(),
                (this.rows - 1) * LOCAL_BLOCKS_PER_SAMPLE
            );
            this.viewOriginX = clamp(
                this.focusX - this.viewSpanX * 0.5,
                geometry.originX(),
                geometry.originX() +
                    geometry.spanXBlocks() -
                    this.viewSpanX
            );
            this.viewOriginZ = clamp(
                this.focusZ - this.viewSpanZ * 0.5,
                geometry.originZ(),
                geometry.originZ() +
                    geometry.spanZBlocks() -
                    this.viewSpanZ
            );
        } else {
            double aspect =
                geometry.spanXBlocks() / geometry.spanZBlocks();
            if (!Double.isFinite(aspect) || aspect <= 0.0) {
                aspect = 1.0;
            }

            if (aspect >= 1.0) {
                this.columns = MAX_COLUMNS;
                this.rows = Math.max(
                    24,
                    Math.min(
                        MAX_ROWS,
                        (int)Math.round(MAX_COLUMNS / aspect)
                    )
                );
            } else {
                this.rows = MAX_ROWS;
                this.columns = Math.max(
                    24,
                    Math.min(
                        MAX_COLUMNS,
                        (int)Math.round(MAX_ROWS * aspect)
                    )
                );
            }
            this.viewOriginX = geometry.originX();
            this.viewOriginZ = geometry.originZ();
            this.viewSpanX = geometry.spanXBlocks();
            this.viewSpanZ = geometry.spanZBlocks();
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
                count(sample);
            }
        }
    }

    @Override
    public void extractRenderState(
        GuiGraphicsExtractor graphics,
        int mouseX,
        int mouseY,
        float partialTick
    ) {
        super.extractRenderState(graphics, mouseX, mouseY, partialTick);

        int availableWidth = Math.max(80, this.width - 24);
        int availableHeight = Math.max(
            60,
            this.height -
            this.layout.getHeaderHeight() -
            this.layout.getFooterHeight() -
            70
        );

        int pixel = Math.max(
            1,
            Math.min(
                availableWidth / this.columns,
                availableHeight / this.rows
            )
        );
        int mapWidth = this.columns * pixel;
        int mapHeight = this.rows * pixel;
        int left = (this.width - mapWidth) / 2;
        int top = this.layout.getHeaderHeight() + 8;

        graphics.fill(
            left - 2,
            top - 2,
            left + mapWidth + 2,
            top + mapHeight + 2,
            0xFF101820
        );

        for (int row = 0; row < this.rows; row++) {
            int y0 = top + row * pixel;
            int y1 = y0 + pixel;
            int runStart = 0;
            int runColor = this.colors[row * this.columns];

            for (int column = 1; column <= this.columns; column++) {
                int nextColor = column < this.columns
                    ? this.colors[row * this.columns + column]
                    : Integer.MIN_VALUE;
                if (column == this.columns || nextColor != runColor) {
                    graphics.fill(
                        left + runStart * pixel,
                        y0,
                        left + column * pixel,
                        y1,
                        runColor
                    );
                    if (column < this.columns) {
                        runStart = column;
                        runColor = nextColor;
                    }
                }
            }
        }

        GeoWorldV1.Manifest manifest = this.model.manifest();
        boolean focusVisible =
            this.focusX != null &&
            this.focusZ != null &&
            this.viewSpanX > 0.0 &&
            this.viewSpanZ > 0.0 &&
            this.focusX >= this.viewOriginX &&
            this.focusX <= this.viewOriginX + this.viewSpanX &&
            this.focusZ >= this.viewOriginZ &&
            this.focusZ <= this.viewOriginZ + this.viewSpanZ;
        if (focusVisible) {
            double fx =
                (this.focusX - this.viewOriginX) / this.viewSpanX;
            double fz =
                (this.focusZ - this.viewOriginZ) / this.viewSpanZ;
            int markerX = left + (int)Math.round(fx * mapWidth);
            int markerY = top + (int)Math.round(fz * mapHeight);
            graphics.fill(
                markerX - 4,
                markerY - 1,
                markerX + 5,
                markerY + 2,
                0xFFFF3B30
            );
            graphics.fill(
                markerX - 1,
                markerY - 4,
                markerX + 2,
                markerY + 5,
                0xFFFF3B30
            );
        }

        double sampleScale = Math.max(
            this.columns > 1
                ? this.viewSpanX / (this.columns - 1)
                : this.viewSpanX,
            this.rows > 1
                ? this.viewSpanZ / (this.rows - 1)
                : this.viewSpanZ
        );
        String summary = String.format(
            Locale.ROOT,
            "Seed %d · %s · %.0f×%.0f blocks · %.1f blocks/sample",
            manifest.seed(),
            Component.translatable(
                this.localView
                    ? "geoworld.preview.local_view"
                    : "geoworld.preview.global_view"
            ).getString(),
            this.viewSpanX,
            this.viewSpanZ,
            sampleScale
        );
        int textY = top + mapHeight + 8;
        graphics.text(
            this.font,
            summary,
            Math.max(6, (this.width - this.font.width(summary)) / 2),
            textY,
            0xFFFFFFFF
        );

        if (this.focusX != null && this.focusZ != null) {
            GeoWorldV1.RuntimeSample focusSample =
                this.model.sample(this.focusX, this.focusZ);
            int surfaceY =
                HEIGHTS.boundaryY(focusSample.groundHeight()) - 1;
            Component positionText = Component.translatable(
                "geoworld.preview.current_detail",
                Math.round(this.focusX),
                Math.round(this.focusZ),
                surfaceY,
                focusSample.biomeFamily(),
                focusSample.surfaceProfile()
            );
            graphics.text(
                this.font,
                positionText,
                Math.max(
                    6,
                    (this.width - this.font.width(positionText)) / 2
                ),
                textY + 11,
                focusVisible ? 0xFFFF7770 : 0xFFB0B0B0
            );
        } else {
            String counts = String.format(
                Locale.ROOT,
                "land %d · ocean %d · lake %d · river %d",
                this.landSamples,
                this.oceanSamples,
                this.lakeSamples,
                this.riverSamples
            );
            graphics.text(
                this.font,
                counts,
                Math.max(6, (this.width - this.font.width(counts)) / 2),
                textY + 11,
                0xFFB0B0B0
            );
        }
    }

    @Override
    protected void repositionElements() {
        this.layout.arrangeElements();
    }

    @Override
    public void onClose() {
        this.minecraft.gui.setScreen(this.parent);
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
            case "River" -> 0xFF55B9E8;
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
            if (surfaceValue.contains("snow") || surfaceValue.contains("ice")) {
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
        if (value.contains("taiga") || value.contains("conifer")) {
            return 0xFF496B55;
        }
        if (value.contains("rainforest") || value.contains("tropical")) {
            return 0xFF2D7546;
        }
        if (value.contains("forest")) return 0xFF4E8150;
        if (value.contains("wetland")) return 0xFF5F8C69;
        if (value.contains("grass")) return 0xFF79A85B;
        if (value.contains("shrub")) return 0xFF839260;
        if (value.contains("snow") || value.contains("ice")) {
            return 0xFFDCE5E2;
        }
        return heightColor(sample.groundHeight());
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
