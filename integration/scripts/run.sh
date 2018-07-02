#!/bin/bash

DOCKERLOCATION="lawrencegripper"

echo "Starting cluster - Current directory: ${PWD}"

echo "!WARNING: Containerized clusters require IPV6 enabled. Without updating your docker settings this will fail"
echo "see https://docs.microsoft.com/en-us/azure/service-fabric/service-fabric-get-started-mac for details"


echo "######## Remove previous containers if they exist ###########"
docker rm -f sftestcluster 
docker rm -f sfsampleinstaller
docker rm -f sfappinstaller

echo "######## Starting onebox cluster docker container ###########"
docker run --name sftestcluster -d -p 19080:19080 -p 19000:19000 -p 25100-25200:25100-25200 -p 8080:8080 -p 8081:8081 $DOCKERLOCATION/sfoneboxwithnode 

bash -f ./wait_for_healthy.sh

echo "######## Deploying sample node apps to cluster ###########"
if [ ! -f "./upload_test_apps.sh" ]
then
	echo "Cannot find '${PWD}/upload_test_apps.sh' script must run under '/integration/scripts' folder."
    exit 1
fi
docker run --name sfappinstaller -d --network=host -v ${PWD}/../:/src $DOCKERLOCATION/sfctl -f ./scripts/upload_test_apps.sh

# Note: Previously attemted to use 'docker commit' on sftestcluster to capture the state so apps didn't need to be installed each time 
# however, this caused an issue with the SF cluster so have worked around this by installing apps each time.  

# This is required as -it fails when invoked from golang
# TODO: Investigate workarounds
docker wait sfappinstaller
docker logs sfappinstaller
docker rm -f sfappinstaller

bash -f ./wait_for_healthy.sh