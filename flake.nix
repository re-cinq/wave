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
          bun
          go
          deno
          claude-code
          gh
          tmux
          git
          jq
          curl
          bubblewrap  # Sandboxing tool for filesystem isolation
        ];

        # Base shell hook showing versions
        baseShellHook = ''
          export NAVI_ROOT="$PWD"
          export TMPDIR="/tmp"
          export GOTMPDIR="/tmp"
          echo "Bun $(bun -v) | Go $(go version | cut -d' ' -f3) | Deno $(deno -v | head -n1)"
        '';

        # Shell helper functions (shared between sandbox and yolo)
        shellFunctions = ''
          # Socket path (matches Makefile)
          export NAVI_SOCKET="$NAVI_ROOT/.navi-orchestrator.sock"
          export HEALTH_PORT=8089

          help() {
            echo ""
            echo "┌─────────────────────────────────────────────┐"
            echo "│           NAVI - Agent Orchestrator         │"
            echo "└─────────────────────────────────────────────┘"
            echo ""
            echo "TUI & Daemon:"
            echo "  tui            Start TUI (starts daemon if needed)"
            echo "  serve          Start daemon only (background)"
            echo "  stop           Stop all NAVI processes"
            echo "  restart        Restart daemon + TUI"
            echo "  status         Quick status overview"
            echo "  health         Check daemon health (JSON)"
            echo "  logs           Show recent activity"
            echo "  watch          Live tail of logs"
            echo ""
            echo "NAVI Commands:"
            echo "  navi           Start daemon + TUI (default)"
            echo "  navi serve     Start daemon only (background)"
            echo "  navi stop      Stop daemon"
            echo "  navi get tasks List tasks (-o json/yaml/wide)"
            echo "  navi get workers"
            echo "  navi scale workers <n>"
            echo "  navi create task -r <repo> -p <priority> \"prompt\""
            echo ""
            echo "Quick aliases:"
            echo "  tasks          List tasks"
            echo "  workers        List workers"
            echo "  repos          List repositories"
            echo "  scale <n>      Set max workers"
            echo "  apply <file>   Create tasks from manifest"
            echo ""
            echo "Claude:"
            echo "  claude resume  Resume Claude (skip permissions)"
            echo "  claude freerun Run new Claude Session (skip permissions)"
            echo "  claude spawn   Spawn Claude worker (sonnet)"
            echo "  claude <args>  Pass through to claude-code"
            echo ""
            echo "Development:"
            echo "  dev            Start development server"
            echo "  build          Build for production"
            echo "  test           Run tests"
            echo "  setup          Clean install dependencies"
            echo ""
          }

          # === TUI & Daemon Commands ===

          serve() {
            cd "$NAVI_ROOT" && make serve
          }

          tui() {
            cd "$NAVI_ROOT" && make navi
          }

          stop() {
            cd "$NAVI_ROOT" && make stop
          }

          restart() {
            stop
            sleep 1
            tui
          }

          status() {
            cd "$NAVI_ROOT" && make status
          }

          health() {
            curl -s "http://localhost:$HEALTH_PORT/health" 2>/dev/null | jq . || echo "Daemon not running"
          }

          logs() {
            cd "$NAVI_ROOT" && make logs
          }

          watch() {
            cd "$NAVI_ROOT" && make watch
          }

          # === Resource Commands (kubectl-style) ===

          navi() {
            # All commands route through the CLI entry point
            # The CLI uses $PWD as the project root, so it works from any directory
            if [ $# -eq 0 ]; then
              # Default: start daemon + TUI for current project
              deno run --allow-all "$NAVI_ROOT/orchestrator/cli/main.ts" run
            else
              deno run --allow-all "$NAVI_ROOT/orchestrator/cli/main.ts" "$@"
            fi
          }

          tasks() {
            local format="''${1:-}"
            if [ -n "$format" ]; then
              navi get tasks -o "$format"
            else
              navi get tasks
            fi
          }

          workers() {
            local format="''${1:-}"
            if [ -n "$format" ]; then
              navi get workers -o "$format"
            else
              navi get workers
            fi
          }

          repos() {
            local format="''${1:-}"
            if [ -n "$format" ]; then
              navi get repos -o "$format"
            else
              navi get repos
            fi
          }

          scale() {
            local count="''${1:-}"
            if [ -z "$count" ]; then
              echo "Usage: scale <count>"
              echo "Set max workers (1-100)"
              return 1
            fi
            navi scale workers "$count"
          }

          apply() {
            local file="''${1:-}"
            if [ -z "$file" ]; then
              echo "Usage: apply <manifest.yaml>"
              echo "Create tasks from YAML manifest"
              return 1
            fi
            navi apply -f "$file"
          }

          # === Claude Commands ===

          claude() {
            local cmd="''${1:-}"
            case "$cmd" in
              resume)
                command claude --dangerously-skip-permissions --resume
                ;;
              freerun)
                command claude --dangerously-skip-permissions
                ;;
              spawn)
                local model="''${2:-sonnet}"
                local workdir="$NAVI_ROOT/workspace/workers/$model-$(date +%s)"
                echo "Spawning Claude worker: $model"
                echo "Working directory: $workdir"
                mkdir -p "$workdir"
                command claude --model "$model" --working-directory "$workdir"
                ;;
              "")
                command claude
                ;;
              *)
                command claude "$@"
                ;;
            esac
          }

          # === Development Commands ===

          dev() {
            if [ -d "node_modules" ]; then
              bun run dev
            else
              echo "ERROR: node_modules not found. Run 'setup' first."
            fi
          }

          build() { bun run build; }
          test() { bun run test; }

          setup() {
            echo "Cleaning node_modules and lockfile..."
            rm -rf node_modules bun.lock
            echo "Installing dependencies..."
            bun install
          }
        '';
      in
      {
        # Packages for installation
        packages = {
          default = self.packages.${system}.navi;

          navi = pkgs.writeShellScriptBin "navi" ''
            exec ${pkgs.deno}/bin/deno run --allow-all ${self}/orchestrator/cli/main.ts "$@"
          '';

          navi-tui = pkgs.buildGoModule {
            pname = "navi-tui";
            version = "2.0.0";
            src = ./cmd/navi-tui;
            vendorHash = null; # Update after first build attempt
            meta.description = "NAVI Terminal UI";
          };
        };

        # Apps for `nix run`
        apps = {
          default = {
            type = "app";
            program = "${self.packages.${system}.navi}/bin/navi";
          };
        };

        devShells = {
          # Default: Sandboxed shell - restricts filesystem access for Claude Code
          default = let
            # Shell functions script (sourced inside sandbox)
            shellFunctionsScript = pkgs.writeText "shell-functions.sh" ''
              ${shellFunctions}
            '';

            sandboxScript = pkgs.writeShellScriptBin "enter-sandbox" ''
              # Build bwrap arguments dynamically
              PROJECT_DIR="''${SANDBOX_PROJECT_DIR:-$PWD}"

              BWRAP_ARGS=(
                --unshare-all
                --share-net
                --die-with-parent

                # Root filesystem - READ-ONLY
                --ro-bind / /

                # Device and proc filesystems
                --dev /dev
                --proc /proc

                # Hide home directory (tmpfs overlays the ro-bind)
                --tmpfs "$HOME"

                # Writable: project, claude config, bun cache, deno cache, go cache
                --bind "$PROJECT_DIR" "$PROJECT_DIR"
                --bind "$HOME/.claude" "$HOME/.claude"
                --bind "$HOME/.claude.json" "$HOME/.claude.json"
                --bind "$HOME/.bun" "$HOME/.bun"
                --bind "$HOME/.cache/deno" "$HOME/.cache/deno"
                --bind "$HOME/.cache/go-build" "$HOME/.cache/go-build"
                --bind "$HOME/go" "$HOME/go"

                # Read-only: git, ssh (for git operations)
                #--ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"
                --ro-bind "$HOME/.ssh" "$HOME/.ssh"

                # Temp - bind mount instead of tmpfs so daemon/workers can communicate
                # and Go/Deno can use shared tmp directory
                --bind /tmp /tmp

                # Set environment
                --setenv HOME "$HOME"
                --setenv PATH "$PATH"
                --setenv LD_LIBRARY_PATH "''${LD_LIBRARY_PATH:-}"
                --setenv TERM "''${TERM:-xterm}"
                --setenv TENV_AUTO_INSTALL "''${TENV_AUTO_INSTALL:-}"
                --setenv SANDBOX_ACTIVE "1"
                --setenv SHELL_FUNCTIONS "${shellFunctionsScript}"
                --chdir "$PROJECT_DIR"
              )

              # Ensure directories/files exist before binding
              mkdir -p "$HOME/.bun" "$HOME/.claude" "$HOME/.cache/deno" "$HOME/.cache/go-build" "$HOME/go"
              touch "$HOME/.claude.json"

              # If arguments passed, run them as command; otherwise interactive shell
              if [ $# -gt 0 ]; then
                exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" "$@"
              else
                exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" ${pkgs.bash}/bin/bash
              fi
            '';
          in pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript ];
            shellHook = ''
              export SANDBOX_PROJECT_DIR="$PWD"

              # Skip sandbox entry if running via --command (non-interactive)
              if [ ! -t 0 ] || [ -n "$NIX_DEVELOP_COMMAND" ]; then
                ${baseShellHook}
                echo ""
                echo "=== NAVI Environment (sandbox available via 'enter-sandbox') ==="
                echo ""
              else
                echo ""
                echo "=== NAVI SANDBOXED Environment ==="
                echo ""
                echo "Filesystem restrictions:"
                echo "  ✓ WRITE: $PWD, ~/.bun, ~/.claude, /tmp"
                echo "  ✓ READ-ONLY: everything else"
                echo ""
                echo "Starting sandboxed shell..."
                echo ""

                # Auto-enter the sandbox for interactive sessions
                export SHELL_FUNCTIONS="${shellFunctionsScript}"
                exec enter-sandbox ${pkgs.bash}/bin/bash --rcfile <(cat << SANDBOX_BASHRC
                ${baseShellHook}
                source "$SHELL_FUNCTIONS"
                echo ""
                echo "Start:    navi (daemon+TUI), navi serve, navi stop"
                echo "Query:    navi get tasks, navi get workers"
                echo "Claude:   claude resume, claude spawn"
                echo ""
                echo "Type 'help' for full list, 'exit' to leave sandbox"
                echo ""
                PS1="[navi] \w \$ "
SANDBOX_BASHRC
                )
              fi
            '';
          };

          # Yolo shell - full filesystem access (no sandbox)
          yolo = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              ${baseShellHook}
              ${shellFunctions}
              echo ""
              echo "=== NAVI Development Environment ==="
              echo ""
              echo "Start:     navi (daemon+TUI), navi serve, navi stop"
              echo "Query:     navi get tasks, navi get workers, tasks, workers"
              echo "Claude:    claude resume, claude spawn"
              echo "Dev:       dev, build, test, setup"
              echo ""
              echo "Type 'help' for full command list"
              echo ""
            '';
          };

        };
      }
    );
}
