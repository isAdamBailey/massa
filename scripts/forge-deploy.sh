#!/usr/bin/env bash
set -euo pipefail

# Forge zero-downtime: this runs inside the new release, before
# $ACTIVATE_RELEASE(). $PWD is that release, not `current`.
ROOT="$PWD"

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

# Site-root .env lives two levels above releases/<id>. The other paths
# cover a symlink into the release or the old web-directory layout.
ENV_CANDIDATES=("$ROOT/../../.env" "$ROOT/frontend/.env" "$ROOT/.env")
ENV_FILE=""
for candidate in "${ENV_CANDIDATES[@]}"; do
  if [[ -f "$candidate" ]]; then
    ENV_FILE="$candidate"
    break
  fi
done

if [[ -z "$ENV_FILE" ]]; then
  echo "ERROR: no .env found at any of: ${ENV_CANDIDATES[*]}" >&2
  echo "Contents of $ROOT:" >&2
  ls -la "$ROOT" >&2
  exit 1
fi

echo "Using env file: $ENV_FILE"
set -a
# shellcheck disable=SC1091
source "$ENV_FILE"
set +a

exec 9>/tmp/node-build.lock
flock -x 9
echo "Acquired /tmp/node-build.lock"

export GOFLAGS="-p=2"

cd "$ROOT/backend"
go build -o server ./cmd/server
go run ./cmd/migrate up
cd "$ROOT"

cd "$ROOT/frontend"
npm ci
npm run generate
cd "$ROOT"

chmod +x scripts/run-api.sh
