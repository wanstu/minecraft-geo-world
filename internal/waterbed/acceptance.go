package waterbed

import (
	"math"

	"geoworld/internal/reconcile"
)

// FluvialAcceptanceCandidate is a read-only diagnostic location derived from
// the same v1.1.7 fluvial-detail functions used by Runtime sampling. It exists
// so acceptance tooling can locate implemented later-stage river features
// without duplicating generation rules.
type FluvialAcceptanceCandidate struct {
	Kind     string
	X        float64
	Z        float64
	Strength float64
}

// AcceptanceFluvialCandidates returns bounded deterministic candidate points
// for implemented v1.1.7 floodplain/terrace/levee/oxbow features.
func (m *Model) AcceptanceFluvialCandidates() []FluvialAcceptanceCandidate {
	if m == nil || m.Base == nil {
		return nil
	}
	const perKindLimit = 4
	counts := map[string]int{}
	out := make([]FluvialAcceptanceCandidate, 0, 16)
	add := func(kind string, x, z, strength float64) {
		if counts[kind] >= perKindLimit {
			return
		}
		for _, existing := range out {
			if existing.Kind == kind && math.Hypot(existing.X-x, existing.Z-z) < 96 {
				return
			}
		}
		out = append(out, FluvialAcceptanceCandidate{
			Kind: kind, X: x, Z: z, Strength: strength,
		})
		counts[kind]++
	}

	for _, seg := range m.Rivers {
		if seg.MouthExpansionVersion < 5 {
			continue
		}
		strength := riverFluvialDetailStrength(seg)
		if strength <= 0 {
			continue
		}
		key := riverFluvialKey(seg)

		for _, t := range []float64{0.22, 0.34, 0.46, 0.58, 0.70, 0.82} {
			cx, cz, nx, nz := RiverCenterFrame(seg, t)
			bankRadius := riverMouthHalfWidth(seg, t) +
				riverBankBlendBlocks(seg.HalfWidth)
			reach := riverFluvialReach(seg)
			if reach <= bankRadius+1 {
				continue
			}

			for _, side := range []float64{-1, 1} {
				for _, feature := range []struct {
					kind string
					n    float64
				}{
					{kind: "floodplain", n: 0.20},
					{kind: "terrace", n: 0.62},
				} {
					if counts[feature.kind] >= perKindLimit {
						continue
					}
					distance := bankRadius +
						feature.n*(reach-bankRadius)
					x := cx + nx*side*distance
					z := cz + nz*side*distance
					bucket := m.bucketIndexForWorld(x, z)
					if bucket < 0 ||
						riverBankProtectedAt(m.Rivers, m.riverBuckets[bucket], x, z) {
						continue
					}
					baseHeight := m.Base.SampleNormalized(x, z)
					sample := m.Sample(x, z)
					if sample.FinalHeight >= baseHeight-0.00020 {
						continue
					}
					if sample.WaterKind != reconcile.WaterNone {
						continue
					}
					add(feature.kind, x, z, strength)
				}

				if counts["levee"] < perKindLimit &&
					riverNaturalLeveeEnabled(seg) {
					n := 0.08
					distance := bankRadius + n*(reach-bankRadius)
					x := cx + nx*side*distance
					z := cz + nz*side*distance
					bucket := m.bucketIndexForWorld(x, z)
					if bucket >= 0 &&
						!riverBankProtectedAt(m.Rivers, m.riverBuckets[bucket], x, z) {
						lift := riverNaturalLeveeLift(seg, t, x, z, n)
						if lift > 0.00012 {
							baseHeight := m.Base.SampleNormalized(x, z)
							sample := m.Sample(x, z)
							if sample.FinalHeight < baseHeight-0.00010 &&
								sample.WaterKind == reconcile.WaterNone {
								add("levee", x, z, lift)
							}
						}
					}
				}
			}

			if counts["oxbow"] < perKindLimit &&
				key%13 == 0 &&
				(key/13)%3 == 0 &&
				strength >= 0.62 &&
				!seg.OceanMouth &&
				!seg.LakeMouth &&
				!seg.LakeOutlet &&
				seg.HalfWidth >= 18 {
				offset := riverAbandonedMeanderOffset(seg, t, key)
				x := cx + nx*offset
				z := cz + nz*offset
				activeDistance := math.Abs(offset)
				bucket := m.bucketIndexForWorld(x, z)
				if bucket < 0 ||
					riverBankProtectedAt(m.Rivers, m.riverBuckets[bucket], x, z) {
					continue
				}
				_, _, weight, ok := riverOxbowWater(
					m.Config,
					seg,
					t,
					x,
					z,
					activeDistance,
				)
				if !ok || weight <= 0.08 {
					continue
				}
				sample := m.Sample(x, z)
				if sample.WaterKind == reconcile.WaterLake {
					add("oxbow", x, z, weight)
				}
			}
		}
	}

	return out
}
