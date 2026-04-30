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
          golangci-lint  # CI lint parity (forbidigo, etc.); match v2.x in .github/workflows/lint.yml
          gh
          git
	  tea
	  glab
          jq
          curl
          sqlite
          bubblewrap
          nodejs_22  # Claude Code requires Node.js
          uv         # Python package manager for skill installation
          codex      # OpenAI Codex CLI (adapter target)
          gemini-cli # Google Gemini CLI (adapter target)
        ];

        claudeYoloScript = pkgs.writeShellScriptBin "claude-yolo" ''
          exec claude --dangerously-skip-permissions "$@"
        '';

        claudeResumeScript = pkgs.writeShellScriptBin "claude-resume" ''
          exec claude --dangerously-skip-permissions --resume "$@"
        '';

        # SWE-bench dataset fetcher — downloads from HuggingFace datasets viewer API.
        # Pure curl+jq, no Python/numpy needed — works inside bwrap sandbox.
        benchFetchScript = pkgs.writeShellScriptBin "wave-bench-fetch" ''
          set -euo pipefail
          DATASET="''${1:-princeton-nlp/SWE-bench_Lite}"
          SPLIT="''${2:-test}"
          OUTDIR=".wave/bench/datasets"
          mkdir -p "$OUTDIR"

          SLUG="$(echo "$DATASET" | tr '/' '_')"
          OUTFILE="$OUTDIR/$SLUG.jsonl"

          if [ -f "$OUTFILE" ]; then
            LINES=$(wc -l < "$OUTFILE")
            echo "Dataset already exists: $OUTFILE ($LINES tasks)"
            echo "Delete it first to re-download."
            exit 0
          fi

          echo "Fetching $DATASET (split=$SPLIT) via HuggingFace API..."

          # The datasets viewer API returns paginated JSON. Fetch all rows.
          API="https://datasets-server.huggingface.co"
          OFFSET=0
          LENGTH=100
          TOTAL=""
          TMPFILE=$(mktemp)
          trap 'rm -f "$TMPFILE"' EXIT

          while true; do
            RESPONSE=$(${pkgs.curl}/bin/curl -sS \
              "$API/rows?dataset=$DATASET&config=default&split=$SPLIT&offset=$OFFSET&length=$LENGTH")

            # Check for error
            ERROR=$(echo "$RESPONSE" | ${pkgs.jq}/bin/jq -r '.error // empty')
            if [ -n "$ERROR" ]; then
              echo "API error: $ERROR" >&2
              rm -f "$TMPFILE"
              exit 1
            fi

            # Extract rows and append as JSONL
            echo "$RESPONSE" | ${pkgs.jq}/bin/jq -c '.rows[].row' >> "$TMPFILE"

            # Get total on first request
            if [ -z "$TOTAL" ]; then
              TOTAL=$(echo "$RESPONSE" | ${pkgs.jq}/bin/jq -r '.num_rows_total')
              echo "Total rows: $TOTAL"
            fi

            # Count rows returned in this batch
            BATCH=$(echo "$RESPONSE" | ${pkgs.jq}/bin/jq '.rows | length')
            OFFSET=$((OFFSET + BATCH))

            echo "  fetched $OFFSET / $TOTAL"

            if [ "$BATCH" -lt "$LENGTH" ] || [ "$OFFSET" -ge "$TOTAL" ]; then
              break
            fi
          done

          mv "$TMPFILE" "$OUTFILE"
          LINES=$(wc -l < "$OUTFILE")
          echo "Wrote $LINES tasks to $OUTFILE"
        '';

        # Bubblewrap sandbox wrapper — isolates the entire dev session
        sandboxScript = pkgs.writeShellScriptBin "wave-sandbox" ''
          PROJECT_DIR="''${WAVE_PROJECT_DIR:-$PWD}"

          # Ensure bind targets exist before bwrap
          mkdir -p "$HOME/.claude"
          mkdir -p "$HOME/.local/bin"
          mkdir -p "$HOME/go"
          mkdir -p "$HOME/.local/share/uv"
          mkdir -p "$HOME/notes"
          touch -a "$HOME/.local/bin/wave"
          touch -a "$HOME/.local/bin/opencode-patched"
          touch -a "$HOME/.claude.json"

          BWRAP_ARGS=(
            --unshare-all
            --share-net          # Full net for now; proxy filtering is future work
            --die-with-parent

            # Root filesystem — READ-ONLY
            --ro-bind / /
            --dev /dev
            --proc /proc

            # Hide entire home directory
            --tmpfs "$HOME"

            # Writable: project directory
            --bind "$PROJECT_DIR" "$PROJECT_DIR"

            # Writable: Claude Code config (session state, credentials)
            --bind "$HOME/.claude" "$HOME/.claude"
            --bind "$HOME/.claude.json" "$HOME/.claude.json"

            # Writable: uv tool cache (persists installed skills like speckit)
            --bind "$HOME/.local/share/uv" "$HOME/.local/share/uv"

            # Writable: Go module cache (avoids re-downloading on every step)
            --bind "$HOME/go" "$HOME/go"

            # Writable: notesium notes directory
            --bind "$HOME/notes" "$HOME/notes"

            # Shared /tmp — Nix store and tooling needs it; still process-isolated via namespaces
            --bind /tmp /tmp

            # Read-only: git config for commits
            --ro-bind-try "$HOME/.gitconfig" "$HOME/.gitconfig"
            --ro-bind-try "$HOME/.config/git" "$HOME/.config/git"

            # Read-only: SSH keys for git push/pull
            --ro-bind-try "$HOME/.ssh" "$HOME/.ssh"

            # Read-only: gh CLI auth config
            --ro-bind-try "$HOME/.config/gh" "$HOME/.config/gh"
            --ro-bind-try "$HOME/.config/opencode" "$HOME/.config/opencode"

            # Read-only: NPM/Node config (Claude Code may need it)
            --ro-bind-try "$HOME/.npmrc" "$HOME/.npmrc"
            --ro-bind-try "$HOME/.config/nvm" "$HOME/.config/nvm"

            # Read-only: local tools
            --ro-bind-try "$HOME/.local/bin/notesium" "$HOME/.local/bin/notesium"
            --ro-bind-try "$HOME/.local/bin/claudit" "$HOME/.local/bin/claudit"
            --ro-bind-try "$HOME/.local/bin/opencode-patched" "$HOME/.local/bin/opencode-patched"

            # Writable: wave binary (go build target)
            --bind-try "$HOME/.local/bin/wave" "$HOME/.local/bin/wave"

            --setenv HOME "$HOME"
            --setenv PATH "$PATH"
            --setenv TERM "''${TERM:-xterm}"
            --setenv SANDBOX_ACTIVE 1
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
        packages = {
          wave = pkgs.buildGoModule {
            pname = "wave";
            version = "dev";
            src = ./.;
            # To update: run `nix build .#wave 2>&1` and replace with the hash from the error
            vendorHash = "sha256-kaDW1Ci9wgF09sVPSew6Zj6G4CEkhEcVdIYuadk64sk=";
            subPackages = [ "cmd/wave" ];
            ldflags = [
              "-s" "-w"
              "-X main.version=${self.shortRev or "dev"}"
              "-X main.commit=${self.shortRev or "none"}"
              "-X main.date=1970-01-01T00:00:00Z"
            ];
            meta = with pkgs.lib; {
              description = "Multi-agent pipeline orchestrator";
              homepage = "https://github.com/re-cinq/wave";
              license = licenses.asl20;
              mainProgram = "wave";
            };
          };
          default = self.packages.${system}.wave;
        };

        devShells = {
          # Default: sandboxed on Linux, unsandboxed on macOS (bwrap needs namespaces)
          default = pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript benchFetchScript claudeYoloScript claudeResumeScript ];
            shellHook = ''
              echo ""
              echo "  ╦ ╦╔═╗╦  ╦╔═╗"
              echo "  ║║║╠═╣╚╗╔╝║╣ "
              echo "  ╚╩╝╩ ╩ ╚╝ ╚═╝"
              echo "  Multi-Agent Pipeline Orchestrator"
              echo ""

              export WAVE_PROJECT_DIR="$PWD"

              # Skip system SSH config that pulls in broken Nix store systemd drop-in
              export GIT_SSH_COMMAND="ssh -F ~/.ssh/config"

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
                echo ""
                echo "  Writable:   $PWD"
                echo "              ~/.claude, ~/.claude.json"
                echo "              ~/.local/bin/wave, ~/go"
                echo "              ~/notes, /tmp"
                echo ""
                echo "  Read-only:  / (entire root)"
                echo "              ~/.ssh, ~/.gitconfig, ~/.config/git"
                echo "              ~/.config/gh, ~/.config/opencode, ~/.npmrc, ~/.config/nvm"
                echo "              ~/.local/bin/{notesium,claudit,opencode-patched}"
                echo ""
                exec wave-sandbox ${pkgs.bash}/bin/bash --rcfile <(cat << 'WAVE_BASHRC'
                  PS1="[sandbox] \w \$ "
WAVE_BASHRC
                )
              fi

              echo ""
            '';
          };

          # Escape hatch: no sandbox (also used on macOS)
          yolo = pkgs.mkShell {
            buildInputs = commonPackages ++ [ benchFetchScript ];
            shellHook = ''
              echo ""
              echo "  ╦ ╦╔═╗╦  ╦╔═╗"
              echo "  ║║║╠═╣╚╗╔╝║╣ "
              echo "  ╚╩╝╩ ╩ ╚╝ ╚═╝"
              echo "  Multi-Agent Pipeline Orchestrator (NO SANDBOX)"
              echo ""

              # Skip system SSH config that pulls in broken Nix store systemd drop-in
              export GIT_SSH_COMMAND="ssh -F ~/.ssh/config"

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
