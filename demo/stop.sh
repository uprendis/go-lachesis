#!/bin/bash

PROG=benchopera

# kill all benchopera processes
killall "${PROG}"
sleep 3

# remove demo data
rm -rf /tmp/benchopera-demo/datadir/
