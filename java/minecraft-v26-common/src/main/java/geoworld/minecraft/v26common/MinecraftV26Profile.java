package geoworld.minecraft.v26common;

import geoworld.minecraft.common.BiomeFamilyMapping;
import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.SurfaceProfileMapping;
import geoworld.minecraft.common.biome.BiomeCatalog;
import geoworld.minecraft.common.worldgen.HeightMapping;

/** Concrete profile shared by Minecraft 26.x versions. */
public final class MinecraftV26Profile implements MinecraftVersionProfile {
    private final String minecraftVersion;
    private final HeightMapping heights = new MinecraftV26HeightMapping();
    private final SurfaceProfileMapping surfaces =
        new SurfaceProfileMapping() {
            @Override
            public String surfaceBlock(String profile) {
                return MinecraftV26ProfileMapping.surfaceBlock(profile);
            }

            @Override
            public String underBlock(String profile) {
                return MinecraftV26ProfileMapping.underBlock(profile);
            }
        };
    private final BiomeFamilyMapping biomes =
        MinecraftV26ProfileMapping::biomeId;
    private final BiomeCatalog biomeCatalog;
    private final boolean supportsSulfurCaves;

    public MinecraftV26Profile(
        String minecraftVersion,
        boolean supportsSulfurCaves,
        boolean supportsPaleGarden,
        boolean supportsDappledForest
    ) {
        this.minecraftVersion = java.util.Objects.requireNonNull(
            minecraftVersion,
            "minecraftVersion"
        );
        this.supportsSulfurCaves = supportsSulfurCaves;
        this.biomeCatalog = new MinecraftV26BiomeCatalog(
            supportsPaleGarden,
            supportsDappledForest
        );
    }

    public boolean supportsSulfurCaves() {
        return supportsSulfurCaves;
    }

    @Override
    public String minecraftVersion() {
        return minecraftVersion;
    }

    @Override
    public HeightMapping heightMapping() {
        return heights;
    }

    @Override
    public SurfaceProfileMapping surfaceProfileMapping() {
        return surfaces;
    }

    @Override
    public BiomeFamilyMapping biomeFamilyMapping() {
        return biomes;
    }

    @Override
    public BiomeCatalog biomeCatalog() {
        return biomeCatalog;
    }
}
