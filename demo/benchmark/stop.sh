#!/bin/bash

PROG=benchopera

# kill all benchopera processes
pkill "${PROG}"

# remove demo data
sudo rm -rf /tmp/benchopera-demo-replay/datadir
rm -rf exec*.sh dump.traffic
