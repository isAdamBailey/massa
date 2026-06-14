#!/usr/bin/env bash
set -euo pipefail

cd "$FORGE_SITE_PATH"

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
