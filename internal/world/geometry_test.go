package world

import (
	"math"
	"testing"
)

func TestChangingResolutionPreservesWorldExtent(t *testing.T) {
	low := GridGeometry{Width: 5, Height: 3, OriginX: -512, OriginZ: -256, SpanXBlocks: 1024, SpanZBlocks: 512}
	high := low
	high.Width = 9
	high.Height = 5
	for _, g := range []GridGeometry{low, high} {
		if err := g.Validate(); err != nil {
			t.Fatal(err)
		}
		x, z := g.NodePosition(g.Width-1, g.Height-1)
		if x != 512 || z != 256 {
			t.Fatalf("extent changed: %v,%v", x, z)
		}
		x, z = g.NodePosition((g.Width-1)/2, (g.Height-1)/2)
		if x != 0 || z != 0 {
			t.Fatalf("origin convention changed: %v,%v", x, z)
		}
	}
	if high.SpacingX() != low.SpacingX()/2 || high.SpacingZ() != low.SpacingZ()/2 {
		t.Fatal("resolution did not change spacing")
	}
}

func TestInvalidGeometryAndLegacyAnisotropyAreRejected(t *testing.T) {
	good := GridGeometry{Width: 3, Height: 3, SpanXBlocks: 256, SpanZBlocks: 256}
	maxInt := int(^uint(0) >> 1)
	cases := []GridGeometry{
		{Width: 1, Height: 3, SpanXBlocks: 256, SpanZBlocks: 256},
		{Width: maxInt, Height: 2, SpanXBlocks: 256, SpanZBlocks: 256},
		{Width: 3, Height: 3, SpanXBlocks: 0, SpanZBlocks: 256},
		{Width: 3, Height: 3, OriginX: math.NaN(), SpanXBlocks: 256, SpanZBlocks: 256},
		{Width: 3, Height: 3, SpanXBlocks: math.Inf(1), SpanZBlocks: 256},
		{Width: 3, Height: 3, OriginX: 1e30, SpanXBlocks: 256, SpanZBlocks: 256},
	}
	for i, g := range cases {
		if g.Validate() == nil {
			t.Fatalf("invalid geometry accepted: %d", i)
		}
	}
	good.SpanZBlocks = 512
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := good.SquareCellSize(); err == nil {
		t.Fatal("anisotropic geometry silently stretched")
	}
}
