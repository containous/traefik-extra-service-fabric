#!/bin/bash
echo "######## Building docker images ###########"
docker build -t lawrencegripper/sfonebox -f ./cluster.Dockerfile .
docker build -t lawrencegripper/sfoneboxwithnode -f ./clusterwithnode.Dockerfile .
docker build --network=host -t lawrencegripper/sfctl -f ./sfctl.Dockerfile .

./test-lite.sh