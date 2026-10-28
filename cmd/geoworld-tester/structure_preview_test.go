package main

import (
	"testing"

	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
)

func TestStructureSnapshotParity(t *testing.T) {
	snapshot, err := loadStructureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.MinecraftVersion != "26.3" {
		t.Fatalf("minecraft version = %q", snapshot.MinecraftVersion)
	}
	if len(snapshot.StructureSets) != 21 {
		t.Fatalf("structure sets = %d, want 21", len(snapshot.StructureSets))
	}
}

func TestJavaRandomKnownSequence(t *testing.T) {
	r := newJavaRandom(123456789)
	if got := r.nextInt(6); got != 1 {
		t.Fatalf("nextInt(6)=%d want 1", got)
	}
	if got := r.nextDouble(); got != 0.7633164593123165 {
		t.Fatalf("nextDouble=%0.17g", got)
	}
}

func TestOutpostExclusionUsesVillagePlacement(t *testing.T) {
	snapshot, err := loadStructureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	outpost := snapshot.StructureSets["minecraft:pillager_outposts"]
	if outpost.Placement.ExclusionZone == nil {
		t.Fatal("pillager outpost exclusion missing")
	}
	if outpost.Placement.ExclusionZone.OtherSet != "minecraft:villages" ||
		outpost.Placement.ExclusionZone.ChunkCount != 10 {
		t.Fatalf("unexpected exclusion: %+v", outpost.Placement.ExclusionZone)
	}
}

func TestStructureGeoCompatibilityHighConfidenceRules(t *testing.T) {
	sample := func(
		water reconcile.WaterKind,
		family mcprofile.BiomeFamily,
	) runtimeprofile.Sample {
		var got runtimeprofile.Sample
		got.World.WaterKind = water
		got.BiomeFamily = family
		return got
	}

	tests := []struct {
		setID  string
		sample runtimeprofile.Sample
		want   bool
	}{
		{"minecraft:ocean_monuments", sample(reconcile.WaterOcean, mcprofile.BiomeTemperateOcean), true},
		{"minecraft:ocean_monuments", sample(reconcile.WaterNone, mcprofile.BiomeTemperateForest), false},
		{"minecraft:desert_pyramids", sample(reconcile.WaterNone, mcprofile.BiomeHotDesert), true},
		{"minecraft:desert_pyramids", sample(reconcile.WaterNone, mcprofile.BiomeTemperateForest), false},
		{"minecraft:igloos", sample(reconcile.WaterNone, mcprofile.BiomeTundra), true},
		{"minecraft:igloos", sample(reconcile.WaterNone, mcprofile.BiomeTropicalRainforest), false},
		{"minecraft:jungle_temples", sample(reconcile.WaterNone, mcprofile.BiomeTropicalRainforest), true},
		{"minecraft:jungle_temples", sample(reconcile.WaterNone, mcprofile.BiomeTemperateForest), false},
		{"minecraft:swamp_huts", sample(reconcile.WaterNone, mcprofile.BiomeWetland), true},
		{"minecraft:swamp_huts", sample(reconcile.WaterNone, mcprofile.BiomeSavanna), false},
		{"minecraft:woodland_mansions", sample(reconcile.WaterNone, mcprofile.BiomeTemperateForest), true},
		{"minecraft:woodland_mansions", sample(reconcile.WaterNone, mcprofile.BiomeHotDesert), false},
		{"minecraft:villages", sample(reconcile.WaterNone, mcprofile.BiomeSavanna), true},
		{"minecraft:villages", sample(reconcile.WaterOcean, mcprofile.BiomeSavanna), false},

		// Complex underground/coast structures stay candidates unless we can
		// reject them confidently from the surface model.
		{"minecraft:mineshafts", sample(reconcile.WaterOcean, mcprofile.BiomeWarmOcean), true},
		{"minecraft:trial_chambers", sample(reconcile.WaterOcean, mcprofile.BiomeWarmOcean), true},
	}
	for _, tt := range tests {
		if got := structureGeoCompatible(tt.setID, tt.sample); got != tt.want {
			t.Fatalf("%s compatibility=%v want=%v sample=%+v", tt.setID, got, tt.want, tt.sample)
		}
	}
}

func TestStructureGeoAssessmentSeparatesPassRejectUnknown(t *testing.T) {
	sample := func(
		water reconcile.WaterKind,
		family mcprofile.BiomeFamily,
	) runtimeprofile.Sample {
		var got runtimeprofile.Sample
		got.World.WaterKind = water
		got.BiomeFamily = family
		return got
	}

	tests := []struct {
		name   string
		setID  string
		sample runtimeprofile.Sample
		want   string
	}{
		{
			name:   "pass",
			setID:  "minecraft:villages",
			sample: sample(reconcile.WaterNone, mcprofile.BiomeSavanna),
			want:   "pass",
		},
		{
			name:   "reject",
			setID:  "minecraft:villages",
			sample: sample(reconcile.WaterOcean, mcprofile.BiomeSavanna),
			want:   "reject",
		},
		{
			name:   "unknown",
			setID:  "minecraft:mineshafts",
			sample: sample(reconcile.WaterOcean, mcprofile.BiomeWarmOcean),
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason := structureGeoAssessment(tt.setID, tt.sample)
			if status != tt.want {
				t.Fatalf("status=%q want=%q", status, tt.want)
			}
			if reason == "" {
				t.Fatal("assessment reason must not be empty")
			}
		})
	}
}

func TestFindNearestVillageCandidate(t *testing.T) {
	snapshot, err := loadStructureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	candidate, distance, ok := findNearestRandomSpreadCandidate(
		123456, snapshot, "minecraft:villages",
		0, 0, -12000, 12000, -12000, 12000,
		nil,
	)
	if !ok {
		t.Fatal("nearest village candidate not found")
	}
	if candidate.SetID != "minecraft:villages" {
		t.Fatalf("set = %q", candidate.SetID)
	}
	if distance < 0 {
		t.Fatalf("distance = %f", distance)
	}
	placement := snapshot.StructureSets["minecraft:villages"].Placement
	if !isRandomSpreadPlacementChunk(
		123456, candidate.ChunkX, candidate.ChunkZ, placement, true, snapshot,
	) {
		t.Fatalf("returned chunk is not a valid village placement: %+v", candidate)
	}
}
