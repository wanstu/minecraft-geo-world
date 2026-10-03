package geoworld.neoforge.client;

import geoworld.neoforge.engine.GeoWorldEngineBridge;

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

    private final Screen parent;
    private final GeoWorldV1 model;
    private final Double focusX;
    private final Double focusZ;
    private final HeaderAndFooterLayout layout =
        new HeaderAndFooterLayout(this, 29, 33);

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
        this.buildPreview();
    }

    @Override
    protected void init() {
        this.layout.addTitleHeader(this.title, this.font);
        LinearLayout footer = this.layout.addToFooter(
            LinearLayout.horizontal().spacing(8)
        );
        footer.addChild(
            Button.builder(CommonComponents.GUI_BACK, button -> this.onClose())
                .width(150)
                .build()
        );
        this.layout.visitWidgets(this::addRenderableWidget);
        this.repositionElements();
    }

    private void buildPreview() {
        GeoWorldV1.Geometry geometry =
            this.model.manifest().geometry();

        double aspect = geometry.spanXBlocks() / geometry.spanZBlocks();
        if (!Double.isFinite(aspect) || aspect <= 0.0) {
            aspect = 1.0;
        }

        if (aspect >= 1.0) {
            this.columns = MAX_COLUMNS;
            this.rows = Math.max(
                24,
                Math.min(MAX_ROWS, (int)Math.round(MAX_COLUMNS / aspect))
            );
        } else {
            this.rows = MAX_ROWS;
            this.columns = Math.max(
                24,
                Math.min(MAX_COLUMNS, (int)Math.round(MAX_ROWS * aspect))
            );
        }

        this.colors = new int[Math.multiplyExact(this.columns, this.rows)];

        for (int row = 0; row < this.rows; row++) {
            double fz = this.rows == 1
                ? 0.5
                : (double)row / (this.rows - 1);
            double z = geometry.originZ() + fz * geometry.spanZBlocks();

            for (int column = 0; column < this.columns; column++) {
                double fx = this.columns == 1
                    ? 0.5
                    : (double)column / (this.columns - 1);
                double x = geometry.originX() + fx * geometry.spanXBlocks();

                GeoWorldV1.RuntimeSample sample =
                    this.model.sample(x, z);
                this.colors[row * this.columns + column] = color(sample);
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
            54
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
        GeoWorldV1.Geometry geometry = manifest.geometry();
        boolean focusVisible =
            this.focusX != null &&
            this.focusZ != null &&
            geometry.spanXBlocks() > 0 &&
            geometry.spanZBlocks() > 0 &&
            this.focusX >= geometry.originX() &&
            this.focusX <= geometry.originX() + geometry.spanXBlocks() &&
            this.focusZ >= geometry.originZ() &&
            this.focusZ <= geometry.originZ() + geometry.spanZBlocks();
        if (focusVisible) {
            double fx =
                (this.focusX - geometry.originX()) / geometry.spanXBlocks();
            double fz =
                (this.focusZ - geometry.originZ()) / geometry.spanZBlocks();
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

        String summary = String.format(
            Locale.ROOT,
            "Seed %d · %dx%d nodes · %.0fx%.0f blocks · land %d · ocean %d · lake %d · river %d",
            manifest.seed(),
            manifest.geometry().width(),
            manifest.geometry().height(),
            manifest.geometry().spanXBlocks(),
            manifest.geometry().spanZBlocks(),
            this.landSamples,
            this.oceanSamples,
            this.lakeSamples,
            this.riverSamples
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
            Component positionText = Component.translatable(
                "geoworld.preview.current_position",
                Math.round(this.focusX),
                Math.round(this.focusZ)
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

    private static int color(GeoWorldV1.RuntimeSample sample) {
        return switch (sample.waterKind()) {
            case "Ocean" -> oceanColor(sample);
            case "Lake" -> 0xFF3F9FD0;
            case "River" -> 0xFF55B9E8;
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
            if (surfaceValue.contains("snow") || surfaceValue.contains("ice")) {
                return 0xFFDCE5E2;
            }
            if (surfaceValue.contains("alpine rock")) {
                return 0xFF96958D;
            }
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

    private static int heightColor(double height) {
        if (height > 0.82) return 0xFF9B9A91;
        if (height > 0.68) return 0xFF7C8563;
        if (height > 0.52) return 0xFF6E9856;
        return 0xFF83A963;
    }
}
