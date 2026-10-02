package geoworld.neoforge.client;

import geoworld.neoforge.GeoWorldMod;
import geoworld.neoforge.engine.GeoWorldEngineBridge;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.neoforge.worldgen.GeoWorldBiomeSource;
import geoworld.neoforge.worldgen.GeoWorldChunkGenerator;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.components.CycleButton;
import net.minecraft.client.gui.components.EditBox;
import net.minecraft.client.gui.components.StringWidget;
import net.minecraft.client.gui.layouts.CommonLayouts;
import net.minecraft.client.gui.layouts.HeaderAndFooterLayout;
import net.minecraft.client.gui.layouts.LinearLayout;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.client.gui.screens.worldselection.CreateWorldScreen;
import net.minecraft.client.gui.screens.worldselection.WorldCreationContext;
import net.minecraft.core.registries.Registries;
import net.minecraft.network.chat.CommonComponents;
import net.minecraft.network.chat.Component;

import java.util.List;
import java.util.concurrent.ThreadLocalRandom;

public final class GeoWorldPresetScreen extends Screen {
    private static final Component TITLE =
        Component.translatable("geoworld.create.title");
    private static final Component SEED_LABEL =
        Component.translatable("geoworld.create.seed");
    private static final Component WIDTH_LABEL =
        Component.translatable("geoworld.create.width");
    private static final Component HEIGHT_LABEL =
        Component.translatable("geoworld.create.height");
    private static final Component CELL_SIZE_LABEL =
        Component.translatable("geoworld.create.cell_size");
    private static final Component PLATES_LABEL =
        Component.translatable("geoworld.create.plates");
    private static final Component OCEAN_LABEL =
        Component.translatable("geoworld.create.ocean_fraction");
    private static final Component FRAGMENTATION_LABEL =
        Component.translatable("geoworld.create.fragmentation");
    private static final Component SEPARATION_LABEL =
        Component.translatable("geoworld.create.separation");
    private static final Component PRESET_LABEL =
        Component.translatable("geoworld.create.continent_preset");

    private final CreateWorldScreen parent;
    private final HeaderAndFooterLayout layout =
        new HeaderAndFooterLayout(this, 33, 58);

    private GeoWorldCreationConfig initialConfig;
    private ContinentPreset selectedPreset;
    private boolean applyingPreset;

    private EditBox seedEdit;
    private EditBox widthEdit;
    private EditBox heightEdit;
    private EditBox cellSizeEdit;
    private EditBox platesEdit;
    private EditBox oceanEdit;
    private EditBox fragmentationEdit;
    private EditBox separationEdit;
    private CycleButton<ContinentPreset> presetButton;
    private StringWidget statusText;
    private StringWidget spanText;
    private Button previewButton;
    private Button viewMapButton;
    private Button doneButton;
    private boolean previewGenerating;
    private int generationSerial;

    public GeoWorldPresetScreen(
        CreateWorldScreen parent,
        WorldCreationContext settings
    ) {
        super(TITLE);
        this.parent = parent;
        this.initialConfig =
            settings.selectedDimensions().overworld()
                instanceof GeoWorldChunkGenerator generator
                    ? generator.config()
                    : GeoWorldCreationSession.get(parent.getUiState());
        this.selectedPreset = this.initialConfig.continentPreset();
    }

    @Override
    protected void init() {
        this.layout.addTitleHeader(this.title, this.font);

        LinearLayout body = this.layout.addToContents(
            LinearLayout.vertical().spacing(4)
        );
        body.defaultCellSetting().alignHorizontallyCenter();

        this.seedEdit = numberBox(
            230,
            SEED_LABEL,
            this.parent.getUiState().getSeed()
        );
        this.seedEdit.setResponder(value -> {
            this.parent.getUiState().setSeed(value);
            this.generationSerial++;
            GeoWorldCreationSession.invalidateGenerated(
                this.parent.getUiState()
            );
            this.validateForm();
        });
        LinearLayout seedRow = LinearLayout.horizontal().spacing(6);
        seedRow.addChild(this.seedEdit);
        seedRow.addChild(
            Button.builder(
                Component.translatable("geoworld.create.random_seed"),
                button -> this.seedEdit.setValue(
                    Long.toString(ThreadLocalRandom.current().nextLong())
                )
            ).width(74).build()
        );
        body.addChild(new StringWidget(SEED_LABEL, this.font));
        body.addChild(seedRow);

        this.presetButton = CycleButton
            .builder(ContinentPreset::displayName, this.selectedPreset)
            .withValues(List.of(ContinentPreset.values()))
            .create(
                0,
                0,
                310,
                20,
                PRESET_LABEL,
                (button, preset) -> this.applyPreset(preset)
            );
        body.addChild(this.presetButton);

        this.widthEdit = addNumber(
            body,
            WIDTH_LABEL,
            Integer.toString(this.initialConfig.width())
        );
        this.heightEdit = addNumber(
            body,
            HEIGHT_LABEL,
            Integer.toString(this.initialConfig.height())
        );
        this.cellSizeEdit = addNumber(
            body,
            CELL_SIZE_LABEL,
            Integer.toString(this.initialConfig.cellSizeBlocks())
        );
        this.platesEdit = addNumber(
            body,
            PLATES_LABEL,
            Integer.toString(this.initialConfig.plateCount())
        );
        this.oceanEdit = addNumber(
            body,
            OCEAN_LABEL,
            formatFraction(this.initialConfig.oceanFraction())
        );
        this.fragmentationEdit = addNumber(
            body,
            FRAGMENTATION_LABEL,
            formatFraction(this.initialConfig.continentFragmentation())
        );
        this.separationEdit = addNumber(
            body,
            SEPARATION_LABEL,
            formatFraction(this.initialConfig.continentSeparation())
        );

        this.widthEdit.setResponder(_ -> this.onConfigEdited());
        this.heightEdit.setResponder(_ -> this.onConfigEdited());
        this.cellSizeEdit.setResponder(_ -> this.onConfigEdited());
        this.platesEdit.setResponder(_ -> this.onConfigEdited());
        this.oceanEdit.setResponder(_ -> this.onContinentNumberEdited());
        this.fragmentationEdit.setResponder(_ -> this.onContinentNumberEdited());
        this.separationEdit.setResponder(_ -> this.onContinentNumberEdited());

        this.spanText = body.addChild(
            new StringWidget(Component.empty(), this.font)
        );
        LinearLayout previewRow = LinearLayout.horizontal().spacing(6);
        this.previewButton = previewRow.addChild(
            Button.builder(
                Component.translatable("geoworld.create.generate_preview"),
                button -> this.generatePreview()
            ).width(152).build()
        );
        this.viewMapButton = previewRow.addChild(
            Button.builder(
                Component.translatable("geoworld.create.view_map"),
                button -> this.openPreviewMap()
            ).width(152).build()
        );
        body.addChild(previewRow);
        this.statusText = body.addChild(
            new StringWidget(
                Component.translatable("geoworld.create.preview_pending"),
                this.font
            )
        );

        LinearLayout footer = this.layout.addToFooter(
            LinearLayout.horizontal().spacing(8)
        );
        this.doneButton = footer.addChild(
            Button.builder(CommonComponents.GUI_DONE, button -> this.applyAndClose())
                .width(150)
                .build()
        );
        footer.addChild(
            Button.builder(CommonComponents.GUI_CANCEL, button -> this.onClose())
                .width(150)
                .build()
        );

        this.layout.visitWidgets(this::addRenderableWidget);
        this.repositionElements();
        this.validateForm();
        this.restoreCachedPreview();
    }

    private EditBox addNumber(
        LinearLayout body,
        Component label,
        String value
    ) {
        EditBox box = numberBox(310, label, value);
        body.addChild(CommonLayouts.labeledElement(this.font, box, label));
        return box;
    }

    private EditBox numberBox(int width, Component narration, String value) {
        EditBox box = new EditBox(this.font, width, 20, narration);
        box.setValue(value);
        box.setMaxLength(32);
        return box;
    }

    private void applyPreset(ContinentPreset preset) {
        this.selectedPreset = preset;
        if (preset == ContinentPreset.CUSTOM) {
            this.onConfigEdited();
            return;
        }

        this.applyingPreset = true;
        try {
            this.oceanEdit.setValue(formatFraction(preset.oceanFraction()));
            this.fragmentationEdit.setValue(formatFraction(preset.fragmentation()));
            this.separationEdit.setValue(formatFraction(preset.separation()));
        } finally {
            this.applyingPreset = false;
        }
        this.onConfigEdited();
    }

    private void onContinentNumberEdited() {
        if (this.applyingPreset) {
            return;
        }
        try {
            double ocean = Double.parseDouble(this.oceanEdit.getValue());
            double fragmentation =
                Double.parseDouble(this.fragmentationEdit.getValue());
            double separation =
                Double.parseDouble(this.separationEdit.getValue());
            this.selectedPreset = GeoWorldCreationConfig.detectPreset(
                ocean,
                fragmentation,
                separation
            );
            this.presetButton.setValue(this.selectedPreset);
        } catch (NumberFormatException ignored) {
            this.selectedPreset = ContinentPreset.CUSTOM;
            this.presetButton.setValue(this.selectedPreset);
        }
        this.onConfigEdited();
    }

    private void onConfigEdited() {
        this.generationSerial++;
        this.previewGenerating = false;
        GeoWorldCreationSession.invalidateGenerated(
            this.parent.getUiState()
        );
        this.validateForm();
    }

    private void validateForm() {
        if (
            this.doneButton == null ||
            this.previewButton == null ||
            this.viewMapButton == null ||
            this.statusText == null
        ) {
            return;
        }

        try {
            GeoWorldCreationConfig config = this.readConfig();
            List<String> errors = config.validationErrors();
            if (!errors.isEmpty()) {
                this.doneButton.active = false;
                this.previewButton.active = false;
                this.viewMapButton.active = false;
                this.statusText.setMessage(
                    Component.literal(errors.getFirst())
                );
                this.spanText.setMessage(Component.empty());
                return;
            }

            GeoWorldCreationSession.set(
                this.parent.getUiState(),
                config
            );
            this.previewButton.active = !this.previewGenerating;
            this.spanText.setMessage(
                Component.translatable(
                    "geoworld.create.span",
                    config.spanXBlocks(),
                    config.spanZBlocks()
                )
            );

            long seed = this.syncSeedFromEditor();
            var generated = GeoWorldCreationSession
                .generated(this.parent.getUiState())
                .filter(value ->
                    value.seed() == seed &&
                    value.config().equals(config)
                );

            if (generated.isPresent()) {
                GeoWorldEngineBridge.GeneratedModel model = generated.get();
                this.doneButton.active = true;
                this.viewMapButton.active = true;
                this.statusText.setMessage(
                    Component.translatable(
                        "geoworld.create.preview_ready",
                        model.model().manifest().schemaVersion(),
                        model.model().manifest().generatorVersion(),
                        model.durationMs(),
                        model.cached()
                            ? Component.translatable(
                                "geoworld.create.preview_cache_hit"
                            )
                            : Component.translatable(
                                "geoworld.create.preview_generated"
                            )
                    )
                );
            } else {
                this.doneButton.active = false;
                this.viewMapButton.active = false;
                if (!this.previewGenerating) {
                    this.statusText.setMessage(
                        Component.translatable(
                            "geoworld.create.preview_pending"
                        )
                    );
                }
            }
        } catch (NumberFormatException e) {
            this.doneButton.active = false;
            this.previewButton.active = false;
            this.viewMapButton.active = false;
            this.spanText.setMessage(Component.empty());
            this.statusText.setMessage(
                Component.translatable("geoworld.create.invalid_number")
            );
        }
    }

    private void restoreCachedPreview() {
        if (
            GeoWorldCreationSession
                .generated(this.parent.getUiState())
                .isPresent()
        ) {
            return;
        }

        GeoWorldCreationConfig config;
        try {
            config = this.readConfig();
        } catch (NumberFormatException e) {
            return;
        }
        if (!config.isValid()) {
            return;
        }

        long seed = this.syncSeedFromEditor();
        int serial = this.generationSerial;
        this.previewGenerating = true;
        this.previewButton.active = false;
        this.viewMapButton.active = false;
        this.doneButton.active = false;
        this.statusText.setMessage(
            Component.translatable("geoworld.create.preview_restoring")
        );

        GeoWorldEngineBridge
            .loadCachedAsync(seed, config)
            .whenComplete((cached, error) -> {
                if (this.minecraft == null) {
                    return;
                }
                this.minecraft.execute(() -> {
                    if (serial != this.generationSerial) {
                        return;
                    }
                    this.previewGenerating = false;
                    if (error == null && cached.isPresent()) {
                        GeoWorldCreationSession.setGenerated(
                            this.parent.getUiState(),
                            cached.get()
                        );
                    }
                    this.validateForm();
                });
            });
    }

    private void generatePreview() {
        GeoWorldCreationConfig config;
        try {
            config = this.readConfig();
        } catch (NumberFormatException e) {
            this.validateForm();
            return;
        }
        if (!config.isValid()) {
            this.validateForm();
            return;
        }

        long seed = this.syncSeedFromEditor();
        GeoWorldMod.LOGGER.info(
            "GeoWorld preview generation requested: seed={}, config={}",
            seed,
            config
        );
        GeoWorldCreationSession.set(
            this.parent.getUiState(),
            config
        );
        GeoWorldCreationSession.invalidateGenerated(
            this.parent.getUiState()
        );

        int serial = ++this.generationSerial;
        this.previewGenerating = true;
        this.previewButton.active = false;
        this.viewMapButton.active = false;
        this.doneButton.active = false;
        this.statusText.setMessage(
            Component.translatable("geoworld.create.preview_generating")
        );

        GeoWorldEngineBridge
            .generateAsync(seed, config)
            .whenComplete((generated, error) -> {
                if (this.minecraft == null) {
                    return;
                }
                this.minecraft.execute(() -> {
                    if (serial != this.generationSerial) {
                        return;
                    }
                    this.previewGenerating = false;
                    if (error != null) {
                        this.previewButton.active = true;
                        this.viewMapButton.active = false;
                        this.doneButton.active = false;
                        this.statusText.setMessage(
                            Component.translatable(
                                "geoworld.create.preview_failed",
                                rootMessage(error)
                            )
                        );
                        return;
                    }

                    GeoWorldCreationSession.setGenerated(
                        this.parent.getUiState(),
                        generated
                    );
                    this.validateForm();
                    this.openPreviewMap();
                });
            });
    }

    private long syncSeedFromEditor() {
        this.parent.getUiState().setSeed(this.seedEdit.getValue());
        return this.parent
            .getUiState()
            .getSettings()
            .options()
            .seed();
    }

    private void openPreviewMap() {
        GeoWorldCreationSession
            .generated(this.parent.getUiState())
            .ifPresent(generated ->
                this.minecraft.gui.setScreen(
                    new GeoWorldPreviewScreen(this, generated)
                )
            );
    }

    private static String rootMessage(Throwable error) {
        Throwable current = error;
        while (current.getCause() != null) {
            current = current.getCause();
        }
        String message = current.getMessage();
        return message == null || message.isBlank()
            ? current.getClass().getSimpleName()
            : message;
    }

    private GeoWorldCreationConfig readConfig() {
        double ocean = Double.parseDouble(this.oceanEdit.getValue());
        double fragmentation =
            Double.parseDouble(this.fragmentationEdit.getValue());
        double separation =
            Double.parseDouble(this.separationEdit.getValue());
        ContinentPreset preset = GeoWorldCreationConfig.detectPreset(
            ocean,
            fragmentation,
            separation
        );

        return new GeoWorldCreationConfig(
            Integer.parseInt(this.widthEdit.getValue()),
            Integer.parseInt(this.heightEdit.getValue()),
            Integer.parseInt(this.cellSizeEdit.getValue()),
            Integer.parseInt(this.platesEdit.getValue()),
            ocean,
            fragmentation,
            separation,
            preset
        );
    }

    private void applyAndClose() {
        try {
            GeoWorldCreationConfig config = this.readConfig();
            if (!config.isValid()) {
                this.validateForm();
                return;
            }

            long seed = this.syncSeedFromEditor();

            var generated = GeoWorldCreationSession
                .generated(this.parent.getUiState())
                .filter(value ->
                    value.seed() == seed &&
                    value.config().equals(config)
                );
            if (generated.isEmpty()) {
                this.validateForm();
                return;
            }

            GeoWorldEngineBridge.GeneratedModel ready = generated.get();
            GeoWorldMod.LOGGER.info(
                "GeoWorld create settings accepted: worldSeed={}, modelSeed={}, model={}",
                seed,
                ready.model().manifest().seed(),
                ready.path()
            );
            GeoWorldCreationSession.setGenerated(
                this.parent.getUiState(),
                ready
            );

            this.parent.getUiState().updateDimensions(
                (registryAccess, dimensions) -> {
                    GeoWorldBiomeSource biomeSource =
                        GeoWorldBiomeSource.createDefault(
                            registryAccess.lookupOrThrow(Registries.BIOME)
                        );
                    GeoWorldChunkGenerator generator =
                        new GeoWorldChunkGenerator(biomeSource, config);
                    generator.attachModel(
                        seed,
                        ready.model(),
                        ready.path()
                    );
                    return dimensions.replaceOverworldGenerator(
                        registryAccess,
                        generator
                    );
                }
            );

            this.onClose();
        } catch (NumberFormatException e) {
            this.validateForm();
        }
    }

    @Override
    protected void repositionElements() {
        this.layout.arrangeElements();
    }

    @Override
    public void onClose() {
        this.generationSerial++;
        this.minecraft.gui.setScreen(this.parent);
    }

    private static String formatFraction(double value) {
        return String.format(java.util.Locale.ROOT, "%.2f", value);
    }
}
