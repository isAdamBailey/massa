#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

ENV_FILE=""
for candidate in "$ROOT/.env" "$ROOT/../.env"; do
  if [[ -f "$candidate" ]]; then
    ENV_FILE="$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")"
    break
  fi
done

if [[ -z "$ENV_FILE" ]]; then
  echo "No .env found (checked $ROOT/.env and $ROOT/../.env)" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
source "$ENV_FILE"
set +a

cd "$ROOT/backend"
exec ./server
