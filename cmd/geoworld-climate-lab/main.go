package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"geoworld/internal/climate"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	plates := flag.Int("plates", 18, "plate count")
	ocean := flag.Float64("ocean", 0.68, "target ocean area fraction")
	out := flag.String("out", "prototype/generated/climate", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		PlateCount:    *plates,
		OceanFraction: *ocean,
	})

	model, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := climate.RenderAll(model, *out); err != nil {
		log.Fatal(err)
	}

	minTemp := math.Inf(1)
	maxTemp := math.Inf(-1)
	minPressure := math.Inf(1)
	maxPressure := math.Inf(-1)
	minWind := math.Inf(1)
	maxWind := math.Inf(-1)
	minSST := math.Inf(1)
	maxSST := math.Inf(-1)
	minCurrent := math.Inf(1)
	maxCurrent := math.Inf(-1)
	minCorrection := math.Inf(1)
	maxCorrection := math.Inf(-1)
	minAdjusted := math.Inf(1)
	maxAdjusted := math.Inf(-1)
	minPrecip := math.Inf(1)
	maxPrecip := math.Inf(-1)
	landPrecipSum, oceanPrecipSum := 0.0, 0.0
	landMoistureSum, oceanMoistureSum := 0.0, 0.0
	dryLand, wetLand := 0, 0
	var tropicalLandPrecip, temperateLandPrecip, polarLandPrecip float64
	var tropicalLandN, temperateLandN, polarLandN int
	minSeasonalAmplitude := math.Inf(1)
	maxSeasonalAmplitude := math.Inf(-1)
	maxMonsoon := 0.0
	monsoonSum := 0.0
	monsoonN := 0
	var warmSeasonLandSum, coldSeasonLandSum, seasonalAnnualLandSum float64
	var seasonalityLandSum, monsoonRainLandSum float64
	var seasonalityLandN, monsoonRainLandN int
	maxRainfallSeasonality := 0.0
	maxMonsoonRainRatio := 0.0
	var weakMonsoonWarm, weakMonsoonCold, strongMonsoonWarm, strongMonsoonCold float64
	var weakMonsoonN, strongMonsoonN int
	landSum, oceanSum := 0.0, 0.0
	landN, oceanN := 0, 0
	for i, temp := range model.AnnualMeanTempC {
		if temp < minTemp {
			minTemp = temp
		}
		if temp > maxTemp {
			maxTemp = temp
		}
		if model.PressureIndex[i] < minPressure {
			minPressure = model.PressureIndex[i]
		}
		if model.PressureIndex[i] > maxPressure {
			maxPressure = model.PressureIndex[i]
		}
		wind := math.Hypot(model.WindX[i], model.WindZ[i])
		if wind < minWind {
			minWind = wind
		}
		if wind > maxWind {
			maxWind = wind
		}
		if model.SeasonalAmplitudeC[i] < minSeasonalAmplitude {
			minSeasonalAmplitude = model.SeasonalAmplitudeC[i]
		}
		if model.SeasonalAmplitudeC[i] > maxSeasonalAmplitude {
			maxSeasonalAmplitude = model.SeasonalAmplitudeC[i]
		}
		if model.MonsoonIndex[i] > 0 {
			monsoonSum += model.MonsoonIndex[i]
			monsoonN++
			if model.MonsoonIndex[i] > maxMonsoon {
				maxMonsoon = model.MonsoonIndex[i]
			}
		}
		if model.OceanTempCorrectionC[i] < minCorrection {
			minCorrection = model.OceanTempCorrectionC[i]
		}
		if model.OceanTempCorrectionC[i] > maxCorrection {
			maxCorrection = model.OceanTempCorrectionC[i]
		}
		if model.AdjustedAnnualTempC[i] < minAdjusted {
			minAdjusted = model.AdjustedAnnualTempC[i]
		}
		if model.AdjustedAnnualTempC[i] > maxAdjusted {
			maxAdjusted = model.AdjustedAnnualTempC[i]
		}
		if model.AnnualPrecipMM[i] < minPrecip {
			minPrecip = model.AnnualPrecipMM[i]
		}
		if model.AnnualPrecipMM[i] > maxPrecip {
			maxPrecip = model.AnnualPrecipMM[i]
		}
		if world.Crust[i] == phase1.Continental {
			landSum += temp
			landPrecipSum += model.AnnualPrecipMM[i]
			landMoistureSum += model.Moisture[i]
			warmSeasonLandSum += model.WarmSeasonPrecipMM[i]
			coldSeasonLandSum += model.ColdSeasonPrecipMM[i]
			seasonalAnnualLandSum += model.SeasonalAnnualPrecipMM[i]
			seasonalityLandSum += model.RainfallSeasonality[i]
			seasonalityLandN++
			if model.RainfallSeasonality[i] > maxRainfallSeasonality {
				maxRainfallSeasonality = model.RainfallSeasonality[i]
			}
			if model.MonsoonRainRatio[i] > 0 {
				monsoonRainLandSum += model.MonsoonRainRatio[i]
				monsoonRainLandN++
				if model.MonsoonRainRatio[i] > maxMonsoonRainRatio {
					maxMonsoonRainRatio = model.MonsoonRainRatio[i]
				}
			}
			if model.MonsoonIndex[i] < 0.20 {
				weakMonsoonWarm += model.WarmSeasonPrecipMM[i]
				weakMonsoonCold += model.ColdSeasonPrecipMM[i]
				weakMonsoonN++
			}
			if model.MonsoonIndex[i] >= 0.60 {
				strongMonsoonWarm += model.WarmSeasonPrecipMM[i]
				strongMonsoonCold += model.ColdSeasonPrecipMM[i]
				strongMonsoonN++
			}
			if model.AnnualPrecipMM[i] < 250 {
				dryLand++
			}
			if model.AnnualPrecipMM[i] > 1500 {
				wetLand++
			}
			absLat := math.Abs(model.LatitudeDeg[i])
			switch {
			case absLat < 30:
				tropicalLandPrecip += model.AnnualPrecipMM[i]
				tropicalLandN++
			case absLat < 60:
				temperateLandPrecip += model.AnnualPrecipMM[i]
				temperateLandN++
			default:
				polarLandPrecip += model.AnnualPrecipMM[i]
				polarLandN++
			}
			landN++
		} else {
			oceanSum += temp
			oceanPrecipSum += model.AnnualPrecipMM[i]
			oceanMoistureSum += model.Moisture[i]
			oceanN++
			if model.SeaSurfaceTempC[i] < minSST {
				minSST = model.SeaSurfaceTempC[i]
			}
			if model.SeaSurfaceTempC[i] > maxSST {
				maxSST = model.SeaSurfaceTempC[i]
			}
			current := math.Hypot(model.CurrentX[i], model.CurrentZ[i])
			if current < minCurrent {
				minCurrent = current
			}
			if current > maxCurrent {
				maxCurrent = current
			}
		}
	}

	fmt.Printf("GeoWorld Phase 3.2 Seasonal Precipitation Lab\n")
	fmt.Printf("seed=%d grid=%dx%d latitude=±%.1f°\n", *seed, *width, *height, model.Config.MaxLatitudeDeg)
	fmt.Printf("annual-temp=%.2f..%.2f C land-avg=%.2f C ocean-avg=%.2f C\n",
		minTemp, maxTemp, landSum/float64(landN), oceanSum/float64(oceanN))
	fmt.Printf("equator=%.1f C polar=%.1f C lapse=%.1f C/km max-land-elevation=%.0f m\n",
		model.Config.EquatorTempC, model.Config.PolarTempC, model.Config.LapseRateCPerKm, model.Config.MaxLandElevationM)
	fmt.Printf("pressure-index=%.3f..%.3f wind-magnitude=%.3f..%.3f\n",
		minPressure, maxPressure, minWind, maxWind)
	fmt.Printf("sst=%.2f..%.2f C current-magnitude=%.3f..%.3f\n",
		minSST, maxSST, minCurrent, maxCurrent)
	fmt.Printf("ocean-temp-correction=%.3f..%.3f C adjusted-temp=%.2f..%.2f C\n",
		minCorrection, maxCorrection, minAdjusted, maxAdjusted)
	fmt.Printf("annual-precip=%.1f..%.1f mm land-avg=%.1f mm ocean-avg=%.1f mm\n",
		minPrecip, maxPrecip, landPrecipSum/float64(landN), oceanPrecipSum/float64(oceanN))
	fmt.Printf("moisture land-avg=%.4f ocean-avg=%.4f dry-land=%.1f%% wet-land=%.1f%%\n",
		landMoistureSum/float64(landN), oceanMoistureSum/float64(oceanN),
		float64(dryLand)/float64(landN)*100, float64(wetLand)/float64(landN)*100)
	fmt.Printf("land-precip tropical=%.1f mm temperate=%.1f mm polar=%.1f mm\n",
		safeAvg(tropicalLandPrecip, tropicalLandN),
		safeAvg(temperateLandPrecip, temperateLandN),
		safeAvg(polarLandPrecip, polarLandN))
	avgMonsoon := 0.0
	if monsoonN > 0 {
		avgMonsoon = monsoonSum / float64(monsoonN)
	}
	fmt.Printf("seasonal-amplitude=%.2f..%.2f C monsoon-index avg=%.3f max=%.3f cells=%d\n",
		minSeasonalAmplitude, maxSeasonalAmplitude, avgMonsoon, maxMonsoon, monsoonN)
	avgMonsoonRain := 0.0
	if monsoonRainLandN > 0 {
		avgMonsoonRain = monsoonRainLandSum / float64(monsoonRainLandN)
	}
	fmt.Printf("seasonal-land-precip warm=%.1f mm cold=%.1f mm annual=%.1f mm\n",
		warmSeasonLandSum/float64(landN),
		coldSeasonLandSum/float64(landN),
		seasonalAnnualLandSum/float64(landN))
	fmt.Printf("rainfall-seasonality avg=%.3f max=%.3f monsoon-rain-ratio avg=%.3f max=%.3f\n",
		seasonalityLandSum/float64(seasonalityLandN), maxRainfallSeasonality,
		avgMonsoonRain, maxMonsoonRainRatio)
	fmt.Printf("weak-monsoon warm=%.1f cold=%.1f strong-monsoon warm=%.1f cold=%.1f\n",
		safeAvg(weakMonsoonWarm, weakMonsoonN), safeAvg(weakMonsoonCold, weakMonsoonN),
		safeAvg(strongMonsoonWarm, strongMonsoonN), safeAvg(strongMonsoonCold, strongMonsoonN))
	fmt.Printf("output=%s\n", *out)
}

func safeAvg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
