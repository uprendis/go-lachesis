#!/usr/bin/env bash
cd $(dirname $0)
. ./_params.sh


docker $SWARM service create \
  --benchopera benchopera \
  --name tracing \
  --publish 16686:16686 \
  --replicas 1 \
  --detach=false \
  jaegertracing/all-in-one
