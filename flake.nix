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
          bubblewrap
        ];

        # Bubblewrap sandbox wrapper — isolates the entire dev session
        sandboxScript = pkgs.writeShellScriptBin "wave-sandbox" ''
          PROJECT_DIR="''${WAVE_PROJECT_DIR:-$PWD}"

          # Ensure bind targets exist before bwrap
          mkdir -p "$HOME/.claude"
          touch -a "$HOME/.claude.json"

          BWRAP_ARGS=(
            --unshare-all
            --share-net          # Full net for now; proxy filtering is future work
            --die-with-parent
            --new-session        # Prevent terminal escape sequence attacks

            # Root filesystem — READ-ONLY
            --ro-bind / /
            --dev /dev
            --proc /proc

            # Hide entire home directory
            --tmpfs "$HOME"

            # Writable: project directory only
            --bind "$PROJECT_DIR" "$PROJECT_DIR"

            # Writable: Claude Code config (session state, credentials)
            --bind "$HOME/.claude" "$HOME/.claude"
            --bind "$HOME/.claude.json" "$HOME/.claude.json"

            # Writable: isolated temp (NOT shared with host)
            --tmpfs /tmp

            # Read-only: git config for commits
            --ro-bind-try "$HOME/.gitconfig" "$HOME/.gitconfig"

            # Environment — curated, not inherited
            --clearenv
            --setenv HOME "$HOME"
            --setenv PATH "$PATH"
            --setenv TERM "''${TERM:-xterm-256color}"
            --setenv TMPDIR "/tmp"
            --setenv SANDBOX_ACTIVE "1"
            --setenv ANTHROPIC_API_KEY "''${ANTHROPIC_API_KEY:-}"
            --setenv GH_TOKEN "''${GH_TOKEN:-}"

            --chdir "$PROJECT_DIR"
          )

          if [ $# -gt 0 ]; then
            exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" "$@"
          else
            exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" \
              ${pkgs.bash}/bin/bash
          fi
        '';
      in
      {
        devShells = {
          # Default: sandboxed on Linux, unsandboxed on macOS (bwrap needs namespaces)
          default = pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript ];
            shellHook = ''
              echo ""
              echo "  ╦ ╦╔═╗╦  ╦╔═╗"
              echo "  ║║║╠═╣╚╗╔╝║╣ "
              echo "  ╚╩╝╩ ╩ ╚╝ ╚═╝"
              echo "  Multi-Agent Pipeline Orchestrator"
              echo ""

              export WAVE_PROJECT_DIR="$PWD"

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

              # Auto-enter bubblewrap sandbox on interactive Linux sessions
              if [ -t 0 ] && [ -z "$SANDBOX_ACTIVE" ] && [ "$(uname -s)" = "Linux" ]; then
                echo ""
                echo "  Entering bubblewrap sandbox..."
                echo "  WRITE: $PWD, ~/.claude, /tmp"
                echo "  READ:  / (read-only)"
                echo ""
                exec wave-sandbox bash
              fi

              echo ""
            '';
          };

          # Escape hatch: no sandbox (also used on macOS)
          yolo = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              echo ""
              echo "  ╦ ╦╔═╗╦  ╦╔═╗"
              echo "  ║║║╠═╣╚╗╔╝║╣ "
              echo "  ╚╩╝╩ ╩ ╚╝ ╚═╝"
              echo "  Multi-Agent Pipeline Orchestrator (NO SANDBOX)"
              echo ""

              # Export GH_TOKEN so Wave subprocesses can auth without keyring
              if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
                export GH_TOKEN=$(gh auth token 2>/dev/null)
              fi

              if [ -z "$ANTHROPIC_API_KEY" ]; then
                if [ -f "$HOME/.claude/.credentials.json" ]; then
                  echo "  ⚠  ANTHROPIC_API_KEY not set — using Claude Code OAuth (may expire)"
                else
                  echo "  ✗  No Anthropic credentials found. Set ANTHROPIC_API_KEY or run: claude login"
                fi
              fi

              echo ""
            '';
          };
        };
      }
    );
}
