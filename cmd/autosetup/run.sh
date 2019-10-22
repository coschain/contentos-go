#!/bin/bash

nodeName=$1
chainid=$2

cd ../cosd/

GO=`which go`

if [[ "$chainid" == "mainnet" ]]
then
  $GO build
elif [[ "$chainid" == "testnet" ]]
then
  $GO build -tags testnet
else
  $GO build -tags devnet
fi

nohup ./cosd start -n $nodeName 1>/dev/null &
#nohup ./cosd start 2> /data/logs/coschain/cosd/trash_err.log 1>/dev/null &


