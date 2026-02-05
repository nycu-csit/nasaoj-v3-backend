#!/usr/bin/env -S bash

set -o xtrace -o errexit -o pipefail

${COMPOSE[@]} exec backend /judge migrate init
${COMPOSE[@]} exec backend /judge migrate up

WORKER_TOKEN="$(${COMPOSE[@]} exec backend /judge admin gen token worker \
  --endpoint http://127.0.0.1:8080 --access-token poop | yq '.Token')"

sed -i -r "s/^ACCESS_TOKEN=.*$/ACCESS_TOKEN=$WORKER_TOKEN/" worker/.env

${COMPOSE[@]} up -d --wait --force-recreate worker
