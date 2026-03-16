package commands

import (
	"testing"
)

func TestNewChatCmd(t *testing.T) {
	cmd := NewChatCmd()

	if cmd.Use != "chat [run-id]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	// Verify flags exist
	flags := []string{"step", "artifact", "manifest", "model", "prompt", "list", "resume"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}
}

func TestNewChatCmd_FlagDefaults(t *testing.T) {
	cmd := NewChatCmd()

	manifest, _ := cmd.Flags().GetString("manifest")
	if manifest != "wave.yaml" {
		t.Errorf("expected default manifest 'wave.yaml', got %q", manifest)
	}

	model, _ := cmd.Flags().GetString("model")
	if model != "" {
		t.Errorf("expected default model '', got %q", model)
	}

	list, _ := cmd.Flags().GetBool("list")
	if list {
		t.Error("expected default list=false")
	}

	step, _ := cmd.Flags().GetString("step")
	if step != "" {
		t.Errorf("expected default step '', got %q", step)
	}

	artifact, _ := cmd.Flags().GetString("artifact")
	if artifact != "" {
		t.Errorf("expected default artifact '', got %q", artifact)
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	if prompt != "" {
		t.Errorf("expected default prompt '', got %q", prompt)
	}
}

func TestNewChatCmd_AcceptsRunIDArg(t *testing.T) {
	cmd := NewChatCmd()

	// Should accept 0 or 1 arg
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("should accept 0 args: %v", err)
	}
	if err := cmd.Args(cmd, []string{"run-id"}); err != nil {
		t.Errorf("should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{"run-id", "extra"}); err == nil {
		t.Error("should reject 2 args")
	}
}

func TestChatOptions_Defaults(t *testing.T) {
	opts := ChatOptions{
		Manifest: "wave.yaml",
	}

	if opts.RunID != "" {
		t.Errorf("expected empty RunID, got %q", opts.RunID)
	}
	if opts.List {
		t.Error("expected List=false")
	}
	if opts.Model != "" {
		t.Errorf("expected empty Model, got %q", opts.Model)
	}
	if opts.Artifact != "" {
		t.Errorf("expected empty Artifact, got %q", opts.Artifact)
	}
	if opts.Prompt != "" {
		t.Errorf("expected empty Prompt, got %q", opts.Prompt)
	}
	if opts.Resume != "" {
		t.Errorf("expected empty Resume, got %q", opts.Resume)
	}
}

func TestNewChatCmd_ResumeFlag(t *testing.T) {
	cmd := NewChatCmd()

	resumeFlag := cmd.Flags().Lookup("resume")
	if resumeFlag == nil {
		t.Fatal("missing --resume flag")
	}
	if resumeFlag.DefValue != "" {
		t.Errorf("expected empty default for resume, got %q", resumeFlag.DefValue)
	}
}

func TestNewChatCmd_ResumeFlagDescription(t *testing.T) {
	cmd := NewChatCmd()

	resumeFlag := cmd.Flags().Lookup("resume")
	if resumeFlag == nil {
		t.Fatal("missing --resume flag")
	}
	if resumeFlag.Usage == "" {
		t.Error("resume flag should have a usage description")
	}
}
