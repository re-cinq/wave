package listing

import (
	"fmt"
	"sort"

	"github.com/recinq/wave/internal/pipelinecatalog"
)

// ListCompositions discovers pipelines under pipelinesDir and returns those
// that are composition pipelines — i.e. they declare a sub-pipeline, an
// iterate/branch/loop/aggregate/gate step, or are categorised as composition.
func ListCompositions(pipelinesDir string) ([]CompositionInfo, error) {
	pipelines, err := pipelinecatalog.DiscoverPipelines(pipelinesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover pipelines: %w", err)
	}

	var compositions []CompositionInfo
	for _, info := range pipelines {
		p, err := pipelinecatalog.LoadPipelineByName(pipelinesDir, info.Name)
		if err != nil {
			continue
		}

		isComposition := info.Category == "composition"
		var subPipelines []string
		stepTypeSet := make(map[string]bool)

		for _, step := range p.Steps {
			if step.SubPipeline != "" {
				subPipelines = append(subPipelines, step.SubPipeline)
			}
			if step.Iterate != nil {
				stepTypeSet["iterate"] = true
				isComposition = true
			}
			if step.Branch != nil {
				stepTypeSet["branch"] = true
				isComposition = true
			}
			if step.Gate != nil {
				stepTypeSet["gate"] = true
				isComposition = true
			}
			if step.Loop != nil {
				stepTypeSet["loop"] = true
				isComposition = true
			}
			if step.Aggregate != nil {
				stepTypeSet["aggregate"] = true
				isComposition = true
			}
			if step.SubPipeline != "" && step.Iterate == nil && step.Branch == nil {
				stepTypeSet["sub-pipeline"] = true
				isComposition = true
			}
		}

		if !isComposition {
			continue
		}

		stepTypes := make([]string, 0, len(stepTypeSet))
		for st := range stepTypeSet {
			stepTypes = append(stepTypes, st)
		}
		sort.Strings(stepTypes)

		compositions = append(compositions, CompositionInfo{
			Name:         info.Name,
			Description:  info.Description,
			SubPipelines: subPipelines,
			StepTypes:    stepTypes,
		})
	}

	return compositions, nil
}
