#!/usr/bin/env bash
# Update VMT: pull latest code, rebuild with fresh base images, recreate the
# container, and prune the old image. Data in the vmt_data volume is preserved;
# schema migrations run automatically on startup.
set -euo pipefail

cd "$(dirname "$0")"

# Pick "docker compose" (v2) or fall back to "docker-compose" (v1).
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  DC="docker-compose"
else
  echo "error: docker compose is not installed" >&2
  exit 1
fi

echo "==> Reminder: back up first (Settings → Download backup) if this matters."

echo "==> Pulling latest code"
if [ -d .git ]; then
  git pull --ff-only
else
  echo "    (not a git checkout; deploy your updated files manually)"
fi

echo "==> Rebuilding app image with fresh base layers"
$DC build --pull

echo "==> Recreating containers"
$DC up -d

echo "==> Pruning dangling images"
docker image prune -f >/dev/null

echo "==> Status"
$DC ps
echo "==> Done. Tail logs with:  $DC logs -f app"
