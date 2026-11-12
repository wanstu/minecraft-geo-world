package worldsample

import (
	"testing"

	"geoworld/internal/waterbed"
)

func TestR81gCoastalRiverTransitionVersioned(t *testing.T) {
	// The R8.1f blocker had two connected river columns, Y=66 beside Y=63,
	// solely because the sea-backwater path switched at the sea-level bed.
	const sea = 0.50
	lowRiverBed := waterbed.Sample{
		FinalHeight:                 sea - .00010,
		RiverWeight:                 0.9,
		RiverCoastalReceiverDefined: true,
		RiverCoastalReceiverSurface: sea,
	}
	highRiverBed := lowRiverBed
	highRiverBed.FinalHeight = sea + .00020
	nominalRiver := sea + .0061 // three Minecraft blocks above sea
	old := Config{CoastalBackwaterVersion: 1}
	next := Config{CoastalBackwaterVersion: 2}
	if !canApplyCoastalBackwater(old, lowRiverBed, sea, 0, nominalRiver, true) {
		t.Fatal("v1.1.25 coast must retain its previous deterministic override")
	}
	if canApplyCoastalBackwater(next, lowRiverBed, sea, 0, nominalRiver, true) {
		t.Fatal("v2 should not drop a supported water surface >1 block to sea level")
	}
	if canApplyCoastalBackwater(next, highRiverBed, sea, 0, nominalRiver, true) {
		t.Fatal("above-sea riverbed must not cause coastal flooding")
	}
	if !canApplyCoastalBackwater(next, lowRiverBed, sea, 0, sea+.0005, true) {
		t.Fatal("v2 should still join the sea once the river water datum is near sea level")
	}
	if !canApplyCoastalBackwater(next, lowRiverBed, sea, 0, nominalRiver, false) {
		t.Fatal("v2 keeps disconnected-from-channel but receiver-owned low-bank sea inundation")
	}
	unrelated := lowRiverBed
	unrelated.RiverCoastalReceiverDefined = false
	if canApplyCoastalBackwater(next, unrelated, sea, 0, sea, true) {
		t.Fatal("unowned inland depression must never become coastal water")
	}
	legacy := Config{}
	if canApplyCoastalBackwater(legacy, lowRiverBed, sea, 0, nominalRiver, true) {
		t.Fatal("legacy rules must not gain backwater semantics")
	}
}
