#!/bin/bash

DOCKERLOCATION="lawrencegripper"

echo "Resettting cluster - Current directory: ${PWD}"

bash -f ./wait_for_healthy.sh

echo "######## Reset node apps to cluster ###########"
if [ ! -f "./reset_test_apps.sh" ]
then
	echo "Cannot find './reset_test_apps.sh' script must run under '/integration' folder."
    exit 1
fi
docker run --name sfappinstaller -d --network=host -v ${PWD}/../:/src $DOCKERLOCATION/sfctl -f ./scripts/reset_test_apps.sh

# Note: Previously attemted to use 'docker commit' on sftestcluster to capture the state so apps didn't need to be installed each time 
# however, this caused an issue with the SF cluster so have worked around this by installing apps each time.  

# This is required as -it fails when invoked from golang
# TODO: Investigate workarounds
docker wait sfappinstaller
docker logs sfappinstaller
docker rm -f sfappinstaller

bash -f ./wait_for_healthy.sh