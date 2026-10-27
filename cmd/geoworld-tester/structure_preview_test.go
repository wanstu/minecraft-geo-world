package main

import "testing"

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
