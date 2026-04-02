#!/usr/bin/env bash
set -Eeuo pipefail

# Sub2API source build + deploy helper
# Commands:
#   upgrade   Pull latest code, build, backup current runtime, replace binary, restart with health check
#   rollback  Restore from backup id (or latest)
#   list      List backups

REPO_URL="${REPO_URL:-https://github.com/tanaer/sub2api}"
SOURCE_DIR="${SOURCE_DIR:-/root/sub2api}"
APP_DIR="${APP_DIR:-/opt/sub2api}"
BIN_PATH="${BIN_PATH:-/opt/sub2api/sub2api}"
SERVICE_NAME="${SERVICE_NAME:-sub2api}"
ENV_FILE="${ENV_FILE:-/etc/sub2api/sub2api.env}"
SERVICE_FILE="${SERVICE_FILE:-/etc/systemd/system/sub2api.service}"
BACKUP_ROOT="${BACKUP_ROOT:-/opt/sub2api/backups}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
HEALTH_TIMEOUT_SEC="${HEALTH_TIMEOUT_SEC:-90}"

log() { echo "[$(date '+%F %T')] $*" >&2; }
err() { echo "[$(date '+%F %T')] ERROR: $*" >&2; }

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    err "Please run as root"
    exit 1
  fi
}

have_cmd() { command -v "$1" >/dev/null 2>&1; }

semver_ge() {
  # returns 0 when $1 >= $2
  local a b IFS=.
  read -r -a a <<<"${1}"
  read -r -a b <<<"${2}"
  for i in 0 1 2; do
    local av="${a[$i]:-0}"
    local bv="${b[$i]:-0}"
    if ((10#${av} > 10#${bv})); then
      return 0
    fi
    if ((10#${av} < 10#${bv})); then
      return 1
    fi
  done
  return 0
}

install_deps_if_needed() {
  local missing=()
  for cmd in git curl make npm; do
    have_cmd "$cmd" || missing+=("$cmd")
  done

  if ! have_cmd go; then
    missing+=("go")
  fi

  if ((${#missing[@]} == 0)) && have_cmd pnpm; then
    log "Dependencies are already installed"
    return
  fi

  log "Installing missing dependencies: ${missing[*]:-pnpm}"

  if have_cmd apt-get; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -y
    apt-get install -y git curl make golang-go ca-certificates
  elif have_cmd yum; then
    yum install -y git curl make golang
  elif have_cmd dnf; then
    dnf install -y git curl make golang
  else
    err "Unsupported package manager. Install git/curl/make/go/npm manually."
    exit 1
  fi

  if ! have_cmd pnpm; then
    npm install -g pnpm
  fi

  if have_cmd corepack; then
    corepack enable || true
  fi
}

ensure_go_toolchain() {
  local required current go_bin dl_url tmp_tgz

  required="$(awk '/^go[[:space:]]+/ {print $2; exit}' "$SOURCE_DIR/backend/go.mod")"
  if [[ -z "$required" ]]; then
    err "Unable to read required Go version from $SOURCE_DIR/backend/go.mod"
    exit 1
  fi

  go_bin="$(command -v go || true)"
  if [[ -n "$go_bin" ]]; then
    current="$("$go_bin" version | awk '{print $3}' | sed 's/^go//')"
  else
    current="0.0.0"
  fi

  if [[ -n "$go_bin" ]] && semver_ge "$current" "$required"; then
    log "Go toolchain OK: current=$current required=$required"
    return
  fi

  log "Installing Go toolchain $required (current=${current:-none})"
  dl_url="https://go.dev/dl/go${required}.linux-amd64.tar.gz"
  tmp_tgz="$(mktemp /tmp/go-toolchain.XXXXXX.tgz)"
  if ! curl -fL "$dl_url" -o "$tmp_tgz"; then
    dl_url="https://dl.google.com/go/go${required}.linux-amd64.tar.gz"
    curl -fL "$dl_url" -o "$tmp_tgz"
  fi

  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$tmp_tgz"
  rm -f "$tmp_tgz"

  export PATH="/usr/local/go/bin:$PATH"
  current="$(go version | awk '{print $3}' | sed 's/^go//')"
  if ! semver_ge "$current" "$required"; then
    err "Installed Go version $current is still below required $required"
    exit 1
  fi
  log "Go upgraded to $current"
}

ensure_source_repo() {
  if [[ ! -d "$SOURCE_DIR/.git" ]]; then
    err "SOURCE_DIR is not a git repository: $SOURCE_DIR"
    err "Please put the repo at this path first."
    exit 1
  fi

  local remote_url
  remote_url="$(git -C "$SOURCE_DIR" remote get-url origin 2>/dev/null || true)"
  if [[ -n "$remote_url" && "$remote_url" != "$REPO_URL" ]]; then
    log "Warning: origin remote is '$remote_url' (expected '$REPO_URL')"
  fi

  log "Updating source with git pull --ff-only"
  git -C "$SOURCE_DIR" pull --ff-only
}

build_new_binary() {
  local build_dir="$SOURCE_DIR/.build"
  local version commit date

  rm -rf "$build_dir"
  mkdir -p "$build_dir"

  log "Installing frontend dependencies"
  pnpm --dir "$SOURCE_DIR/frontend" install --frozen-lockfile >&2

  log "Building frontend"
  pnpm --dir "$SOURCE_DIR/frontend" run build >&2

  if [[ ! -f "$SOURCE_DIR/backend/internal/web/dist/index.html" ]]; then
    err "Frontend build output missing at $SOURCE_DIR/backend/internal/web/dist/index.html"
    exit 1
  fi

  version="$(tr -d '\r\n' < "$SOURCE_DIR/backend/cmd/server/VERSION")"
  commit="$(git -C "$SOURCE_DIR" rev-parse --short HEAD)"
  date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  log "Building backend binary (embed frontend)"
  (
    cd "$SOURCE_DIR/backend"
    export GOTOOLCHAIN=auto
    CGO_ENABLED=0 go build \
      -tags=embed \
      -ldflags="-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.Date=${date} -X main.BuildType=source" \
      -trimpath \
      -o "$build_dir/sub2api.new" \
      ./cmd/server >&2
  )

  "$build_dir/sub2api.new" --version >/dev/null
  echo "$build_dir/sub2api.new"
}

create_backup() {
  local backup_id backup_dir
  backup_id="$(date +%Y%m%d-%H%M%S)"
  backup_dir="$BACKUP_ROOT/$backup_id"

  mkdir -p "$backup_dir"

  [[ -f "$BIN_PATH" ]] && cp -a "$BIN_PATH" "$backup_dir/sub2api.bin"
  [[ -f "$ENV_FILE" ]] && cp -a "$ENV_FILE" "$backup_dir/sub2api.env"
  [[ -f "$SERVICE_FILE" ]] && cp -a "$SERVICE_FILE" "$backup_dir/sub2api.service"
  [[ -f "$APP_DIR/data/config.yaml" ]] && cp -a "$APP_DIR/data/config.yaml" "$backup_dir/config.yaml"

  cat > "$backup_dir/meta.txt" <<META
backup_id=$backup_id
created_at=$(date -Is)
host=$(hostname)
service=$SERVICE_NAME
bin_path=$BIN_PATH
source_dir=$SOURCE_DIR
META

  ln -sfn "$backup_dir" "$BACKUP_ROOT/latest"
  echo "$backup_id"
}

wait_health() {
  local deadline now
  deadline=$(( $(date +%s) + HEALTH_TIMEOUT_SEC ))

  while true; do
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
      return 0
    fi
    now=$(date +%s)
    if (( now >= deadline )); then
      return 1
    fi
    sleep 1
  done
}

restart_service() {
  log "Restarting service: $SERVICE_NAME"
  systemctl restart "$SERVICE_NAME"
}

do_rollback() {
  local backup_id="${1:-latest}"
  local backup_dir

  if [[ "$backup_id" == "latest" ]]; then
    backup_dir="$BACKUP_ROOT/latest"
  else
    backup_dir="$BACKUP_ROOT/$backup_id"
  fi

  if [[ ! -e "$backup_dir" ]]; then
    err "Backup not found: $backup_dir"
    exit 1
  fi

  backup_dir="$(readlink -f "$backup_dir")"
  log "Restoring backup: $(basename "$backup_dir")"

  [[ -f "$backup_dir/sub2api.bin" ]] || { err "Backup binary missing"; exit 1; }

  install -m 0755 "$backup_dir/sub2api.bin" "$BIN_PATH"

  if [[ -f "$backup_dir/sub2api.env" ]]; then
    cp -a "$backup_dir/sub2api.env" "$ENV_FILE"
    chmod 600 "$ENV_FILE" || true
  fi

  if [[ -f "$backup_dir/sub2api.service" ]]; then
    cp -a "$backup_dir/sub2api.service" "$SERVICE_FILE"
    systemctl daemon-reload
  fi

  if [[ -f "$backup_dir/config.yaml" ]]; then
    mkdir -p "$APP_DIR/data"
    cp -a "$backup_dir/config.yaml" "$APP_DIR/data/config.yaml"
  fi

  restart_service
  if wait_health; then
    log "Rollback succeeded and health check passed"
  else
    err "Rollback completed, but health check failed"
    systemctl --no-pager --full status "$SERVICE_NAME" || true
    exit 1
  fi
}

do_upgrade() {
  require_root
  install_deps_if_needed
  ensure_source_repo
  ensure_go_toolchain

  local new_bin backup_id
  new_bin="$(build_new_binary)"

  backup_id="$(create_backup)"
  log "Backup created: $backup_id"

  install -m 0755 "$new_bin" "$BIN_PATH"

  restart_service

  if wait_health; then
    log "Upgrade succeeded (backup_id=$backup_id)"
    log "Current version: $($BIN_PATH --version 2>&1 | tail -n1)"
  else
    err "Health check failed after upgrade, rolling back automatically"
    do_rollback "$backup_id"
    exit 1
  fi
}

list_backups() {
  mkdir -p "$BACKUP_ROOT"
  ls -1 "$BACKUP_ROOT" 2>/dev/null | grep -E '^[0-9]{8}-[0-9]{6}$' | sort -r || true
}

usage() {
  cat <<USAGE
Usage:
  $0 upgrade              Pull latest code, build, backup, deploy, restart, health-check
  $0 rollback [backup_id] Restore a backup (default: latest)
  $0 list                 List all backups

Optional env vars:
  REPO_URL, SOURCE_DIR, APP_DIR, BIN_PATH, SERVICE_NAME,
  ENV_FILE, SERVICE_FILE, BACKUP_ROOT, HEALTH_URL, HEALTH_TIMEOUT_SEC
USAGE
}

main() {
  local cmd="${1:-upgrade}"
  case "$cmd" in
    upgrade)
      do_upgrade
      ;;
    rollback)
      require_root
      do_rollback "${2:-latest}"
      ;;
    list)
      list_backups
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      err "Unknown command: $cmd"
      usage
      exit 1
      ;;
  esac
}

main "$@"
