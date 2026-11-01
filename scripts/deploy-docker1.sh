#!/bin/bash
# One-shot production switch for docker1: back up, stop, rename, start the
# new image, wait for health, and roll back by itself if the new one does
# not come up.
#
# Why one script: the caller's own model route may go through this very
# gateway. Stopping magpie cuts that route, so the switch must finish
# without the caller. Run it detached and poll the log:
#
#   ssh docker1 'nohup bash /tmp/deploy-docker1.sh magpie:0.1.2450-p1 >/tmp/deploy.log 2>&1 &'
#   ssh docker1 'tail -f /tmp/deploy.log'
#
# Usage: deploy-docker1.sh <image> [tag-for-backup]
set -uo pipefail

NEW=${1:?usage: deploy-docker1.sh <image> [tag]}
TAG=${2:-${NEW#magpie:}}
TS=$(date +%Y%m%d-%H%M%S)
B=/home/hermes/backups/magpie/pre-${TAG}-${TS}
OLD=magpie-pre-${TAG}-${TS}
VOL=/var/lib/docker/volumes/magpie-config/_data

log() { echo "[$(date -u +%H:%M:%S)] $*"; }
die() { log "FAILED: $*"; exit 1; }

log "new image: $NEW  backup: $B"
docker image inspect "$NEW" >/dev/null 2>&1 || die "image $NEW is not on this host (transfer it first)"

# the old container's env is the source of truth for the key and public URL
WK=$(docker inspect magpie --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MAGPIE_WEB_KEY=//p')
PUB=$(docker inspect magpie --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MAGPIE_PUBLIC_URL=//p')
[ -n "$WK" ] || die "MAGPIE_WEB_KEY not found in the running container"
log "env captured (key ${#WK} chars, public url ${#PUB} chars)"

mkdir -p "$B" || die "cannot create $B"
docker inspect magpie > "$B/container.json" || die "container inspect failed"
log "container.json written"

docker stop magpie >/dev/null || die "docker stop magpie failed"
log "old container stopped"

sudo tar -C "$VOL" -czf "$B/volume.tgz" . || { log "volume tar failed; starting the old container back"; docker start magpie; exit 1; }
sudo chown "$(id -u):$(id -g)" "$B" -R 2>/dev/null
log "volume backed up ($(du -h "$B/volume.tgz" | cut -f1))"

docker rename magpie "$OLD" || { log "rename failed; starting the old container back"; docker start magpie; exit 1; }
log "old container renamed to $OLD"

docker run -d --name magpie --restart unless-stopped --user nonroot \
  -p 3425:3425 -p 3430:3430 -v magpie-config:/config:z \
  -e MAGPIE_WEB_KEY="$WK" -e MAGPIE_PUBLIC_URL="$PUB" \
  "$NEW" web --addr 0.0.0.0:3430 --no-open >/dev/null || {
    log "docker run failed; rolling back"
    docker rm -f magpie >/dev/null 2>&1
    docker rename "$OLD" magpie && docker start magpie
    die "rolled back to the previous container"
  }
log "new container started"

# wait for healthy (up to 180s); anything else is a rollback
for i in $(seq 1 60); do
  st=$(docker inspect magpie --format '{{.State.Health.Status}}' 2>/dev/null || echo none)
  [ "$st" = healthy ] && { log "healthy after ${i}0s"; break; }
  if ! docker ps --format '{{.Names}}' | grep -qx magpie; then
    log "new container exited; rolling back"
    docker rm -f magpie >/dev/null 2>&1
    docker rename "$OLD" magpie && docker start magpie
    die "rolled back to the previous container"
  fi
  sleep 3
done
[ "$(docker inspect magpie --format '{{.State.Health.Status}}')" = healthy ] || {
  log "not healthy in 180s; rolling back"
  docker stop magpie >/dev/null 2>&1; docker rm magpie >/dev/null 2>&1
  docker rename "$OLD" magpie && docker start magpie
  die "rolled back to the previous container"
}

log "DONE: magpie now runs $NEW; previous container kept as $OLD (stopped)"
log "backup: $B"
