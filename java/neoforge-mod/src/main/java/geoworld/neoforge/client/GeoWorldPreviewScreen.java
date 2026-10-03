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
    private final GeoWorldEngineBridge.GeneratedModel generated;
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
        super(TITLE);
        this.parent = parent;
        this.generated = generated;
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
            this.generated.model().manifest().geometry();

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
                    this.generated.model().sample(x, z);
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
            42
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

        GeoWorldV1.Manifest manifest = this.generated.model().manifest();
        String summary = String.format(
            Locale.ROOT,
            "Seed %d 路 %dx%d nodes 路 %.0fx%.0f blocks 路 land %d 路 ocean %d 路 lake %d 路 river %d",
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
