#!/bin/bash

PROG=network

# kill all network processes
pkill "${PROG}"

# remove demo data
sudo rm -rf /tmp/network-demo-replay/datadir
rm -rf exec*.sh dump.traffic
