#!/bin/bash

DOCKERLOCATION="lawrencegripper"

echo "######## Building docker images ###########"
docker build -t $DOCKERLOCATION/sfonebox -f ./cluster.Dockerfile .
docker build -t $DOCKERLOCATION/sfoneboxwithnode -f ./clusterwithnode.Dockerfile .
docker build -t $DOCKERLOCATION/sfctl -f ./sfctl.Dockerfile .
