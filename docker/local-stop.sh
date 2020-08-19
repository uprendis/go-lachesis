#!/usr/bin/env bash
cd $(dirname $0)

docker stop prometheus
killall tx-storm
killall benchopera
docker stop tracing
