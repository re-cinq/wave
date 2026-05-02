#!/usr/bin/env bash
# Wave experiment VM entrypoint. Clones a target project via tea, runs
# wave init + onboard-project, then boots the webui.
#
# Required env:
#   WAVE_PROJECT_HOST   — e.g. codeberg.org
#   WAVE_PROJECT_OWNER  — e.g. libretech
#   WAVE_PROJECT_REPO   — e.g. wave-testing
#   WAVE_PROJECT_TOKEN  — token with read access (passes to tea)
#
# Optional env:
#   WAVE_ADAPTER        — claude (default) | opencode
#   WAVE_MODEL          — balanced (default) | opus | cheapest
#   WAVE_PORT           — 8080 (default)
#   WAVE_SKIP_ONBOARD   — set non-empty to skip onboard-project (manifest-only)
#
# Volume layout (compose mounts these):
#   /work               — project clone (target repo's working tree)
#   /home/wave/.agents  — Wave state + outputs (persists across container restarts)
#   /home/wave/.config  — claude-code auth + tea config

set -e

WAVE_PROJECT_HOST="${WAVE_PROJECT_HOST:-}"
WAVE_PROJECT_OWNER="${WAVE_PROJECT_OWNER:-}"
WAVE_PROJECT_REPO="${WAVE_PROJECT_REPO:-}"
WAVE_PROJECT_TOKEN="${WAVE_PROJECT_TOKEN:-}"
WAVE_ADAPTER="${WAVE_ADAPTER:-claude}"
WAVE_MODEL="${WAVE_MODEL:-balanced}"
WAVE_PORT="${WAVE_PORT:-8080}"

log() { printf '[entrypoint %s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }

require_env() {
  for var in WAVE_PROJECT_HOST WAVE_PROJECT_OWNER WAVE_PROJECT_REPO WAVE_PROJECT_TOKEN; do
    if [ -z "${!var}" ]; then
      log "missing required env: $var"
      exit 1
    fi
  done
}

# Idempotent: if /work/.git exists, skip clone — container restarts reuse the
# clone instead of fetching cold every time.
clone_project() {
  if [ -d /work/.git ]; then
    log "/work already a git tree; skipping clone"
    return 0
  fi
  log "configuring tea login for ${WAVE_PROJECT_HOST}"
  tea login add \
    --name wave-vm \
    --url "https://${WAVE_PROJECT_HOST}" \
    --token "${WAVE_PROJECT_TOKEN}" \
    >/dev/null

  log "cloning ${WAVE_PROJECT_OWNER}/${WAVE_PROJECT_REPO}"
  cd /work
  git clone \
    "https://oauth2:${WAVE_PROJECT_TOKEN}@${WAVE_PROJECT_HOST}/${WAVE_PROJECT_OWNER}/${WAVE_PROJECT_REPO}.git" \
    .

  # Embed the token in remote so wave/tea can push later. Replaced on each
  # boot so a rotated token takes effect without a fresh clone.
  git remote set-url origin \
    "https://oauth2:${WAVE_PROJECT_TOKEN}@${WAVE_PROJECT_HOST}/${WAVE_PROJECT_OWNER}/${WAVE_PROJECT_REPO}.git"
}

init_wave() {
  cd /work
  if [ -f wave.yaml ] && [ -d .agents ]; then
    log "wave.yaml + .agents already present; skipping wave init"
    return 0
  fi
  log "running wave init --adapter ${WAVE_ADAPTER}"
  wave init --adapter "${WAVE_ADAPTER}" >/dev/null
}

run_onboard_project() {
  if [ -n "${WAVE_SKIP_ONBOARD:-}" ]; then
    log "WAVE_SKIP_ONBOARD set; skipping onboard-project"
    return 0
  fi
  if [ -f /work/.agents/.onboarding-done ]; then
    log "onboarding sentinel present; skipping onboard-project"
    return 0
  fi
  log "running onboard-project (adapter=${WAVE_ADAPTER} model=${WAVE_MODEL})"
  cd /work
  wave run onboard-project \
    --adapter "${WAVE_ADAPTER}" \
    --model "${WAVE_MODEL}" \
    --auto-approve \
    --no-tui
}

boot_webui() {
  log "booting webui on :${WAVE_PORT}"
  cd /work
  exec wave serve --bind 0.0.0.0 --port "${WAVE_PORT}"
}

main() {
  require_env
  clone_project
  init_wave
  run_onboard_project
  boot_webui
}

main "$@"
