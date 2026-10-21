#!/bin/bash

# Build the admin and user frontends from source and drop the artifacts into
# cmd/dashboard/{admin,user}-dist, which cmd/dashboard/main.go embeds.
#
# Unlike fetch-frontends.sh this does not download prebuilt releases: it builds
# the local checkouts, which carry the performance patches (route-level lazy
# loading, chunk splitting, no render-blocking CDN stylesheets).
#
# Expects the two repos to sit next to this one:
#   ../admin-frontend     (nezhahq/admin-frontend)
#   ../nezha-dash-v2      (hamster1963/nezha-dash-v2)
#
# The admin build uses npm (package-lock.json); the user frontend uses pnpm,
# pinned by its "packageManager" field and pnpm-lock.yaml.

set -euo pipefail

ROOT_DIR="$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")"
ADMIN_SRC="${ADMIN_SRC:-$ROOT_DIR/../admin-frontend}"
USER_SRC="${USER_SRC:-$ROOT_DIR/../nezha-dash-v2}"
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmmirror.com}"

require_dir() {
  if [ ! -d "$1" ]; then
    echo "error: $1 not found; clone the frontend repo next to the nezha checkout" >&2
    exit 1
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: $1 is required but not installed" >&2
    exit 1
  fi
}

install_dist() {
  local src="$1" target="$2"
  rm -rf "$target"
  cp -r "$src/dist" "$target"
  echo "installed $(basename "$target") from $src/dist"
}

require_dir "$ADMIN_SRC"
require_dir "$USER_SRC"
require_cmd node
require_cmd npm

echo "==> building admin frontend"
pushd "$ADMIN_SRC" >/dev/null
npm ci --no-audit --no-fund --registry="$NPM_REGISTRY"
npm run build
popd >/dev/null
install_dist "$ADMIN_SRC" "$ROOT_DIR/cmd/dashboard/admin-dist"

echo "==> building user frontend"
require_cmd pnpm
pushd "$USER_SRC" >/dev/null
pnpm install --frozen-lockfile --registry="$NPM_REGISTRY"
pnpm build
popd >/dev/null
install_dist "$USER_SRC" "$ROOT_DIR/cmd/dashboard/user-dist"

echo "done; rebuild the dashboard binary to embed the new dists"
