package specialfeatures

import (
	"testing"

	"geoworld/internal/mcprofile"
)

func TestR82SnowMountainSitingVersionGate(t *testing.T) {
	if usesSnowMountainSiting("geoworld-rules-v1.1.24") {
		t.Fatal("v1.1.24 must keep the locked pre-Slice-D snow siting")
	}
	if !usesSnowMountainSiting("geoworld-rules-v1.1.25") {
		t.Fatal("v1.1.25 must enable R8.2 snow background siting")
	}
}

func TestR82SnowCraterSiteRequiresColdMountainRelief(t *testing.T) {
	good := snowSiteMetrics{
		AnnualMeanTempC:  -9,
		CenterY:          231,
		ReliefBlocks:     122,
		LandSamples512:   32,
		Samples512:       32,
		LandSamples768:   32,
		Samples768:       32,
		TerrainVariant:   mcprofile.TerrainMountain,
		BasinDepthBlocks: -11,
	}
	if !qualifiesSnowCraterSite(good) {
		t.Fatal("representative cold mountain crater background was rejected")
	}

	cases := map[string]snowSiteMetrics{
		"warm":       withSnowTemp(good, 8.5),
		"low":        withSnowCenterY(good, 100),
		"flat":       withSnowRelief(good, 20),
		"plain":      withSnowTerrain(good, mcprofile.TerrainPlain),
		"water-ring": withSnowLand512(good, 20),
	}
	for name, got := range cases {
		if qualifiesSnowCraterSite(got) {
			t.Fatalf("%s crater background unexpectedly qualified: %+v", name, got)
		}
	}
}

func TestR82SnowBasinSiteRequiresColdMountainBasin(t *testing.T) {
	good := snowSiteMetrics{
		AnnualMeanTempC:  3.2,
		CenterY:          291,
		ReliefBlocks:     249,
		BasinDepthBlocks: 4.5,
		LandSamples512:   32,
		Samples512:       32,
		LandSamples768:   32,
		Samples768:       32,
		TerrainVariant:   mcprofile.TerrainMountain,
	}
	if !qualifiesSnowBasinSite(good) {
		t.Fatal("representative cold mountain basin background was rejected")
	}

	cases := map[string]snowSiteMetrics{
		"warm":          withSnowTemp(good, 15),
		"low":           withSnowCenterY(good, 70),
		"flat":          withSnowRelief(good, 15),
		"no-depression": withSnowBasinDepth(good, 2),
		"plain":         withSnowTerrain(good, mcprofile.TerrainPlain),
		"wet-ring":      withSnowLand512(good, 20),
		"coastal-ring":  withSnowLand768(good, 28),
	}
	for name, got := range cases {
		if qualifiesSnowBasinSite(got) {
			t.Fatalf("%s basin background unexpectedly qualified: %+v", name, got)
		}
	}
}

func withSnowTemp(in snowSiteMetrics, v float64) snowSiteMetrics {
	in.AnnualMeanTempC = v
	return in
}
func withSnowCenterY(in snowSiteMetrics, v float64) snowSiteMetrics {
	in.CenterY = v
	return in
}
func withSnowRelief(in snowSiteMetrics, v float64) snowSiteMetrics {
	in.ReliefBlocks = v
	return in
}
func withSnowBasinDepth(in snowSiteMetrics, v float64) snowSiteMetrics {
	in.BasinDepthBlocks = v
	return in
}
func withSnowTerrain(in snowSiteMetrics, v mcprofile.TerrainVariant) snowSiteMetrics {
	in.TerrainVariant = v
	return in
}
func withSnowLand512(in snowSiteMetrics, v int) snowSiteMetrics {
	in.LandSamples512 = v
	return in
}
func withSnowLand768(in snowSiteMetrics, v int) snowSiteMetrics {
	in.LandSamples768 = v
	return in
}
