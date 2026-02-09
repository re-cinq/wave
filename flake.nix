{
  description = "WAVE - Multi-agent pipeline orchestrator for AI-assisted development";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };

        # Common packages for all shells
        commonPackages = with pkgs; [
          go
          gh
          git
          jq
          curl
          sqlite
        ];
      in
      {
        devShells = {
          default = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              echo ""
              echo "  ╦ ╦╔═╗╦  ╦╔═╗"
              echo "  ║║║╠═╣╚╗╔╝║╣ "
              echo "  ╚╩╝╩ ╩ ╚╝ ╚═╝"
              echo "  Multi-Agent Pipeline Orchestrator"
              echo ""
              echo "  Wave coordinates multiple AI personas through structured pipelines,"
              echo "  enforcing permissions, contracts, and workspace isolation at every step."
              echo ""
              echo "  Essential Commands:"
              echo "    wave init              Initialize a new Wave project"
              echo "    wave run               Run a pipeline"
              echo "    wave do                Execute an ad-hoc task"
              echo "    wave list              List pipelines and personas"
              echo "    wave status            Show pipeline status"
              echo "    wave logs              Show pipeline logs"
              echo "    wave resume            Resume a paused pipeline"
              echo "    wave clean             Clean up project artifacts"
              echo "    wave validate          Validate Wave configuration"
              echo ""
              echo "  Run 'wave --help' for full command reference"
              echo ""

              # Export GH_TOKEN so Wave subprocesses can auth without keyring
              if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
                export GH_TOKEN=$(gh auth token 2>/dev/null)
              fi

              # Pre-flight auth checks
              if [ -z "$ANTHROPIC_API_KEY" ]; then
                if [ -f "$HOME/.claude/.credentials.json" ]; then
                  echo "  ⚠  ANTHROPIC_API_KEY not set — using Claude Code OAuth (may expire)"
                else
                  echo "  ✗  No Anthropic credentials found. Set ANTHROPIC_API_KEY or run: claude login"
                fi
              fi

              if [ -z "$GH_TOKEN" ]; then
                echo "  ⚠  GH_TOKEN not set — gh commands in pipelines may fail"
              fi
              echo ""
            '';
          };
        };
      }
    );
}
