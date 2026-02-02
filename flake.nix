{
  description = "NAVI - Cybernetic development environment";

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
          #          claude-code
          gh
          git
          jq
          curl
        ];
      in
      {
        # Packages for installation
        devShells = {
          # Default: Sandboxed shell - restricts filesystem access for Claude Code
         default = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              echo ""
              echo "=== WAVE Development Environment ==="
              echo ""
              echo "Start:     muz (daemon+TUI), muz serve, muz stop"
              echo ""
              echo "Type 'help' for full command list"
              echo ""
            '';
          };

        };
      }
    );
}
