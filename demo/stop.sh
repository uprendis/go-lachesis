#!/bin/bash

PROG=benchopera

# kill all benchopera processes
pkill "${PROG}"

# remove demo data
rm -rf /tmp/benchopera-demo/datadir/
