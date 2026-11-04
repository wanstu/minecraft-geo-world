package scopebudget

import (
	"math"
	"sort"
)

const RequiredScaleCapacity = 4.0

type Component struct {
	Name   string  `json:"name"`
	Blocks float64 `json:"blocks"`
}

type Inputs struct {
	SpanXBlocks float64
	SpanZBlocks float64
	Components  []Component
}

type Summary struct {
	SpanXBlocks              float64     `json:"spanXBlocks"`
	SpanZBlocks              float64     `json:"spanZBlocks"`
	MinSpanBlocks            float64     `json:"minSpanBlocks"`
	MaxScaleBlocks           float64     `json:"maxScaleBlocks"`
	MaxScaleName             string      `json:"maxScaleName"`
	ScaleCapacity            float64     `json:"scaleCapacity"`
	BoundaryReserveFraction  float64     `json:"boundaryReserveFraction"`
	InteriorFraction         float64     `json:"interiorFraction"`
	RecommendedMinSpanBlocks float64     `json:"recommendedMinSpanBlocks"`
	ShortfallBlocks          float64     `json:"shortfallBlocks"`
	Status                   string      `json:"status"`
	Macro06Satisfied         bool        `json:"macro06Satisfied"`
	Components               []Component `json:"components"`
}

func Evaluate(in Inputs) Summary {
	out := Summary{
		SpanXBlocks: in.SpanXBlocks,
		SpanZBlocks: in.SpanZBlocks,
		Components:  append([]Component(nil), in.Components...),
	}
	sort.SliceStable(out.Components, func(i, j int) bool {
		if out.Components[i].Blocks == out.Components[j].Blocks {
			return out.Components[i].Name < out.Components[j].Name
		}
		return out.Components[i].Blocks > out.Components[j].Blocks
	})

	if !finitePositive(in.SpanXBlocks) || !finitePositive(in.SpanZBlocks) {
		out.Status = "invalid"
		return out
	}
	out.MinSpanBlocks = math.Min(in.SpanXBlocks, in.SpanZBlocks)

	for _, component := range out.Components {
		if !finitePositive(component.Blocks) {
			continue
		}
		if component.Blocks > out.MaxScaleBlocks {
			out.MaxScaleBlocks = component.Blocks
			out.MaxScaleName = component.Name
		}
	}
	if out.MaxScaleBlocks <= 0 {
		out.Status = "invalid"
		return out
	}

	out.ScaleCapacity = out.MinSpanBlocks / out.MaxScaleBlocks
	out.BoundaryReserveFraction = clamp01(
		2 * out.MaxScaleBlocks / out.MinSpanBlocks,
	)
	out.InteriorFraction = clamp01(1 - out.BoundaryReserveFraction)
	out.RecommendedMinSpanBlocks =
		RequiredScaleCapacity * out.MaxScaleBlocks
	out.ShortfallBlocks = math.Max(
		0,
		out.RecommendedMinSpanBlocks-out.MinSpanBlocks,
	)
	out.Macro06Satisfied =
		out.ScaleCapacity >= RequiredScaleCapacity &&
			out.InteriorFraction >= 0.50

	switch {
	case out.ScaleCapacity >= 5:
		out.Status = "comfortable"
	case out.Macro06Satisfied:
		out.Status = "adequate"
	case out.ScaleCapacity >= 3:
		out.Status = "crowded"
	default:
		out.Status = "insufficient"
	}
	return out
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0) &&
		value > 0
}

func clamp01(value float64) float64 {
	if !math.IsNaN(value) && !math.IsInf(value, 0) {
		if value <= 0 {
			return 0
		}
		if value >= 1 {
			return 1
		}
		return value
	}
	return 0
}
