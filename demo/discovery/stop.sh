#!/bin/bash

# kill all bootnode and network processes
pkill "bootnode"
pkill "lachesis"

# remove demo data
#rm -rf /tmp/network-demo/datadir/
