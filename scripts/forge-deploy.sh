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

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

ENV_FILE=""
for candidate in "$ROOT/.env" "$ROOT/../.env"; do
  if [[ -f "$candidate" ]]; then
    ENV_FILE="$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")"
    break
  fi
done

if [[ -n "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ENV_FILE"
  set +a
fi

export NUXT_PORT="${NUXT_PORT:-3001}"
export PORT="$NUXT_PORT"

git pull origin "$FORGE_SITE_BRANCH"

if [[ "${DEPLOY_SCRIPT_REEXECED:-}" != "1" && -f "$ROOT/scripts/forge-deploy.sh" ]]; then
  export DEPLOY_SCRIPT_REEXECED=1
  exec bash "$ROOT/scripts/forge-deploy.sh"
fi

chmod +x scripts/run-api.sh

cd backend
go build -o server ./cmd/server
cd ..

cd frontend
npm ci
npm run build
pm2 delete massa-web 2>/dev/null || true
pm2 start ecosystem.config.cjs --update-env
pm2 save
cd ..

if [[ -n "${FORGE_API_DAEMON:-}" ]]; then
  if ! sudo supervisorctl restart "$FORGE_API_DAEMON"; then
    echo "Warning: could not restart $FORGE_API_DAEMON — check Server → Daemons for the supervisor name (e.g. daemon-1234567)" >&2
  fi
fi
