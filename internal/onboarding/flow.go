package onboarding

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

// GreenfieldOpts captures the inputs needed for cold-start scaffolding.
type GreenfieldOpts struct {
	Adapter    string
	Workspace  string
	OutputPath string
	All        bool
	Stderr     io.Writer
}

// Greenfield runs the cold-start init flow: ensures the .agents directory tree
// exists, detects the project flavour and forge, builds and writes wave.yaml,
// copies embedded persona/pipeline/contract/prompt assets, seeds project
// instruction files, and creates an initial git commit if needed.
//
// Returns the loaded asset set (so callers can render a success banner) and
// the detected flavour info (for first-run suggestions).
func Greenfield(o GreenfieldOpts) (*AssetSet, *FlavourInfo, error) {
	if err := EnsureWaveDirs(DefaultWaveDirs); err != nil {
		return nil, nil, err
	}

	cwd, _ := os.Getwd()
	flavour := DetectFlavour(cwd)
	project := FlavourToProjectMap(flavour)

	assets, err := LoadAssets(o.Stderr, AssetOptions{All: o.All})
	if err != nil {
		return nil, nil, err
	}

	forgeInfo, _ := forge.DetectFromGitRemotes()
	assets.PersonaConfigs = FilterPersonasByForge(assets.PersonaConfigs, forgeInfo.Type)

	meta := ExtractProjectMetadata(cwd)

	manifestMap := BuildDefaultManifest(o.Adapter, o.Workspace, project, assets.PersonaConfigs)
	if metaMap, ok := manifestMap["metadata"].(map[string]interface{}); ok {
		if meta.Name != "" {
			metaMap["name"] = meta.Name
		}
		if meta.Description != "" {
			metaMap["description"] = meta.Description
		}
	}

	manifestData, err := yaml.Marshal(manifestMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	outputDir := filepath.Dir(o.OutputPath)
	if outputDir != "." && outputDir != "" {
		absOutputDir, _ := filepath.Abs(outputDir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create output directory %s: %w", absOutputDir, err)
		}
	}

	absOutputPath, _ := filepath.Abs(o.OutputPath)
	if err := os.WriteFile(o.OutputPath, manifestData, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write manifest to %s: %w", absOutputPath, err)
	}

	if err := CreateExamplePersonas(assets.Personas); err != nil {
		return nil, nil, fmt.Errorf("failed to create example personas in .agents/personas/: %w", err)
	}
	if err := CreateExamplePipelines(assets.Pipelines); err != nil {
		return nil, nil, fmt.Errorf("failed to create example pipelines in .agents/pipelines/: %w", err)
	}
	if err := CreateExampleContracts(assets.Contracts); err != nil {
		return nil, nil, fmt.Errorf("failed to create example contracts in .agents/contracts/: %w", err)
	}
	if err := CreateExamplePrompts(assets.Prompts); err != nil {
		return nil, nil, fmt.Errorf("failed to create example prompts in .agents/prompts/: %w", err)
	}
	if err := CreateProjectInstructionFiles(); err != nil {
		return nil, nil, fmt.Errorf("failed to create project instruction files: %w", err)
	}

	if err := CreateInitialCommit(o.Stderr, o.OutputPath); err != nil {
		return nil, nil, err
	}

	return assets, flavour, nil
}

// WizardOpts captures the inputs needed for the interactive wizard flow.
type WizardOpts struct {
	Adapter    string
	Workspace  string
	OutputPath string
	All        bool
	// Existing manifest (read by caller after confirming overwrite). May be nil.
	Existing *manifest.Manifest
	Stderr   io.Writer
}

// PrepareWizard ensures the .agents + .claude/commands directory tree exists,
// loads filtered assets, and seeds them on disk so the wizard's pipeline
// picker can discover them. Returns the loaded asset set.
func PrepareWizard(o WizardOpts) (*AssetSet, error) {
	if err := EnsureWaveDirs(WizardWaveDirs); err != nil {
		return nil, err
	}

	assets, err := LoadAssets(o.Stderr, AssetOptions{All: o.All})
	if err != nil {
		return nil, err
	}

	if err := CreateExamplePersonas(assets.Personas); err != nil {
		return nil, fmt.Errorf("failed to create example personas: %w", err)
	}
	if err := CreateExamplePipelines(assets.Pipelines); err != nil {
		return nil, fmt.Errorf("failed to create example pipelines: %w", err)
	}
	if err := CreateExampleContracts(assets.Contracts); err != nil {
		return nil, fmt.Errorf("failed to create example contracts: %w", err)
	}
	if err := CreateExamplePrompts(assets.Prompts); err != nil {
		return nil, fmt.Errorf("failed to create example prompts: %w", err)
	}

	return assets, nil
}
