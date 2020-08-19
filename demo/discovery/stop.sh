#!/bin/bash

# kill all bootnode and benchopera processes
pkill "bootnode"
pkill "benchopera"

# remove demo data
#rm -rf /tmp/benchopera-demo/datadir/
