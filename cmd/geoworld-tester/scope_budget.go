package main

import (
	"geoworld/internal/generator"
	"geoworld/internal/phase1"
	"geoworld/internal/scopebudget"
)

func buildScopeBudget(cfg generator.ResolvedConfig) scopebudget.Summary {
	return scopebudget.Evaluate(scopebudget.Inputs{
		SpanXBlocks: cfg.Geometry.SpanXBlocks,
		SpanZBlocks: cfg.Geometry.SpanZBlocks,
		Components: []scopebudget.Component{
			{
				Name:   "phase1_macro_transition",
				Blocks: phase1.LargestStaticMacroTransitionBlocks(),
			},
			{
				Name:   "climate_coastal_decay",
				Blocks: cfg.Climate.CoastalTempDecayBlocks,
			},
			{
				Name:   "erosion_relief_radius",
				Blocks: cfg.Erosion.ReliefRadiusBlocks,
			},
			{
				Name:   "landform_mountain_influence",
				Blocks: cfg.Landform.MountainInfluenceBlocks,
			},
			{
				Name:   "landform_valley_radius",
				Blocks: cfg.Landform.ValleyRadiusBlocks,
			},
			{
				Name:   "landform_valley_decay",
				Blocks: cfg.Landform.ValleyDecayBlocks,
			},
		},
	})
}
