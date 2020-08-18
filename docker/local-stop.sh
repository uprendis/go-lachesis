#!/usr/bin/env bash
cd $(dirname $0)

docker stop prometheus
killall tx-storm
killall network
docker stop tracing
