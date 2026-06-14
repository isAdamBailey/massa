#!/usr/bin/env bash
set -euo pipefail

if [[ -f "$FORGE_SITE_PATH/scripts/forge-deploy.sh" ]]; then
  ROOT="$FORGE_SITE_PATH"
elif [[ -f "$FORGE_SITE_PATH/../scripts/forge-deploy.sh" ]]; then
  ROOT="$(cd "$FORGE_SITE_PATH/.." && pwd)"
else
  echo "Cannot find repo root from FORGE_SITE_PATH=$FORGE_SITE_PATH" >&2
  exit 1
fi

cd "$ROOT"

git pull origin "$FORGE_SITE_BRANCH"

chmod +x scripts/run-api.sh

cd backend
go build -o server ./cmd/server
cd ..

cd frontend
npm ci
npm run build
pm2 startOrReload ecosystem.config.cjs
cd ..

if [[ -n "${FORGE_API_DAEMON:-}" ]]; then
  sudo supervisorctl restart "$FORGE_API_DAEMON"
fi
