package geoworld.minecraft.client;

import com.mojang.logging.LogUtils;
import org.slf4j.Logger;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.Parameters;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.WorldPreset;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.AbstractSliderButton;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.components.CycleButton;
import net.minecraft.client.gui.components.EditBox;
import net.minecraft.client.gui.components.StringWidget;
import net.minecraft.client.gui.components.Tooltip;
import net.minecraft.client.gui.layouts.CommonLayouts;
import net.minecraft.client.gui.layouts.HeaderAndFooterLayout;
import net.minecraft.client.gui.layouts.LayoutElement;
import net.minecraft.client.gui.layouts.LinearLayout;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.client.gui.screens.worldselection.CreateWorldScreen;
import net.minecraft.client.gui.screens.worldselection.WorldCreationContext;
import net.minecraft.core.HolderGetter;
import net.minecraft.core.registries.Registries;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.minecraft.network.chat.CommonComponents;
import net.minecraft.network.chat.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ThreadLocalRandom;
import java.util.function.BiConsumer;
import java.util.function.Function;

public class GeoWorldPresetScreenBase extends Screen {
    public interface WorldgenAdapter {
        GeoWorldCreationConfig configOf(ChunkGenerator generator);

        ChunkGenerator create(
            HolderGetter<Biome> biomes,
            GeoWorldCreationConfig config,
            long seed,
            GeoWorldEngineBridge.GeneratedModel generated
        );
    }
    public interface UiAdapter {
        <T> CycleButton<T> createCycleButton(
            Function<T, Component> valueStringifier,
            T initialValue,
            List<T> values,
            int width,
            Component message,
            BiConsumer<CycleButton<T>, T> onValueChange
        );

        int parameterPageSize();

        LayoutElement createContent(
            Minecraft minecraft,
            LinearLayout body,
            int contentHeight,
            int minWidth
        );
    }

    @FunctionalInterface
    public interface ScreenNavigator {
        void setScreen(Screen screen);
    }
    @FunctionalInterface
    public interface PreviewFactory {
        Screen create(
            Screen parent,
            GeoWorldEngineBridge.GeneratedModel generated
        );
    }
    private static final Logger LOGGER = LogUtils.getLogger();
    private static final Component TITLE =
        Component.translatable("geoworld.create.title");
    private static final Component SEED_LABEL =
        Component.translatable("geoworld.create.seed");
    private static final Component VANILLA_SEED_LABEL =
        Component.translatable("selectWorld.enterSeed");
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
    private static final Component WORLD_PRESET_LABEL =
        Component.translatable("geoworld.create.world_preset");
    private static final Component PRESET_LABEL =
        Component.translatable("geoworld.create.continent_preset");
    private static final Component SURFACE_ORE_PILE_LABEL =
        Component.translatable("geoworld.create.surface_ore_pile");
    private static final List<String> PARAMETER_IDS = List.of(
        "mainland_scale", "region_scale", "transition_width",
        "mountain_strength", "plain_continuity", "plateau_scale",
        "cliff_rarity", "rare_macro_terrain",
        "river_density", "main_river_scale", "lake_abundance", "wetland_abundance",
        "climate_region_scale", "climate_transition_width",
        "rain_shadow_strength", "biome_continuity",
        "cave_richness", "large_cave_rarity", "mineral_regionalization",
        "spawn_preference", "structure_influence",
        "outer_island_abundance", "polar_transition_width",
        "ocean_climate_strength", "prevailing_wind_strength"
    );

    private final CreateWorldScreen parent;
    private final WorldgenAdapter worldgenAdapter;
    private final PreviewFactory previewFactory;
    private final UiAdapter uiAdapter;
    private final ScreenNavigator screenNavigator;
    private HeaderAndFooterLayout layout;

    private GeoWorldCreationConfig initialConfig;
    private ContinentPreset selectedPreset;
    private WorldPreset selectedWorldPreset;
    private Parameters selectedParameters;
    private int parameterPage;
    private boolean applyingPreset;

    private EditBox seedEdit;
    private EditBox widthEdit;
    private EditBox heightEdit;
    private EditBox cellSizeEdit;
    private EditBox platesEdit;
    private FractionSlider oceanSlider;
    private FractionSlider fragmentationSlider;
    private FractionSlider separationSlider;
    private CycleButton<WorldPreset> worldPresetButton;
    private CycleButton<ContinentPreset> presetButton;
    private CycleButton<Boolean> surfaceOrePileButton;
    private CycleButton<Integer> parameterPageButton;
    private final List<FractionSlider> parameterSliders = new ArrayList<>();
    private StringWidget statusText;
    private StringWidget spanText;
    private Button previewButton;
    private Button viewMapButton;
    private Button doneButton;
    private boolean previewGenerating;
    private int generationSerial;

    public GeoWorldPresetScreenBase(
        CreateWorldScreen parent,
        WorldCreationContext settings,
        WorldgenAdapter worldgenAdapter,
        PreviewFactory previewFactory,
        UiAdapter uiAdapter,
        ScreenNavigator screenNavigator
    ) {
        super(TITLE);
        this.parent = parent;
        this.worldgenAdapter = java.util.Objects.requireNonNull(
            worldgenAdapter,
            "worldgenAdapter"
        );
        this.previewFactory = previewFactory;
        this.uiAdapter = java.util.Objects.requireNonNull(
            uiAdapter,
            "uiAdapter"
        );
        this.screenNavigator = java.util.Objects.requireNonNull(
            screenNavigator,
            "screenNavigator"
        );
        GeoWorldCreationConfig currentConfig = this.worldgenAdapter.configOf(
            settings.selectedDimensions().overworld()
        );
        this.initialConfig = currentConfig != null
            ? currentConfig
            : GeoWorldCreationSession.get(parent.getUiState());
        this.selectedPreset = this.initialConfig.continentPreset();
        this.selectedWorldPreset = this.initialConfig.worldPreset();
        this.selectedParameters = this.initialConfig.parameters();
    }

    /**
     * Version adapters may publish registry-derived generation context here
     * before cache lookup or model generation. The shared UI stays version-
     * neutral; concrete adapters own Minecraft registry/tag access.
     */
    protected void prepareGenerationContext(WorldCreationContext settings) {}

    @Override
    protected void init() {
        this.layout = new HeaderAndFooterLayout(this, 33, 58);
        this.layout.addTitleHeader(this.title, this.font);

        LinearLayout body = LinearLayout.vertical().spacing(4);
        body.defaultCellSetting().alignHorizontallyCenter();

        String seedValue = this.parent.getUiState().getSeed();
        if (seedValue == null || seedValue.isBlank()) {
            seedValue = Long.toString(ThreadLocalRandom.current().nextLong());
            this.setSharedSeed(seedValue);
            GeoWorldCreationSession.invalidateGenerated(
                this.parent.getUiState()
            );
        }
        this.seedEdit = numberBox(
            230,
            SEED_LABEL,
            seedValue
        );
        this.seedEdit.setResponder(value -> {
            this.setSharedSeed(value);
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

        this.worldPresetButton = this.uiAdapter.createCycleButton(
            preset -> Component.translatable(preset.translationKey()),
            this.selectedWorldPreset,
            List.of(WorldPreset.values()),
            310,
            WORLD_PRESET_LABEL,
            (button, preset) -> this.applyWorldPreset(preset)
        );
        body.addChild(this.worldPresetButton);

        this.presetButton = this.uiAdapter.createCycleButton(
            preset -> Component.translatable(preset.translationKey()),
            this.selectedPreset,
            List.of(ContinentPreset.values()),
            310,
            PRESET_LABEL,
            (button, preset) -> this.applyPreset(preset)
        );
        body.addChild(this.presetButton);

        this.surfaceOrePileButton = this.uiAdapter.createCycleButton(
            value -> Component.translatable(
                value
                    ? "geoworld.value.enabled"
                    : "geoworld.value.disabled"
            ),
            this.initialConfig.surfaceOrePile(),
            List.of(false, true),
            310,
            SURFACE_ORE_PILE_LABEL,
            (button, value) -> this.onConfigEdited()
        );
        this.surfaceOrePileButton.setTooltip(
            Tooltip.create(
                Component.translatable(
                    "geoworld.create.surface_ore_pile.desc"
                )
            )
        );
        body.addChild(this.surfaceOrePileButton);

        this.widthEdit = numberBox(
            152,
            WIDTH_LABEL,
            Integer.toString(this.initialConfig.width())
        );
        this.heightEdit = numberBox(
            152,
            HEIGHT_LABEL,
            Integer.toString(this.initialConfig.height())
        );
        LinearLayout sizeRow = LinearLayout.horizontal().spacing(6);
        sizeRow.addChild(
            CommonLayouts.labeledElement(
                this.font,
                this.widthEdit,
                WIDTH_LABEL
            )
        );
        sizeRow.addChild(
            CommonLayouts.labeledElement(
                this.font,
                this.heightEdit,
                HEIGHT_LABEL
            )
        );
        body.addChild(sizeRow);

        this.cellSizeEdit = numberBox(
            152,
            CELL_SIZE_LABEL,
            Integer.toString(this.initialConfig.cellSizeBlocks())
        );
        this.platesEdit = numberBox(
            152,
            PLATES_LABEL,
            Integer.toString(this.initialConfig.plateCount())
        );
        LinearLayout samplingRow = LinearLayout.horizontal().spacing(6);
        samplingRow.addChild(
            CommonLayouts.labeledElement(
                this.font,
                this.cellSizeEdit,
                CELL_SIZE_LABEL
            )
        );
        samplingRow.addChild(
            CommonLayouts.labeledElement(
                this.font,
                this.platesEdit,
                PLATES_LABEL
            )
        );
        body.addChild(samplingRow);
        this.oceanSlider = addFractionSlider(
            body,
            OCEAN_LABEL,
            this.initialConfig.oceanFraction(),
            0.10,
            0.95
        );
        this.fragmentationSlider = addFractionSlider(
            body,
            FRAGMENTATION_LABEL,
            this.initialConfig.continentFragmentation(),
            0.05,
            1.00
        );
        this.separationSlider = addFractionSlider(
            body,
            SEPARATION_LABEL,
            this.initialConfig.continentSeparation(),
            0.05,
            1.00
        );

        this.initParameterControls(body);

        this.widthEdit.setResponder(value -> this.onDirectConfigEdited());
        this.heightEdit.setResponder(value -> this.onDirectConfigEdited());
        this.cellSizeEdit.setResponder(value -> this.onDirectConfigEdited());
        this.platesEdit.setResponder(value -> this.onDirectConfigEdited());

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
        );        this.layout.addToContents(
            this.uiAdapter.createContent(
                this.minecraft,
                body,
                this.layout.getContentHeight(),
                310
            )
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

    private void initParameterControls(LinearLayout body) {
        this.parameterSliders.clear();
        double[] values = this.selectedParameters.values();
        int pageSize = this.parameterPageSize();

        if (pageSize <= 0) {
            body.addChild(new StringWidget(
                Component.translatable("geoworld.create.parameters"),
                this.font
            ));
            for (int i = 0; i < PARAMETER_IDS.size(); i++) {
                String parameterId = PARAMETER_IDS.get(i);
                FractionSlider slider = addFractionSlider(
                    body,
                    Component.translatable(
                        "geoworld.parameter." + parameterId
                    ),
                    values[i],
                    parameterMinimum(parameterId),
                    2.50,
                    this::onParameterEdited
                );
                slider.setTooltip(
                    Tooltip.create(
                        Component.translatable(
                            "geoworld.parameter." +
                            parameterId +
                            ".desc"
                        )
                    )
                );
                this.parameterSliders.add(slider);
            }
            return;
        }

        int pageCount = this.parameterPageCount();
        List<Integer> pages = new ArrayList<>();
        for (int page = 0; page < pageCount; page++) {
            pages.add(page);
        }
        this.parameterPage = Math.max(
            0,
            Math.min(pageCount - 1, this.parameterPage)
        );
        body.addChild(new StringWidget(
            Component.translatable("geoworld.create.parameters"),
            this.font
        ));
        this.parameterPageButton = this.uiAdapter.createCycleButton(
            page -> Component.literal(
                (page + 1) + "/" + pageCount
            ),
            this.parameterPage,
            pages,
            310,
            Component.translatable("geoworld.create.parameter_page_label"),
            (button, page) -> this.switchParameterPage(page)
        );
        body.addChild(this.parameterPageButton);

        for (int slot = 0; slot < pageSize; slot++) {
            int parameterIndex = this.parameterPage * pageSize + slot;
            boolean valid = parameterIndex < PARAMETER_IDS.size();
            String parameterId = valid
                ? PARAMETER_IDS.get(parameterIndex)
                : null;
            FractionSlider slider = addFractionSlider(
                body,
                valid
                    ? Component.translatable(
                        "geoworld.parameter." + parameterId
                    )
                    : Component.literal("-"),
                valid ? values[parameterIndex] : 1.0,
                valid ? parameterMinimum(parameterId) : 0.25,
                2.50,
                this::onParameterEdited
            );
            slider.active = valid;
            slider.setTooltip(
                Tooltip.create(
                    valid
                        ? Component.translatable(
                            "geoworld.parameter." +
                            parameterId +
                            ".desc"
                        )
                        : Component.empty()
                )
            );
            this.parameterSliders.add(slider);
        }
    }

    private int parameterPageSize() {
        return Math.max(0, this.uiAdapter.parameterPageSize());
    }

    private int parameterPageCount() {
        int pageSize = this.parameterPageSize();
        return pageSize <= 0
            ? 1
            : Math.max(
                1,
                (PARAMETER_IDS.size() + pageSize - 1) / pageSize
            );
    }
    private FractionSlider addFractionSlider(
        LinearLayout body,
        Component label,
        double value,
        double min,
        double max
    ) {
        return addFractionSlider(
            body,
            label,
            value,
            min,
            max,
            this::onContinentSliderEdited
        );
    }

    private FractionSlider addFractionSlider(
        LinearLayout body,
        Component label,
        double value,
        double min,
        double max,
        Runnable onChanged
    ) {
        FractionSlider slider = new FractionSlider(
            310,
            label,
            value,
            min,
            max,
            onChanged
        );
        body.addChild(slider);
        return slider;
    }

    private void applyWorldPreset(WorldPreset preset) {
        this.selectedWorldPreset = preset;
        if (preset == WorldPreset.CUSTOM) {
            this.onConfigEdited();
            return;
        }
        this.selectedParameters = Parameters.forPreset(preset);
        this.applyingPreset = true;
        try {
            GeoWorldCreationConfig next = this.initialConfig
                .withWorldPreset(preset);
            this.widthEdit.setValue(Integer.toString(next.width()));
            this.heightEdit.setValue(Integer.toString(next.height()));
            this.refreshParameterPage();
            this.worldPresetButton.setValue(preset);
        } finally {
            this.applyingPreset = false;
        }
        this.onConfigEdited();
    }

    private void switchParameterPage(int page) {
        if (this.parameterPageSize() <= 0) {
            return;
        }
        this.storeVisibleParameterValues();
        this.parameterPage = Math.max(
            0,
            Math.min(this.parameterPageCount() - 1, page)
        );
        this.refreshParameterPage();
    }

    private void refreshParameterPage() {
        int pageSize = this.parameterPageSize();
        if (
            pageSize <= 0 ||
            this.parameterSliders.size() != pageSize
        ) {
            return;
        }
        double[] values = this.selectedParameters.values();
        for (int slot = 0; slot < pageSize; slot++) {
            int parameterIndex = this.parameterPage * pageSize + slot;
            FractionSlider slider = this.parameterSliders.get(slot);
            if (parameterIndex >= PARAMETER_IDS.size()) {
                slider.setDefinition(
                    Component.literal("-"),
                    0.25,
                    2.50,
                    1.0
                );
                slider.setTooltip(Tooltip.create(Component.empty()));
                slider.active = false;
                continue;
            }
            String parameterId = PARAMETER_IDS.get(parameterIndex);
            slider.active = true;
            slider.setDefinition(
                Component.translatable(
                    "geoworld.parameter." + parameterId
                ),
                parameterMinimum(parameterId),
                2.50,
                values[parameterIndex]
            );
            slider.setTooltip(
                Tooltip.create(
                    Component.translatable(
                        "geoworld.parameter." +
                        parameterId +
                        ".desc"
                    )
                )
            );
        }
    }

    private void storeVisibleParameterValues() {
        int pageSize = this.parameterPageSize();
        if (pageSize <= 0) {
            if (this.parameterSliders.size() != PARAMETER_IDS.size()) {
                return;
            }
            double[] values =
                new double[this.parameterSliders.size()];
            for (int i = 0; i < values.length; i++) {
                values[i] =
                    this.parameterSliders.get(i).actualValue();
            }
            this.selectedParameters = Parameters.fromValues(values);
            return;
        }

        if (this.parameterSliders.size() != pageSize) {
            return;
        }
        double[] values = this.selectedParameters.values();
        for (int slot = 0; slot < pageSize; slot++) {
            int parameterIndex = this.parameterPage * pageSize + slot;
            if (parameterIndex >= PARAMETER_IDS.size()) {
                continue;
            }
            values[parameterIndex] =
                this.parameterSliders.get(slot).actualValue();
        }
        this.selectedParameters = Parameters.fromValues(values);
    }

    private void onParameterEdited() {
        if (this.applyingPreset) return;
        this.storeVisibleParameterValues();
        this.selectedWorldPreset = WorldPreset.CUSTOM;
        this.worldPresetButton.setValue(WorldPreset.CUSTOM);
        this.onConfigEdited();
    }

    private static double parameterMinimum(String parameterId) {
        return parameterId.equals("spawn_preference") ||
            parameterId.equals("structure_influence")
            ? 0.0
            : 0.25;
    }

    private void applyPreset(ContinentPreset preset) {
        this.selectedPreset = preset;
        if (preset == ContinentPreset.CUSTOM) {
            this.onConfigEdited();
            return;
        }

        this.applyingPreset = true;
        try {
            this.oceanSlider.setActualValue(preset.oceanFraction());
            this.fragmentationSlider.setActualValue(preset.fragmentation());
            this.separationSlider.setActualValue(preset.separation());
        } finally {
            this.applyingPreset = false;
        }
        this.onConfigEdited();
    }

    private void onContinentSliderEdited() {
        if (this.applyingPreset) {
            return;
        }
        this.selectedPreset = GeoWorldCreationConfig.detectPreset(
            this.oceanSlider.actualValue(),
            this.fragmentationSlider.actualValue(),
            this.separationSlider.actualValue()
        );
        this.presetButton.setValue(this.selectedPreset);
        this.onConfigEdited();
    }

    private void onDirectConfigEdited() {
        if (!this.applyingPreset) {
            this.onConfigEdited();
        }
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
                this.setViewMapActive(false);
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
            // Screen#setScreen re-initializes this same screen after closing the
            // map preview. Keep the latest accepted editor config as the source
            // for the next init() so returning from preview cannot silently
            // restore constructor-time defaults and invalidate the ready model.
            this.initialConfig = config;
            this.selectedPreset = config.continentPreset();
            this.selectedWorldPreset = config.worldPreset();
            this.selectedParameters = config.parameters();
            this.previewButton.active = !this.previewGenerating;
            List<String> warningKeys = config.validationWarningKeys();
            this.spanText.setMessage(
                warningKeys.isEmpty()
                    ? Component.translatable(
                        "geoworld.create.span",
                        config.spanXBlocks(),
                        config.spanZBlocks()
                    )
                    : Component.translatable(
                        "geoworld.create.span_warning",
                        config.spanXBlocks(),
                        config.spanZBlocks(),
                        Component.translatable(warningKeys.get(0))
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
                this.setViewMapActive(true);
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
                this.setViewMapActive(false);
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
            this.setViewMapActive(false);
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
        this.setViewMapActive(false);
        this.doneButton.active = false;
        this.statusText.setMessage(
            Component.translatable("geoworld.create.preview_restoring")
        );

        this.prepareGenerationContext(
            this.parent.getUiState().getSettings()
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
        LOGGER.info(
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
        this.setViewMapActive(false);
        this.doneButton.active = false;
        this.statusText.setMessage(
            Component.translatable("geoworld.create.preview_generating")
        );

        this.prepareGenerationContext(
            this.parent.getUiState().getSettings()
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
                        this.setViewMapActive(false);
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

    private void setSharedSeed(String value) {
        this.parent.getUiState().setSeed(value);

        // Vanilla's World tab only copies UiState -> seed EditBox once in the
        // tab constructor. It does not subscribe that EditBox to later UiState
        // changes, so changing the seed inside a preset editor leaves the outer
        // field visually stale even though the actual world options are already
        // updated. Keep that existing widget synchronized while this editor is
        // open.
        for (var listener : this.parent.children()) {
            if (
                listener instanceof EditBox box &&
                box.getWidth() == 308 &&
                box.getMessage().getString().equals(
                    VANILLA_SEED_LABEL.getString()
                ) &&
                !box.getValue().equals(value)
            ) {
                box.setValue(value);
                break;
            }
        }
    }

    private long syncSeedFromEditor() {
        this.setSharedSeed(this.seedEdit.getValue());
        return this.parent
            .getUiState()
            .getSettings()
            .options()
            .seed();
    }

    private void setViewMapActive(boolean active) {
        if (this.viewMapButton != null) {
            this.viewMapButton.active = active;
        }
    }

    private void openPreviewMap() {
        if (this.previewFactory == null) {
            return;
        }
        GeoWorldCreationSession
            .generated(this.parent.getUiState())
            .ifPresent(generated ->
                this.screenNavigator.setScreen(
                    this.previewFactory.create(this, generated)
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
        this.storeVisibleParameterValues();
        double ocean = this.oceanSlider.actualValue();
        double fragmentation = this.fragmentationSlider.actualValue();
        double separation = this.separationSlider.actualValue();
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
            preset,
            this.selectedWorldPreset,
            this.selectedParameters,
            this.surfaceOrePileButton.getValue()
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
            LOGGER.info(
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
                    ChunkGenerator generator =
                        this.worldgenAdapter.create(
                            registryAccess.lookupOrThrow(Registries.BIOME),
                            config,
                            seed,
                            ready
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
        this.screenNavigator.setScreen(this.parent);
    }

    private static String formatFraction(double value) {
        return String.format(java.util.Locale.ROOT, "%.2f", value);
    }

    private static final class FractionSlider extends AbstractSliderButton {
        private static final double STEP = 0.01;

        private Component label;
        private double min;
        private double max;
        private final Runnable onChanged;

        private FractionSlider(
            int width,
            Component label,
            double initialValue,
            double min,
            double max,
            Runnable onChanged
        ) {
            super(0, 0, width, 20, Component.empty(), 0.0);
            this.label = label;
            this.min = min;
            this.max = max;
            this.onChanged = onChanged;
            this.setActualValue(initialValue);
        }

        private double actualValue() {
            return snap(this.min + this.value * (this.max - this.min));
        }

        private void setDefinition(
            Component label,
            double min,
            double max,
            double actualValue
        ) {
            this.label = label;
            this.min = min;
            this.max = max;
            this.setActualValue(actualValue);
        }

        private void setActualValue(double actualValue) {
            double snapped = snap(actualValue);
            this.value = Math.max(
                0.0,
                Math.min(1.0, (snapped - this.min) / (this.max - this.min))
            );
            this.updateMessage();
        }

        @Override
        protected void updateMessage() {
            this.setMessage(
                Component.empty()
                    .append(this.label)
                    .append(Component.literal(": " + formatFraction(this.actualValue())))
            );
        }

        @Override
        protected void applyValue() {
            double snapped = this.actualValue();
            this.value = Math.max(
                0.0,
                Math.min(1.0, (snapped - this.min) / (this.max - this.min))
            );
            this.onChanged.run();
        }

        private double snap(double actualValue) {
            double clamped = Math.max(this.min, Math.min(this.max, actualValue));
            double snapped = Math.round(clamped / STEP) * STEP;
            return Math.max(this.min, Math.min(this.max, snapped));
        }
    }
}
