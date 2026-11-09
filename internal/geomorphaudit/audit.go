package geomorphaudit

import (
	"errors"
	"math"
	"sort"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/sampling"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/waterbed"
)

const EvidenceClassInvestigation = "investigation_signal"

type Config struct {
	RouteSampleStepBlocks float64
	TransitionProbeBlocks float64
	MaxDetailRecords      int
}

type Report struct {
	EvidenceClass string `json:"evidenceClass"`
	Seed          int64  `json:"seed"`
	RulesVersion  string `json:"rulesVersion"`

	River      RiverReport      `json:"river"`
	Sediment   SedimentReport   `json:"sediment"`
	Transition TransitionReport `json:"transition"`
}

type SedimentReport struct {
	Production        float64 `json:"production"`
	Deposition        float64 `json:"deposition"`
	Export            float64 `json:"export"`
	MassBalanceError  float64 `json:"massBalanceError"`
	DepositedFraction float64 `json:"depositedFraction"`
	ExportFraction    float64 `json:"exportFraction"`

	HillslopeProduction                 float64 `json:"hillslopeProduction"`
	NetworkProduction                   float64 `json:"networkProduction"`
	HillslopeDeposition                 float64 `json:"hillslopeDeposition"`
	NetworkDeposition                   float64 `json:"networkDeposition"`
	FanDeposition                       float64 `json:"fanDeposition"`
	FloodplainDeposition                float64 `json:"floodplainDeposition"`
	LakeDeposition                      float64 `json:"lakeDeposition"`
	OtherNetworkDeposition              float64 `json:"otherNetworkDeposition"`
	DepositionPartitionError            float64 `json:"depositionPartitionError"`
	HillslopeExport                     float64 `json:"hillslopeExport"`
	NetworkExport                       float64 `json:"networkExport"`
	OceanReceiverLoad                   float64 `json:"oceanReceiverLoad"`
	OceanReceiverNetworkLoad            float64 `json:"oceanReceiverNetworkLoad"`
	OceanReceiverHillslopeLoad          float64 `json:"oceanReceiverHillslopeLoad"`
	OtherTerminalExport                 float64 `json:"otherTerminalExport"`
	HillslopeToNetworkDelivery          float64 `json:"hillslopeToNetworkDelivery"`
	HillslopeDirectToLakeDelivery       float64 `json:"hillslopeDirectToLakeDelivery"`
	LakeInletLoad                       float64 `json:"lakeInletLoad"`
	NetworkToHillslopeTransfer          float64 `json:"networkToHillslopeTransfer"`
	HillslopeDeliveryFraction           float64 `json:"hillslopeDeliveryFraction"`
	NetworkInputFraction                float64 `json:"networkInputFraction"`
	NetworkDepositedFraction            float64 `json:"networkDepositedFraction"`
	NetworkExportFraction               float64 `json:"networkExportFraction"`
	LakeDepositionFraction              float64 `json:"lakeDepositionFraction"`
	NetworkFanDepositionFraction        float64 `json:"networkFanDepositionFraction"`
	NetworkFloodplainDepositionFraction float64 `json:"networkFloodplainDepositionFraction"`
	NetworkLakeDepositionFraction       float64 `json:"networkLakeDepositionFraction"`
	NetworkOtherDepositionFraction      float64 `json:"networkOtherDepositionFraction"`
	OceanReceiverFraction               float64 `json:"oceanReceiverFraction"`
	OceanReceiverNetworkFraction        float64 `json:"oceanReceiverNetworkFraction"`
	LakeInletLoadFraction               float64 `json:"lakeInletLoadFraction"`
	HillslopeMassBalanceError           float64 `json:"hillslopeMassBalanceError"`
	NetworkMassBalanceError             float64 `json:"networkMassBalanceError"`
	HillslopeToNetworkTransitions       int     `json:"hillslopeToNetworkTransitions"`
	NetworkToHillslopeTransitions       int     `json:"networkToHillslopeTransitions"`
	LakeInletTransitions                int     `json:"lakeInletTransitions"`
	OceanReceiverTransitions            int     `json:"oceanReceiverTransitions"`
	OceanReceiverNetworkTransitions     int     `json:"oceanReceiverNetworkTransitions"`
	OceanReceiverHillslopeTransitions   int     `json:"oceanReceiverHillslopeTransitions"`

	MaxFlux                                float64 `json:"maxFlux"`
	ActiveCells                            int     `json:"activeCells"`
	HillslopeActiveCells                   int     `json:"hillslopeActiveCells"`
	NetworkActiveCells                     int     `json:"networkActiveCells"`
	ProductionCells                        int     `json:"productionCells"`
	DepositionCells                        int     `json:"depositionCells"`
	MeanTransportCapacity                  float64 `json:"meanTransportCapacity"`
	HillslopeMeanTransportCapacity         float64 `json:"hillslopeMeanTransportCapacity"`
	NetworkMeanTransportCapacity           float64 `json:"networkMeanTransportCapacity"`
	HillslopeMeanAccommodation             float64 `json:"hillslopeMeanAccommodation"`
	NetworkMeanAccommodation               float64 `json:"networkMeanAccommodation"`
	NetworkDepositionWeightedAccommodation float64 `json:"networkDepositionWeightedAccommodation"`
	NetworkLowAccommodationDeposition      float64 `json:"networkLowAccommodationDeposition"`
	NetworkLowAccommodationDepositFraction float64 `json:"networkLowAccommodationDepositFraction"`

	FanCoreCells                      int     `json:"fanCoreCells"`
	NetworkFanCoreCells               int     `json:"networkFanCoreCells"`
	FanCoreAvailableLoad              float64 `json:"fanCoreAvailableLoad"`
	FanCoreDeposition                 float64 `json:"fanCoreDeposition"`
	FanCoreDepositionFraction         float64 `json:"fanCoreDepositionFraction"`
	FanCoreMeanTransportCapacity      float64 `json:"fanCoreMeanTransportCapacity"`
	FanCoreMeanAccommodation          float64 `json:"fanCoreMeanAccommodation"`
	FanCoreMeanCapacityDrop           float64 `json:"fanCoreMeanCapacityDrop"`
	FanCoreLoadWeightedCapacityDrop   float64 `json:"fanCoreLoadWeightedCapacityDrop"`
	FanCoreMeanCompetenceLoss         float64 `json:"fanCoreMeanCompetenceLoss"`
	FanCoreLoadWeightedCompetenceLoss float64 `json:"fanCoreLoadWeightedCompetenceLoss"`
}

type RiverReport struct {
	Segments int `json:"segments"`
	Routes   int `json:"routes"`

	RouteDrySamples                    int               `json:"routeDrySamples"`
	TraversedSurfaceRiseSteps          int               `json:"traversedSurfaceRiseSteps"`
	TraversedInternalRiseSteps         int               `json:"traversedInternalRiseSteps"`
	TraversedReceiverRiseSteps         int               `json:"traversedReceiverRiseSteps"`
	UniqueSurfaceRiseSteps             int               `json:"uniqueSurfaceRiseSteps"`
	UniqueInternalRiseSteps            int               `json:"uniqueInternalRiseSteps"`
	UniqueReceiverRiseSteps            int               `json:"uniqueReceiverRiseSteps"`
	InternalRiseAboveQuarterBlock      int               `json:"internalRiseAboveQuarterBlock"`
	InternalRiseAboveOneBlock          int               `json:"internalRiseAboveOneBlock"`
	MaxInternalRiseBlocks              float64           `json:"maxInternalRiseBlocks"`
	MaxReceiverRiseBlocks              float64           `json:"maxReceiverRiseBlocks"`
	DownstreamSteeperRoutes            int               `json:"downstreamSteeperRoutes"`
	LongFreeFlowRoutes                 int               `json:"longFreeFlowRoutes"`
	BroadDownstreamSteepeningSignals   int               `json:"broadDownstreamSteepeningSignals"`
	DrainageOutlets                    int               `json:"drainageOutlets"`
	LongRepresentativeDrainages        int               `json:"longRepresentativeDrainages"`
	BroadDownstreamSteepeningDrainages int               `json:"broadDownstreamSteepeningDrainages"`
	RouteProfiles                      []RouteProfile    `json:"routeProfiles,omitempty"`
	RepresentativeDrainageProfiles     []RouteProfile    `json:"representativeDrainageProfiles,omitempty"`
	InternalRiseSignals                []RiverRiseSignal `json:"internalRiseSignals,omitempty"`
	ReceiverRiseSignals                []RiverRiseSignal `json:"receiverRiseSignals,omitempty"`

	Confluences                    int                `json:"confluences"`
	ConfluenceSurfaceRises         int                `json:"confluenceSurfaceRises"`
	ConfluenceInternalSurfaceRises int                `json:"confluenceInternalSurfaceRises"`
	ConfluenceReceiverSurfaceRises int                `json:"confluenceReceiverSurfaceRises"`
	ConfluenceWidthRegressions     int                `json:"confluenceWidthRegressions"`
	ConfluenceSignals              []ConfluenceSignal `json:"confluenceSignals,omitempty"`

	OceanMouths           int           `json:"oceanMouths"`
	MouthApproachRises    int           `json:"mouthApproachRises"`
	MouthNarrowingSignals int           `json:"mouthNarrowingSignals"`
	MouthSignals          []MouthSignal `json:"mouthSignals,omitempty"`

	LakeMouths  int `json:"lakeMouths"`
	LakeOutlets int `json:"lakeOutlets"`
}

type RouteProfile struct {
	SourceNode                      int     `json:"sourceNode"`
	EndNode                         int     `json:"endNode"`
	Segments                        int     `json:"segments"`
	LengthBlocks                    float64 `json:"lengthBlocks"`
	TotalDropBlocks                 float64 `json:"totalDropBlocks"`
	UpperDropPerKm                  float64 `json:"upperDropPerKm"`
	MiddleDropPerKm                 float64 `json:"middleDropPerKm"`
	LowerDropPerKm                  float64 `json:"lowerDropPerKm"`
	DrySamples                      int     `json:"drySamples"`
	SurfaceRiseSteps                int     `json:"surfaceRiseSteps"`
	InternalRiseSteps               int     `json:"internalRiseSteps"`
	ReceiverRiseSteps               int     `json:"receiverRiseSteps"`
	DownstreamSteeper               bool    `json:"downstreamSteeper"`
	FreeFlowLengthBlocks            float64 `json:"freeFlowLengthBlocks"`
	FreeFlowDropBlocks              float64 `json:"freeFlowDropBlocks"`
	FreeUpperDropPerKm              float64 `json:"freeUpperDropPerKm"`
	FreeMiddleDropPerKm             float64 `json:"freeMiddleDropPerKm"`
	FreeLowerDropPerKm              float64 `json:"freeLowerDropPerKm"`
	LongFreeFlow                    bool    `json:"longFreeFlow"`
	BroadDownstreamSteepening       bool    `json:"broadDownstreamSteepening"`
	EndWaterKind                    string  `json:"endWaterKind"`
	EndOceanMouth                   bool    `json:"endOceanMouth"`
	EndLakeMouth                    bool    `json:"endLakeMouth"`
	EndLakeOutlet                   bool    `json:"endLakeOutlet"`
	FreeUpperMountainWeight         float64 `json:"freeUpperMountainWeight"`
	FreeLowerMountainWeight         float64 `json:"freeLowerMountainWeight"`
	FreeUpperCanyonWeight           float64 `json:"freeUpperCanyonWeight"`
	FreeLowerCanyonWeight           float64 `json:"freeLowerCanyonWeight"`
	FreeUpperValleyWeight           float64 `json:"freeUpperValleyWeight"`
	FreeLowerValleyWeight           float64 `json:"freeLowerValleyWeight"`
	FreeUpperPlainWeight            float64 `json:"freeUpperPlainWeight"`
	FreeLowerPlainWeight            float64 `json:"freeLowerPlainWeight"`
	FreeUpperTerrainDropPerKm       float64 `json:"freeUpperTerrainDropPerKm"`
	FreeLowerTerrainDropPerKm       float64 `json:"freeLowerTerrainDropPerKm"`
	FreeUpperCoastDropPerKm         float64 `json:"freeUpperCoastDropPerKm"`
	FreeLowerCoastDropPerKm         float64 `json:"freeLowerCoastDropPerKm"`
	FreeUpperBaseDropPerKm          float64 `json:"freeUpperBaseDropPerKm"`
	FreeLowerBaseDropPerKm          float64 `json:"freeLowerBaseDropPerKm"`
	FreeUpperFloodplainWeight       float64 `json:"freeUpperFloodplainWeight"`
	FreeLowerFloodplainWeight       float64 `json:"freeLowerFloodplainWeight"`
	FreeUpperAlluvialPlainWeight    float64 `json:"freeUpperAlluvialPlainWeight"`
	FreeLowerAlluvialPlainWeight    float64 `json:"freeLowerAlluvialPlainWeight"`
	FreeUpperDepositionPotential    float64 `json:"freeUpperDepositionPotential"`
	FreeLowerDepositionPotential    float64 `json:"freeLowerDepositionPotential"`
	FreeUpperGradeCorrectionBlocks  float64 `json:"freeUpperGradeCorrectionBlocks"`
	FreeLowerGradeCorrectionBlocks  float64 `json:"freeLowerGradeCorrectionBlocks"`
	FreeMaxAbsGradeCorrectionBlocks float64 `json:"freeMaxAbsGradeCorrectionBlocks"`
}

type RiverRiseSignal struct {
	Segment                int     `json:"segment"`
	T                      float64 `json:"t"`
	X                      float64 `json:"x"`
	Z                      float64 `json:"z"`
	RiseBlocks             float64 `json:"riseBlocks"`
	PreviousKind           string  `json:"previousKind"`
	CurrentKind            string  `json:"currentKind"`
	PreviousSpecialFeature string  `json:"previousSpecialFeature,omitempty"`
	CurrentSpecialFeature  string  `json:"currentSpecialFeature,omitempty"`
	ReceiverRelated        bool    `json:"receiverRelated"`
}

type ConfluenceSignal struct {
	Node                      int     `json:"node"`
	IncomingSegments          int     `json:"incomingSegments"`
	DownstreamSegment         int     `json:"downstreamSegment"`
	MaxSurfaceRiseBlocks      float64 `json:"maxSurfaceRiseBlocks"`
	RequiredHalfWidthBlocks   float64 `json:"requiredHalfWidthBlocks"`
	DownstreamHalfWidthBlocks float64 `json:"downstreamHalfWidthBlocks"`
	WidthRatio                float64 `json:"widthRatio"`
	ReceiverRelated           bool    `json:"receiverRelated"`
}

type MouthSignal struct {
	Segment                   int     `json:"segment"`
	ApproachSurfaceRiseBlocks float64 `json:"approachSurfaceRiseBlocks"`
	WidthAt80Blocks           float64 `json:"widthAt80Blocks"`
	WidthAt98Blocks           float64 `json:"widthAt98Blocks"`
	WidthRatio                float64 `json:"widthRatio"`
	KindAt98                  string  `json:"kindAt98"`
}

type TransitionReport struct {
	Samples int                    `json:"samples"`
	Classes []TransitionClassStats `json:"classes,omitempty"`
}

type TransitionClassStats struct {
	From               string  `json:"from"`
	To                 string  `json:"to"`
	Count              int     `json:"count"`
	SlopeP50Deg        float64 `json:"slopeP50Deg"`
	SlopeP90Deg        float64 `json:"slopeP90Deg"`
	SlopeP99Deg        float64 `json:"slopeP99Deg"`
	MaxSlopeDeg        float64 `json:"maxSlopeDeg"`
	CurvatureP90Blocks float64 `json:"curvatureP90Blocks"`
	MaxCurvatureBlocks float64 `json:"maxCurvatureBlocks"`
}

type routePoint struct {
	distance              float64
	surfaceY              float64
	wet                   bool
	receiverRelated       bool
	segment               int
	t                     float64
	x                     float64
	z                     float64
	waterKind             reconcile.WaterKind
	specialFeature        string
	connectionSegment     bool
	mountainWeight        float64
	canyonWeight          float64
	valleyWeight          float64
	plainWeight           float64
	terrainY              float64
	coastTerrainY         float64
	baseElevationY        float64
	floodplainWeight      float64
	alluvialPlainWeight   float64
	depositionPotential   float64
	gradeCorrectionBlocks float64
}

type transitionAccumulator struct {
	from         string
	to           string
	slopes       []float64
	curvatures   []float64
	maxSlope     float64
	maxCurvature float64
}

func DefaultConfig() Config {
	return Config{
		RouteSampleStepBlocks: 16,
		TransitionProbeBlocks: 16,
		MaxDetailRecords:      64,
	}
}

func Analyze(model *generator.WorldModel, cfg Config) (Report, error) {
	if model == nil ||
		model.Waterbed == nil ||
		model.Geomorph == nil ||
		model.RuntimeProfile == nil ||
		model.Terrain == nil ||
		model.CoastTerrain == nil {
		return Report{}, errors.New("geomorphaudit: complete generated world model is required")
	}
	cfg = normalizeConfig(cfg)
	report := Report{
		EvidenceClass: EvidenceClassInvestigation,
		Seed:          model.Config.Seed,
		RulesVersion:  model.Config.WorldProfile.GenerationRulesVersion,
	}
	auditRivers(model, cfg, &report.River)
	auditSedimentBudget(model, &report.Sediment)
	auditTransitions(model, cfg, &report.Transition)
	return report, nil
}

func auditSedimentBudget(model *generator.WorldModel, report *SedimentReport) {
	if model == nil || model.Geomorph == nil {
		return
	}
	g := model.Geomorph
	activeCapacitySum := 0.0
	hillslopeCapacitySum := 0.0
	networkCapacitySum := 0.0
	hillslopeAccommodationSum := 0.0
	networkAccommodationSum := 0.0
	networkDepositionAccommodation := 0.0
	fanCoreCapacitySum := 0.0
	fanCoreAccommodationSum := 0.0
	fanCoreCapacityDropSum := 0.0
	fanCoreLoadWeightedCapacityDropSum := 0.0
	fanCoreCompetenceLossSum := 0.0
	fanCoreLoadWeightedCompetenceLossSum := 0.0

	incomingMass := make([]float64, len(g.LandMask))
	incomingCapacityMoment := make([]float64, len(g.LandMask))
	if model.Hydrology != nil {
		for i, to := range model.Hydrology.FlowTo {
			if i < 0 || i >= len(g.LandMask) ||
				to < 0 || to >= len(g.LandMask) ||
				!g.LandMask[to] {
				continue
			}
			flux := 0.0
			capacity := 0.0
			if i < len(g.SedimentFlux) {
				flux = g.SedimentFlux[i]
			}
			if i < len(g.TransportCapacity) {
				capacity = g.TransportCapacity[i]
			}
			incomingMass[to] += flux
			incomingCapacityMoment[to] += flux * capacity
		}
	}

	isNetworkCell := func(i int) bool {
		if model.Hydrology == nil || i < 0 {
			return false
		}
		river := i < len(model.Hydrology.RiverStrength) &&
			model.Hydrology.RiverStrength[i] > 0
		lake := i < len(model.Hydrology.LakeID) &&
			model.Hydrology.LakeID[i] > 0
		return river || lake
	}
	isLakeCell := func(i int) bool {
		return model.Hydrology != nil &&
			i >= 0 &&
			i < len(model.Hydrology.LakeID) &&
			model.Hydrology.LakeID[i] > 0
	}

	for i, production := range g.SedimentProduction {
		deposition := 0.0
		exported := 0.0
		flux := 0.0
		capacity := 0.0
		accommodation := 0.0
		if i < len(g.SedimentDeposition) {
			deposition = g.SedimentDeposition[i]
		}
		if i < len(g.SedimentExport) {
			exported = g.SedimentExport[i]
		}
		if i < len(g.SedimentFlux) {
			flux = g.SedimentFlux[i]
		}
		if i < len(g.TransportCapacity) {
			capacity = g.TransportCapacity[i]
		}
		if i < len(g.SedimentAccommodation) {
			accommodation = g.SedimentAccommodation[i]
		}

		report.Production += production
		report.Deposition += deposition
		report.Export += exported
		report.MaxFlux = math.Max(report.MaxFlux, flux)

		network := isNetworkCell(i)
		fanCore := 0.0
		if i < len(g.AlluvialFanCore) {
			fanCore = math.Max(0, math.Min(1, g.AlluvialFanCore[i]))
		}
		if fanCore > 0.05 {
			report.FanCoreCells++
			if network {
				report.NetworkFanCoreCells++
			}
			available := production + incomingMass[i]
			report.FanCoreAvailableLoad += available
			report.FanCoreDeposition += deposition
			fanCoreCapacitySum += capacity
			fanCoreAccommodationSum += accommodation
			capacityDrop := 0.0
			if incomingMass[i] > 1e-12 {
				upstreamCapacity :=
					incomingCapacityMoment[i] / incomingMass[i]
				capacityDrop = math.Max(
					0,
					math.Min(1, upstreamCapacity-capacity),
				)
			}
			fanCoreCapacityDropSum += capacityDrop
			fanCoreLoadWeightedCapacityDropSum +=
				available * capacityDrop
			competenceLoss := 0.0
			if i < len(g.SedimentCompetenceLoss) {
				competenceLoss = math.Max(
					0,
					math.Min(1, g.SedimentCompetenceLoss[i]),
				)
			}
			fanCoreCompetenceLossSum += competenceLoss
			fanCoreLoadWeightedCompetenceLossSum +=
				available * competenceLoss
		}
		if network {
			report.NetworkProduction += production
			report.NetworkDeposition += deposition
			report.NetworkExport += exported
		} else {
			report.HillslopeProduction += production
			report.HillslopeDeposition += deposition
			report.HillslopeExport += exported
		}
		if network && isLakeCell(i) {
			report.LakeDeposition += deposition
		} else if network && deposition > 0 {
			fanWeight := 0.0
			floodplainWeight := 0.0
			if i < len(g.AlluvialFanWeight) {
				fanWeight = math.Max(0, math.Min(1, g.AlluvialFanWeight[i]))
			}
			if i < len(g.FloodplainWeight) {
				floodplainWeight = math.Max(
					floodplainWeight,
					math.Max(0, math.Min(1, g.FloodplainWeight[i])),
				)
			}
			if i < len(g.AlluvialPlainWeight) {
				floodplainWeight = math.Max(
					floodplainWeight,
					math.Max(0, math.Min(1, g.AlluvialPlainWeight[i])),
				)
			}
			weightSum := fanWeight + floodplainWeight
			if weightSum > 1 {
				fanWeight /= weightSum
				floodplainWeight /= weightSum
			}
			fanDeposition := deposition * fanWeight
			floodplainDeposition := deposition * floodplainWeight
			otherNetworkDeposition := math.Max(
				0,
				deposition-fanDeposition-floodplainDeposition,
			)
			report.FanDeposition += fanDeposition
			report.FloodplainDeposition += floodplainDeposition
			report.OtherNetworkDeposition += otherNetworkDeposition
		}

		if production > 1e-12 {
			report.ProductionCells++
		}
		if deposition > 1e-12 {
			report.DepositionCells++
		}
		if production > 1e-12 ||
			deposition > 1e-12 ||
			exported > 1e-12 ||
			flux > 1e-12 {
			report.ActiveCells++
			activeCapacitySum += capacity
			if network {
				report.NetworkActiveCells++
				networkCapacitySum += capacity
				networkAccommodationSum += accommodation
			} else {
				report.HillslopeActiveCells++
				hillslopeCapacitySum += capacity
				hillslopeAccommodationSum += accommodation
			}
		}
		if network && deposition > 1e-12 {
			networkDepositionAccommodation +=
				deposition * accommodation
			if accommodation < 0.15 {
				report.NetworkLowAccommodationDeposition += deposition
			}
		}

		if model.Hydrology != nil &&
			i < len(model.Hydrology.FlowTo) {
			to := model.Hydrology.FlowTo[i]
			validLandTarget := to >= 0 &&
				to < len(g.LandMask) &&
				g.LandMask[to]
			if validLandTarget {
				toNetwork := isNetworkCell(to)
				if !isLakeCell(i) && isLakeCell(to) && flux > 1e-12 {
					report.LakeInletTransitions++
					report.LakeInletLoad += flux
				}
				switch {
				case !network && toNetwork:
					report.HillslopeToNetworkTransitions++
					report.HillslopeToNetworkDelivery += flux
					if isLakeCell(to) {
						report.HillslopeDirectToLakeDelivery += flux
					}
				case network && !toNetwork:
					report.NetworkToHillslopeTransitions++
					report.NetworkToHillslopeTransfer += flux
				}
			} else if exported > 1e-12 &&
				to >= 0 &&
				to < len(g.LandMask) &&
				!g.LandMask[to] {
				report.OceanReceiverTransitions++
				report.OceanReceiverLoad += exported
				if network {
					report.OceanReceiverNetworkTransitions++
					report.OceanReceiverNetworkLoad += exported
				} else {
					report.OceanReceiverHillslopeTransitions++
					report.OceanReceiverHillslopeLoad += exported
				}
			}
		}
	}

	report.MassBalanceError =
		math.Abs(report.Production - report.Deposition - report.Export)
	report.DepositionPartitionError = math.Abs(
		report.Deposition -
			report.HillslopeDeposition -
			report.FanDeposition -
			report.FloodplainDeposition -
			report.LakeDeposition -
			report.OtherNetworkDeposition,
	)
	report.OtherTerminalExport = math.Max(
		0,
		report.Export-report.OceanReceiverLoad,
	)
	if report.Production > 1e-12 {
		report.DepositedFraction = report.Deposition / report.Production
		report.ExportFraction = report.Export / report.Production
		report.OceanReceiverFraction =
			report.OceanReceiverLoad / report.Production
	}

	hillslopeInput :=
		report.HillslopeProduction + report.NetworkToHillslopeTransfer
	networkInput :=
		report.NetworkProduction + report.HillslopeToNetworkDelivery
	if hillslopeInput > 1e-12 {
		report.HillslopeDeliveryFraction =
			report.HillslopeToNetworkDelivery / hillslopeInput
	}
	if report.Production > 1e-12 {
		report.NetworkInputFraction = networkInput / report.Production
	}
	if networkInput > 1e-12 {
		report.NetworkDepositedFraction =
			report.NetworkDeposition / networkInput
		report.NetworkExportFraction =
			report.NetworkExport / networkInput
		report.OceanReceiverNetworkFraction =
			report.OceanReceiverNetworkLoad / networkInput
	}
	if report.Production > 1e-12 {
		report.LakeInletLoadFraction =
			report.LakeInletLoad / report.Production
	}
	report.HillslopeMassBalanceError = math.Abs(
		hillslopeInput -
			report.HillslopeDeposition -
			report.HillslopeExport -
			report.HillslopeToNetworkDelivery,
	)
	report.NetworkMassBalanceError = math.Abs(
		networkInput -
			report.NetworkDeposition -
			report.NetworkExport -
			report.NetworkToHillslopeTransfer,
	)

	if report.Deposition > 1e-12 {
		report.LakeDepositionFraction =
			report.LakeDeposition / report.Deposition
	}
	if report.NetworkDeposition > 1e-12 {
		report.NetworkFanDepositionFraction =
			report.FanDeposition / report.NetworkDeposition
		report.NetworkFloodplainDepositionFraction =
			report.FloodplainDeposition / report.NetworkDeposition
		report.NetworkLakeDepositionFraction =
			report.LakeDeposition / report.NetworkDeposition
		report.NetworkOtherDepositionFraction =
			report.OtherNetworkDeposition / report.NetworkDeposition
	}
	if report.ActiveCells > 0 {
		report.MeanTransportCapacity =
			activeCapacitySum / float64(report.ActiveCells)
	}
	if report.HillslopeActiveCells > 0 {
		report.HillslopeMeanTransportCapacity =
			hillslopeCapacitySum / float64(report.HillslopeActiveCells)
		report.HillslopeMeanAccommodation =
			hillslopeAccommodationSum / float64(report.HillslopeActiveCells)
	}
	if report.NetworkActiveCells > 0 {
		report.NetworkMeanTransportCapacity =
			networkCapacitySum / float64(report.NetworkActiveCells)
		report.NetworkMeanAccommodation =
			networkAccommodationSum / float64(report.NetworkActiveCells)
	}
	if report.NetworkDeposition > 1e-12 {
		report.NetworkDepositionWeightedAccommodation =
			networkDepositionAccommodation / report.NetworkDeposition
		report.NetworkLowAccommodationDepositFraction =
			report.NetworkLowAccommodationDeposition /
				report.NetworkDeposition
	}
	if report.FanCoreCells > 0 {
		report.FanCoreMeanTransportCapacity =
			fanCoreCapacitySum / float64(report.FanCoreCells)
		report.FanCoreMeanAccommodation =
			fanCoreAccommodationSum / float64(report.FanCoreCells)
		report.FanCoreMeanCapacityDrop =
			fanCoreCapacityDropSum / float64(report.FanCoreCells)
		report.FanCoreMeanCompetenceLoss =
			fanCoreCompetenceLossSum / float64(report.FanCoreCells)
	}
	if report.FanCoreAvailableLoad > 1e-12 {
		report.FanCoreDepositionFraction =
			report.FanCoreDeposition / report.FanCoreAvailableLoad
		report.FanCoreLoadWeightedCapacityDrop =
			fanCoreLoadWeightedCapacityDropSum / report.FanCoreAvailableLoad
		report.FanCoreLoadWeightedCompetenceLoss =
			fanCoreLoadWeightedCompetenceLossSum / report.FanCoreAvailableLoad
	}
}

func normalizeConfig(cfg Config) Config {
	d := DefaultConfig()
	if !finite(cfg.RouteSampleStepBlocks) || cfg.RouteSampleStepBlocks <= 0 {
		cfg.RouteSampleStepBlocks = d.RouteSampleStepBlocks
	}
	if !finite(cfg.TransitionProbeBlocks) || cfg.TransitionProbeBlocks <= 0 {
		cfg.TransitionProbeBlocks = d.TransitionProbeBlocks
	}
	if cfg.MaxDetailRecords <= 0 {
		cfg.MaxDetailRecords = d.MaxDetailRecords
	}
	return cfg
}

func auditRivers(model *generator.WorldModel, cfg Config, report *RiverReport) {
	report.Segments = len(model.Waterbed.Rivers)
	incoming := make(map[int][]int)
	outgoing := make(map[int]int)

	var gradeGrid *sampling.ScalarGrid
	if model.Geomorph != nil &&
		len(model.Geomorph.LongitudinalGradeCorrection) ==
			model.Geomorph.Width*model.Geomorph.Height {
		if grid, err := sampling.NewScalarGridWithGeometry(
			model.Geomorph.Geometry,
			model.Geomorph.LongitudinalGradeCorrection,
		); err == nil {
			gradeGrid = &grid
		}
	}
	type riseKey struct {
		segment int
		t       int64
	}
	uniqueRises := make(map[riseKey]RiverRiseSignal)
	for id, seg := range model.Waterbed.Rivers {
		incoming[seg.To] = append(incoming[seg.To], id)
		if previous, exists := outgoing[seg.From]; !exists ||
			model.Waterbed.Rivers[previous].HalfWidth < seg.HalfWidth {
			outgoing[seg.From] = id
		}
		if seg.OceanMouth {
			report.OceanMouths++
		}
		if seg.LakeMouth {
			report.LakeMouths++
		}
		if seg.LakeOutlet {
			report.LakeOutlets++
		}
	}

	for id, first := range model.Waterbed.Rivers {
		if len(incoming[first.From]) != 0 {
			continue
		}
		route := traceRoute(model.Waterbed.Rivers, id, outgoing)
		if len(route) == 0 {
			continue
		}
		profile, rises := profileRoute(model, cfg, route, gradeGrid)
		report.Routes++
		for _, rise := range rises {
			key := riseKey{segment: rise.Segment, t: int64(math.Round(rise.T * 1e9))}
			if _, exists := uniqueRises[key]; exists {
				continue
			}
			uniqueRises[key] = rise
		}
		report.RouteDrySamples += profile.DrySamples
		report.TraversedSurfaceRiseSteps += profile.SurfaceRiseSteps
		report.TraversedInternalRiseSteps += profile.InternalRiseSteps
		report.TraversedReceiverRiseSteps += profile.ReceiverRiseSteps
		if profile.DownstreamSteeper {
			report.DownstreamSteeperRoutes++
		}
		if profile.LongFreeFlow {
			report.LongFreeFlowRoutes++
		}
		if profile.BroadDownstreamSteepening {
			report.BroadDownstreamSteepeningSignals++
		}
		report.RouteProfiles = append(report.RouteProfiles, profile)
	}

	for _, rise := range uniqueRises {
		report.UniqueSurfaceRiseSteps++
		if rise.ReceiverRelated {
			report.UniqueReceiverRiseSteps++
			report.MaxReceiverRiseBlocks = math.Max(report.MaxReceiverRiseBlocks, rise.RiseBlocks)
			if len(report.ReceiverRiseSignals) < cfg.MaxDetailRecords {
				report.ReceiverRiseSignals = append(report.ReceiverRiseSignals, rise)
			}
			continue
		}
		report.UniqueInternalRiseSteps++
		report.MaxInternalRiseBlocks = math.Max(report.MaxInternalRiseBlocks, rise.RiseBlocks)
		if rise.RiseBlocks >= 0.25 {
			report.InternalRiseAboveQuarterBlock++
		}
		if rise.RiseBlocks >= 1.0 {
			report.InternalRiseAboveOneBlock++
		}
		if len(report.InternalRiseSignals) < cfg.MaxDetailRecords {
			report.InternalRiseSignals = append(report.InternalRiseSignals, rise)
		}
	}

	bestByOutlet := make(map[int]RouteProfile)
	for _, profile := range report.RouteProfiles {
		best, exists := bestByOutlet[profile.EndNode]
		if !exists ||
			profile.FreeFlowLengthBlocks > best.FreeFlowLengthBlocks ||
			(profile.FreeFlowLengthBlocks == best.FreeFlowLengthBlocks &&
				profile.LengthBlocks > best.LengthBlocks) {
			bestByOutlet[profile.EndNode] = profile
		}
	}
	report.DrainageOutlets = len(bestByOutlet)
	for _, profile := range bestByOutlet {
		if profile.LongFreeFlow {
			report.LongRepresentativeDrainages++
		}
		if profile.BroadDownstreamSteepening {
			report.BroadDownstreamSteepeningDrainages++
		}
		report.RepresentativeDrainageProfiles = append(
			report.RepresentativeDrainageProfiles,
			profile,
		)
	}

	sort.Slice(report.InternalRiseSignals, func(i, j int) bool {
		return report.InternalRiseSignals[i].RiseBlocks > report.InternalRiseSignals[j].RiseBlocks
	})
	sort.Slice(report.ReceiverRiseSignals, func(i, j int) bool {
		return report.ReceiverRiseSignals[i].RiseBlocks > report.ReceiverRiseSignals[j].RiseBlocks
	})
	sort.Slice(report.RepresentativeDrainageProfiles, func(i, j int) bool {
		return report.RepresentativeDrainageProfiles[i].FreeFlowLengthBlocks >
			report.RepresentativeDrainageProfiles[j].FreeFlowLengthBlocks
	})
	if len(report.RepresentativeDrainageProfiles) > cfg.MaxDetailRecords {
		report.RepresentativeDrainageProfiles =
			report.RepresentativeDrainageProfiles[:cfg.MaxDetailRecords]
	}
	sort.Slice(report.RouteProfiles, func(i, j int) bool {
		if report.RouteProfiles[i].FreeFlowLengthBlocks != report.RouteProfiles[j].FreeFlowLengthBlocks {
			return report.RouteProfiles[i].FreeFlowLengthBlocks > report.RouteProfiles[j].FreeFlowLengthBlocks
		}
		return report.RouteProfiles[i].LengthBlocks > report.RouteProfiles[j].LengthBlocks
	})
	if len(report.RouteProfiles) > cfg.MaxDetailRecords {
		report.RouteProfiles = report.RouteProfiles[:cfg.MaxDetailRecords]
	}

	auditConfluences(model, cfg, incoming, outgoing, report)
	auditMouths(model, cfg, report)
}

func traceRoute(segments []waterbed.RiverSegment, start int, outgoing map[int]int) []int {
	route := make([]int, 0, 8)
	seen := make(map[int]bool)
	current := start
	for current >= 0 && current < len(segments) && !seen[current] {
		seen[current] = true
		route = append(route, current)
		next, ok := outgoing[segments[current].To]
		if !ok {
			break
		}
		current = next
	}
	return route
}

func profileRoute(
	model *generator.WorldModel,
	cfg Config,
	route []int,
	gradeGrid *sampling.ScalarGrid,
) (RouteProfile, []RiverRiseSignal) {
	first := model.Waterbed.Rivers[route[0]]
	last := model.Waterbed.Rivers[route[len(route)-1]]
	points := make([]routePoint, 0, len(route)*16)
	totalDistance := 0.0
	havePrevious := false
	var previousX, previousZ float64

	for _, id := range route {
		seg := model.Waterbed.Rivers[id]
		lengthUpperBound := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) +
			math.Pi*math.Abs(seg.MeanderOffset)
		steps := int(math.Ceil(lengthUpperBound / cfg.RouteSampleStepBlocks))
		if steps < 2 {
			steps = 2
		}
		for j := 0; j <= steps; j++ {
			if len(points) > 0 && j == 0 {
				continue
			}
			t := float64(j) / float64(steps)
			x, z := waterbed.RiverCenterPoint(seg, t)
			if havePrevious {
				totalDistance += math.Hypot(x-previousX, z-previousZ)
			}
			previousX, previousZ = x, z
			havePrevious = true
			sample := finalRuntimeSample(model, x, z)
			ground := model.Waterbed.Sample(x, z)
			terrainSample := model.Terrain.Sample(x, z)
			gradeCorrectionBlocks := 0.0
			if gradeGrid != nil {
				correction := gradeGrid.SampleCubicBSpline(x, z)
				gradeCorrectionBlocks =
					minecraftY(terrainSample.BaseElevation) -
						minecraftY(terrainSample.BaseElevation-correction)
			}
			wet := sample.World.WaterSurfaceDefined &&
				sample.World.WaterKind != reconcile.WaterNone
			surfaceY := 0.0
			if wet {
				surfaceY = minecraftY(sample.World.WaterSurfaceElevation)
			}
			points = append(points, routePoint{
				distance: totalDistance,
				surfaceY: surfaceY,
				wet:      wet,
				receiverRelated: ground.RiverReceiverSurfaceDefined ||
					sample.World.WaterKind == reconcile.WaterLake ||
					sample.World.WaterKind == reconcile.WaterOcean,
				segment:               id,
				t:                     t,
				x:                     x,
				z:                     z,
				waterKind:             sample.World.WaterKind,
				specialFeature:        sample.SpecialFeature,
				connectionSegment:     seg.OceanMouth || seg.LakeMouth || seg.LakeOutlet,
				mountainWeight:        terrainSample.MountainWeight,
				canyonWeight:          terrainSample.CanyonWeight,
				valleyWeight:          terrainSample.ValleyWeight,
				plainWeight:           terrainSample.RollingPlainWeight,
				terrainY:              minecraftY(terrainSample.FinalHeight),
				coastTerrainY:         minecraftY(model.CoastTerrain.SampleNormalized(x, z)),
				baseElevationY:        minecraftY(terrainSample.BaseElevation),
				floodplainWeight:      terrainSample.FloodplainWeight,
				alluvialPlainWeight:   terrainSample.AlluvialPlainWeight,
				depositionPotential:   terrainSample.DepositionPotential,
				gradeCorrectionBlocks: gradeCorrectionBlocks,
			})
		}
	}

	out := RouteProfile{
		SourceNode:    first.From,
		EndNode:       last.To,
		Segments:      len(route),
		LengthBlocks:  totalDistance,
		EndOceanMouth: last.OceanMouth,
		EndLakeMouth:  last.LakeMouth,
		EndLakeOutlet: last.LakeOutlet,
	}
	if len(points) > 0 {
		out.EndWaterKind = waterKindName(points[len(points)-1].waterKind)
	}
	if len(points) == 0 || totalDistance <= 1e-9 {
		return out, nil
	}

	var rises []RiverRiseSignal
	var previousWet bool
	var previousSurface float64
	var previousReceiverRelated bool
	var previousPoint routePoint
	for _, p := range points {
		if !p.wet {
			out.DrySamples++
			previousWet = false
			continue
		}
		if previousWet && p.surfaceY > previousSurface+0.01 {
			out.SurfaceRiseSteps++
			receiverRelated := previousReceiverRelated || p.receiverRelated
			if receiverRelated {
				out.ReceiverRiseSteps++
			} else {
				out.InternalRiseSteps++
			}
			rises = append(rises, RiverRiseSignal{
				Segment:                p.segment,
				T:                      p.t,
				X:                      p.x,
				Z:                      p.z,
				RiseBlocks:             p.surfaceY - previousSurface,
				PreviousKind:           waterKindName(previousPoint.waterKind),
				CurrentKind:            waterKindName(p.waterKind),
				PreviousSpecialFeature: previousPoint.specialFeature,
				CurrentSpecialFeature:  p.specialFeature,
				ReceiverRelated:        receiverRelated,
			})
		}
		previousSurface = p.surfaceY
		previousReceiverRelated = p.receiverRelated
		previousPoint = p
		previousWet = true
	}

	startY, okStart := surfaceAt(points, 0)
	endY, okEnd := surfaceAt(points, totalDistance)
	q1Y, okQ1 := surfaceAt(points, totalDistance/3)
	q2Y, okQ2 := surfaceAt(points, 2*totalDistance/3)
	if okStart && okEnd {
		out.TotalDropBlocks = startY - endY
	}
	thirdKm := totalDistance / 3 / 1000
	if thirdKm > 1e-9 {
		if okStart && okQ1 {
			out.UpperDropPerKm = (startY - q1Y) / thirdKm
		}
		if okQ1 && okQ2 {
			out.MiddleDropPerKm = (q1Y - q2Y) / thirdKm
		}
		if okQ2 && okEnd {
			out.LowerDropPerKm = (q2Y - endY) / thirdKm
		}
	}
	out.DownstreamSteeper =
		out.LowerDropPerKm > out.UpperDropPerKm*1.75 &&
			out.LowerDropPerKm-out.UpperDropPerKm > 0.5

	free := longestFreeFlowRun(points)
	if len(free) >= 2 {
		freeStart := free[0].distance
		freeEnd := free[len(free)-1].distance
		freeLength := freeEnd - freeStart
		out.FreeFlowLengthBlocks = freeLength
		summarizeFreeFlowTerrain(&out, free)
		if freeLength > 1e-9 {
			start, okStart := surfaceAt(free, freeStart)
			q1, okQ1 := surfaceAt(free, freeStart+freeLength/3)
			q2, okQ2 := surfaceAt(free, freeStart+2*freeLength/3)
			end, okEnd := surfaceAt(free, freeEnd)
			thirdKm := freeLength / 3 / 1000
			if okStart && okEnd {
				out.FreeFlowDropBlocks = start - end
			}
			if thirdKm > 1e-9 {
				if okStart && okQ1 {
					out.FreeUpperDropPerKm = (start - q1) / thirdKm
				}
				if okQ1 && okQ2 {
					out.FreeMiddleDropPerKm = (q1 - q2) / thirdKm
				}
				if okQ2 && okEnd {
					out.FreeLowerDropPerKm = (q2 - end) / thirdKm
				}

				terrainStart, terrainStartOK := routeScalarAt(
					free, freeStart, func(p routePoint) float64 { return p.terrainY },
				)
				terrainQ1, terrainQ1OK := routeScalarAt(
					free, freeStart+freeLength/3, func(p routePoint) float64 { return p.terrainY },
				)
				terrainQ2, terrainQ2OK := routeScalarAt(
					free, freeStart+2*freeLength/3, func(p routePoint) float64 { return p.terrainY },
				)
				terrainEnd, terrainEndOK := routeScalarAt(
					free, freeEnd, func(p routePoint) float64 { return p.terrainY },
				)
				if terrainStartOK && terrainQ1OK {
					out.FreeUpperTerrainDropPerKm = (terrainStart - terrainQ1) / thirdKm
				}
				if terrainQ2OK && terrainEndOK {
					out.FreeLowerTerrainDropPerKm = (terrainQ2 - terrainEnd) / thirdKm
				}

				coastStart, coastStartOK := routeScalarAt(
					free, freeStart, func(p routePoint) float64 { return p.coastTerrainY },
				)
				coastQ1, coastQ1OK := routeScalarAt(
					free, freeStart+freeLength/3, func(p routePoint) float64 { return p.coastTerrainY },
				)
				coastQ2, coastQ2OK := routeScalarAt(
					free, freeStart+2*freeLength/3, func(p routePoint) float64 { return p.coastTerrainY },
				)
				coastEnd, coastEndOK := routeScalarAt(
					free, freeEnd, func(p routePoint) float64 { return p.coastTerrainY },
				)
				if coastStartOK && coastQ1OK {
					out.FreeUpperCoastDropPerKm = (coastStart - coastQ1) / thirdKm
				}
				if coastQ2OK && coastEndOK {
					out.FreeLowerCoastDropPerKm = (coastQ2 - coastEnd) / thirdKm
				}

				baseStart, baseStartOK := routeScalarAt(
					free, freeStart, func(p routePoint) float64 { return p.baseElevationY },
				)
				baseQ1, baseQ1OK := routeScalarAt(
					free, freeStart+freeLength/3, func(p routePoint) float64 { return p.baseElevationY },
				)
				baseQ2, baseQ2OK := routeScalarAt(
					free, freeStart+2*freeLength/3, func(p routePoint) float64 { return p.baseElevationY },
				)
				baseEnd, baseEndOK := routeScalarAt(
					free, freeEnd, func(p routePoint) float64 { return p.baseElevationY },
				)
				if baseStartOK && baseQ1OK {
					out.FreeUpperBaseDropPerKm = (baseStart - baseQ1) / thirdKm
				}
				if baseQ2OK && baseEndOK {
					out.FreeLowerBaseDropPerKm = (baseQ2 - baseEnd) / thirdKm
				}
			}
		}
	}
	out.LongFreeFlow = out.FreeFlowLengthBlocks >= 1500
	out.BroadDownstreamSteepening =
		out.LongFreeFlow &&
			out.FreeLowerDropPerKm > out.FreeUpperDropPerKm*1.5 &&
			out.FreeLowerDropPerKm-out.FreeUpperDropPerKm >= 5

	return out, rises
}

func longestFreeFlowRun(points []routePoint) []routePoint {
	bestStart, bestEnd := -1, -1
	start := -1
	for i, point := range points {
		free := point.wet &&
			point.waterKind == reconcile.WaterRiver &&
			!point.receiverRelated &&
			!point.connectionSegment
		if free {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			if bestStart < 0 ||
				points[i-1].distance-points[start].distance >
					points[bestEnd].distance-points[bestStart].distance {
				bestStart, bestEnd = start, i-1
			}
			start = -1
		}
	}
	if start >= 0 {
		end := len(points) - 1
		if bestStart < 0 ||
			points[end].distance-points[start].distance >
				points[bestEnd].distance-points[bestStart].distance {
			bestStart, bestEnd = start, end
		}
	}
	if bestStart < 0 || bestEnd <= bestStart {
		return nil
	}
	return points[bestStart : bestEnd+1]
}

func summarizeFreeFlowTerrain(out *RouteProfile, points []routePoint) {
	if out == nil || len(points) == 0 {
		return
	}
	start := points[0].distance
	length := points[len(points)-1].distance - start
	if length <= 1e-9 {
		return
	}
	upperCount, lowerCount := 0.0, 0.0
	for _, point := range points {
		out.FreeMaxAbsGradeCorrectionBlocks = math.Max(
			out.FreeMaxAbsGradeCorrectionBlocks,
			math.Abs(point.gradeCorrectionBlocks),
		)
		q := (point.distance - start) / length
		switch {
		case q <= 1.0/3.0:
			upperCount++
			out.FreeUpperMountainWeight += point.mountainWeight
			out.FreeUpperCanyonWeight += point.canyonWeight
			out.FreeUpperValleyWeight += point.valleyWeight
			out.FreeUpperPlainWeight += point.plainWeight
			out.FreeUpperFloodplainWeight += point.floodplainWeight
			out.FreeUpperAlluvialPlainWeight += point.alluvialPlainWeight
			out.FreeUpperDepositionPotential += point.depositionPotential
			out.FreeUpperGradeCorrectionBlocks += point.gradeCorrectionBlocks
		case q >= 2.0/3.0:
			lowerCount++
			out.FreeLowerMountainWeight += point.mountainWeight
			out.FreeLowerCanyonWeight += point.canyonWeight
			out.FreeLowerValleyWeight += point.valleyWeight
			out.FreeLowerPlainWeight += point.plainWeight
			out.FreeLowerFloodplainWeight += point.floodplainWeight
			out.FreeLowerAlluvialPlainWeight += point.alluvialPlainWeight
			out.FreeLowerDepositionPotential += point.depositionPotential
			out.FreeLowerGradeCorrectionBlocks += point.gradeCorrectionBlocks
		}
	}
	if upperCount > 0 {
		out.FreeUpperMountainWeight /= upperCount
		out.FreeUpperCanyonWeight /= upperCount
		out.FreeUpperValleyWeight /= upperCount
		out.FreeUpperPlainWeight /= upperCount
		out.FreeUpperFloodplainWeight /= upperCount
		out.FreeUpperAlluvialPlainWeight /= upperCount
		out.FreeUpperDepositionPotential /= upperCount
		out.FreeUpperGradeCorrectionBlocks /= upperCount
	}
	if lowerCount > 0 {
		out.FreeLowerMountainWeight /= lowerCount
		out.FreeLowerCanyonWeight /= lowerCount
		out.FreeLowerValleyWeight /= lowerCount
		out.FreeLowerPlainWeight /= lowerCount
		out.FreeLowerFloodplainWeight /= lowerCount
		out.FreeLowerAlluvialPlainWeight /= lowerCount
		out.FreeLowerDepositionPotential /= lowerCount
		out.FreeLowerGradeCorrectionBlocks /= lowerCount
	}
}

func routeScalarAt(
	points []routePoint,
	distance float64,
	value func(routePoint) float64,
) (float64, bool) {
	if len(points) == 0 || value == nil {
		return 0, false
	}
	if distance <= points[0].distance {
		return value(points[0]), true
	}
	for i := 1; i < len(points); i++ {
		if points[i].distance < distance {
			continue
		}
		a, b := points[i-1], points[i]
		span := b.distance - a.distance
		if span <= 1e-9 {
			return value(b), true
		}
		t := (distance - a.distance) / span
		return value(a) + (value(b)-value(a))*t, true
	}
	return value(points[len(points)-1]), true
}

func surfaceAt(points []routePoint, distance float64) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	if distance <= 0 {
		for _, p := range points {
			if p.wet {
				return p.surfaceY, true
			}
		}
		return 0, false
	}
	for i := 1; i < len(points); i++ {
		if points[i].distance < distance {
			continue
		}
		a, b := points[i-1], points[i]
		if !a.wet || !b.wet {
			continue
		}
		span := b.distance - a.distance
		if span <= 1e-9 {
			return b.surfaceY, true
		}
		t := (distance - a.distance) / span
		return a.surfaceY + (b.surfaceY-a.surfaceY)*t, true
	}
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].wet {
			return points[i].surfaceY, true
		}
	}
	return 0, false
}

func auditConfluences(
	model *generator.WorldModel,
	cfg Config,
	incoming map[int][]int,
	outgoing map[int]int,
	report *RiverReport,
) {
	for node, ids := range incoming {
		if len(ids) < 2 {
			continue
		}
		downID, ok := outgoing[node]
		if !ok {
			continue
		}
		report.Confluences++
		down := model.Waterbed.Rivers[downID]
		dx, dz := waterbed.RiverCenterPoint(down, 0.04)
		downGround := model.Waterbed.Sample(dx, dz)
		downSample := finalRuntimeSample(model, dx, dz)
		receiverRelated := downGround.RiverReceiverSurfaceDefined ||
			downSample.World.WaterKind == reconcile.WaterLake ||
			downSample.World.WaterKind == reconcile.WaterOcean
		// The confluence contract is persisted on the downstream segment itself.
		// NavigationSampleAt can select an overlapping incoming segment at the
		// exact junction because both bank distances are zero, producing a false
		// width regression signal.
		downWidth := down.HalfWidth

		sumSquares := 0.0
		maxIncoming := 0.0
		maxRiseBlocks := 0.0
		for _, inID := range ids {
			inSeg := model.Waterbed.Rivers[inID]
			sumSquares += inSeg.HalfWidth * inSeg.HalfWidth
			maxIncoming = math.Max(maxIncoming, inSeg.HalfWidth)
			x, z := waterbed.RiverCenterPoint(inSeg, 0.96)
			inGround := model.Waterbed.Sample(x, z)
			inSample := finalRuntimeSample(model, x, z)
			if inGround.RiverReceiverSurfaceDefined ||
				inSample.World.WaterKind == reconcile.WaterLake ||
				inSample.World.WaterKind == reconcile.WaterOcean {
				receiverRelated = true
			}
			if !wetRuntime(inSample) || !wetRuntime(downSample) {
				continue
			}
			rise := minecraftY(downSample.World.WaterSurfaceElevation) -
				minecraftY(inSample.World.WaterSurfaceElevation)
			maxRiseBlocks = math.Max(maxRiseBlocks, rise)
		}
		required := math.Max(maxIncoming, 0.86*math.Sqrt(sumSquares))
		ratio := 1.0
		if required > 1e-9 {
			ratio = downWidth / required
		}
		surfaceSignal := maxRiseBlocks > 0.25
		widthSignal := ratio < 1
		if surfaceSignal {
			report.ConfluenceSurfaceRises++
			if receiverRelated {
				report.ConfluenceReceiverSurfaceRises++
			} else {
				report.ConfluenceInternalSurfaceRises++
			}
		}
		if widthSignal {
			report.ConfluenceWidthRegressions++
		}
		if (surfaceSignal || widthSignal) &&
			len(report.ConfluenceSignals) < cfg.MaxDetailRecords {
			report.ConfluenceSignals = append(report.ConfluenceSignals, ConfluenceSignal{
				Node:                      node,
				IncomingSegments:          len(ids),
				DownstreamSegment:         downID,
				MaxSurfaceRiseBlocks:      maxRiseBlocks,
				RequiredHalfWidthBlocks:   required,
				DownstreamHalfWidthBlocks: downWidth,
				WidthRatio:                ratio,
				ReceiverRelated:           receiverRelated,
			})
		}
	}
}

func auditMouths(model *generator.WorldModel, cfg Config, report *RiverReport) {
	for id, seg := range model.Waterbed.Rivers {
		if !seg.OceanMouth {
			continue
		}
		x80, z80 := waterbed.RiverCenterPoint(seg, 0.80)
		x98, z98 := waterbed.RiverCenterPoint(seg, 0.98)
		s80 := finalRuntimeSample(model, x80, z80)
		s98 := finalRuntimeSample(model, x98, z98)

		rise := 0.0
		if wetRuntime(s80) && wetRuntime(s98) {
			rise = minecraftY(s98.World.WaterSurfaceElevation) -
				minecraftY(s80.World.WaterSurfaceElevation)
		}
		w80 := seg.HalfWidth
		w98 := seg.HalfWidth
		if nav, ok := model.Waterbed.NavigationSampleAt(x80, z80); ok {
			w80 = nav.HalfWidth
		}
		if nav, ok := model.Waterbed.NavigationSampleAt(x98, z98); ok {
			w98 = nav.HalfWidth
		}
		ratio := 1.0
		if w80 > 1e-9 {
			ratio = w98 / w80
		}
		riseSignal := rise > 0.25
		// If the approach sample is already owned by Ocean, the river channel
		// width is no longer the visible receiving-water width. Treating that
		// handoff as a narrowed river is a false signal.
		narrowSignal := ratio < 0.95 && s98.World.WaterKind == reconcile.WaterRiver
		if riseSignal {
			report.MouthApproachRises++
		}
		if narrowSignal {
			report.MouthNarrowingSignals++
		}
		if (riseSignal || narrowSignal) && len(report.MouthSignals) < cfg.MaxDetailRecords {
			report.MouthSignals = append(report.MouthSignals, MouthSignal{
				Segment:                   id,
				ApproachSurfaceRiseBlocks: rise,
				WidthAt80Blocks:           w80,
				WidthAt98Blocks:           w98,
				WidthRatio:                ratio,
				KindAt98:                  waterKindName(s98.World.WaterKind),
			})
		}
	}
}

func auditTransitions(model *generator.WorldModel, cfg Config, report *TransitionReport) {
	g := model.Geometry
	acc := make(map[string]*transitionAccumulator)
	add := func(aClass, bClass string, x0, z0, x1, z1 float64) {
		if aClass == bClass {
			return
		}
		from, to := orderedClasses(aClass, bClass)
		key := from + "->" + to
		item := acc[key]
		if item == nil {
			item = &transitionAccumulator{from: from, to: to}
			acc[key] = item
		}
		mx := (x0 + x1) * 0.5
		mz := (z0 + z1) * 0.5
		dx := x1 - x0
		dz := z1 - z0
		length := math.Hypot(dx, dz)
		if length <= 1e-9 {
			return
		}
		ux, uz := dx/length, dz/length
		probe := math.Min(cfg.TransitionProbeBlocks, length*0.25)
		left := finalRuntimeSample(model, mx-ux*probe, mz-uz*probe)
		middle := finalRuntimeSample(model, mx, mz)
		right := finalRuntimeSample(model, mx+ux*probe, mz+uz*probe)
		y0 := minecraftY(left.World.GroundHeight)
		ym := minecraftY(middle.World.GroundHeight)
		y1 := minecraftY(right.World.GroundHeight)
		slope := math.Atan2(math.Abs(y1-y0), 2*probe) * 180 / math.Pi
		curvature := math.Abs(y0 - 2*ym + y1)
		item.slopes = append(item.slopes, slope)
		item.curvatures = append(item.curvatures, curvature)
		item.maxSlope = math.Max(item.maxSlope, slope)
		item.maxCurvature = math.Max(item.maxCurvature, curvature)
		report.Samples++
	}

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			if !model.Landform.LandMask[i] {
				continue
			}
			wx, wz := g.NodePosition(x, z)
			class := landformClass(model, i)
			if x+1 < g.Width {
				j := z*g.Width + x + 1
				if model.Landform.LandMask[j] {
					nx, nz := g.NodePosition(x+1, z)
					add(class, landformClass(model, j), wx, wz, nx, nz)
				}
			}
			if z+1 < g.Height {
				j := (z+1)*g.Width + x
				if model.Landform.LandMask[j] {
					nx, nz := g.NodePosition(x, z+1)
					add(class, landformClass(model, j), wx, wz, nx, nz)
				}
			}
		}
	}

	keys := make([]string, 0, len(acc))
	for key := range acc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		a := acc[key]
		report.Classes = append(report.Classes, TransitionClassStats{
			From:               a.from,
			To:                 a.to,
			Count:              len(a.slopes),
			SlopeP50Deg:        percentile(a.slopes, 0.50),
			SlopeP90Deg:        percentile(a.slopes, 0.90),
			SlopeP99Deg:        percentile(a.slopes, 0.99),
			MaxSlopeDeg:        a.maxSlope,
			CurvatureP90Blocks: percentile(a.curvatures, 0.90),
			MaxCurvatureBlocks: a.maxCurvature,
		})
	}
}

func landformClass(model *generator.WorldModel, i int) string {
	type candidate struct {
		name   string
		weight float64
	}
	values := []candidate{
		{"plain", model.Landform.RollingPlainWeight[i]},
		{"hill", model.Landform.HillWeight[i]},
		{"foothill", model.Landform.FoothillWeight[i]},
		{"mountain", model.Landform.MountainWeight[i]},
		{"plateau", model.Landform.PlateauWeight[i]},
		{"valley", model.Landform.ValleyWeight[i]},
	}
	best := values[0]
	for _, v := range values[1:] {
		if v.weight > best.weight {
			best = v
		}
	}
	return best.name
}

func orderedClasses(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

func finalRuntimeSample(model *generator.WorldModel, x, z float64) runtimeprofile.Sample {
	base := model.RuntimeProfile.Sample(x, z)
	if len(model.SpecialFeatures) == 0 {
		return base
	}
	return specialfeatures.Apply(
		model.SpecialFeatures,
		base,
		model.Config.SeaLevelNormalized,
		x,
		z,
		model.Config.WorldProfile.GenerationRulesVersion,
	)
}

func wetRuntime(s runtimeprofile.Sample) bool {
	return s.World.WaterSurfaceDefined &&
		s.World.WaterKind != reconcile.WaterNone
}

func waterKindName(kind reconcile.WaterKind) string {
	switch kind {
	case reconcile.WaterRiver:
		return "River"
	case reconcile.WaterLake:
		return "Lake"
	case reconcile.WaterOcean:
		return "Ocean"
	default:
		return "None"
	}
}

func minecraftY(height float64) float64 {
	height = math.Max(0, math.Min(1, height))
	if height <= 0.5 {
		return -64 + 127*(height/0.5)
	}
	return 63 + 257*((height-0.5)/0.5)
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	pos := math.Max(0, math.Min(1, q)) * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	t := pos - float64(lo)
	return sorted[lo] + (sorted[hi]-sorted[lo])*t
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
