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
              echo "  Commands:"
              echo "    wave init              Initialize new project"
              echo "    wave run --pipeline X  Run a pipeline"
              echo "    wave do \"task\"         Ad-hoc task execution"
              echo "    wave validate          Validate configuration"
              echo "    wave list pipelines    List available pipelines"
              echo "    wave list personas     List configured personas"
              echo "    wave resume            Resume interrupted run"
              echo "    wave clean             Clean up artifacts"
              echo ""
              echo "  Run 'wave --help' for full command reference"
              echo ""
            '';
          };
        };
      }
    );
}
