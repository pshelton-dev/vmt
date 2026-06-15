#!/usr/bin/env bash
# Build VMT for the remote's architecture and PUSH it to the registry.
# This is the local half of deployment — pulling/restarting on the Docker server
# is a separate manual step (see README → Deploying to a remote Docker host).
#
# Usage:
#   ./push.sh
#
# Requires:
#   - VMT_IMAGE set (here or in ./.env), e.g. ghcr.io/OWNER/vmt:latest
#   - `docker login ghcr.io` with a GitHub token (write:packages) for push access
#   - buildx locally (bundled with Docker Desktop)
#
# Environment overrides:
#   PLATFORM   build platform (default: linux/amd64)
set -euo pipefail

cd "$(dirname "$0")"

PLATFORM="${PLATFORM:-linux/amd64}"

# Resolve the image reference from the environment or ./.env.
VMT_IMAGE="${VMT_IMAGE:-$(grep -E '^VMT_IMAGE=' .env 2>/dev/null | cut -d= -f2- || true)}"

case "${VMT_IMAGE:-}" in
  "" | *OWNER* | *USERNAME*)
    echo "error: set VMT_IMAGE (env or .env), e.g. ghcr.io/yourname/vmt:latest" >&2
    exit 1
    ;;
esac

echo "==> Building and pushing $VMT_IMAGE for $PLATFORM"
echo "    (run 'docker login ghcr.io' first if this fails to push)"
# --provenance=false keeps it a plain single-arch image (not an OCI index).
docker buildx build --platform "$PLATFORM" --provenance=false -t "$VMT_IMAGE" --push .

echo "==> Pushed $VMT_IMAGE"
echo "    Now update the Docker server manually:"
echo "      cd ~/vmt && docker compose -f docker-compose.prod.yml pull \\"
echo "                && docker compose -f docker-compose.prod.yml up -d"
