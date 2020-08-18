#!/bin/bash

PROG=network

# kill all network processes
pkill "${PROG}"

# remove demo data
rm -rf /tmp/network-demo/datadir/
