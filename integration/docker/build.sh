#!/bin/bash

cd "$(dirname "$0")"
DOCKERLOCATION="lawrencegripper"
DOCKERVERSION="0.2"

echo "######## Building docker images ###########"
docker build -t $DOCKERLOCATION/sfonebox:$DOCKERVERSION -t $DOCKERLOCATION/sfonebox:latest -f ./cluster.Dockerfile .
docker build -t $DOCKERLOCATION/sfoneboxwithnode:$DOCKERVERSION -t $DOCKERLOCATION/sfoneboxwithnode:latest -f ./clusterwithnode.Dockerfile .
docker build -t $DOCKERLOCATION/sfctl:$DOCKERVERSION -t $DOCKERLOCATION/sfctl:latest -f ./sfctl.Dockerfile .



