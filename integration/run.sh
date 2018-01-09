#!/bin/bash

DOCKERLOCATION="lawrencegripper"

echo "Current directory: ${PWD}"

echo "!WARNING: Containerized clusters require IPV6 enabled. Without updating your docker settings this will fail"
echo "see https://docs.microsoft.com/en-us/azure/service-fabric/service-fabric-get-started-mac for details"

function isClusterHealthy () {
    echo "Checking cluster status..."
    HEALTHURL="http://localhost:19080/$/GetClusterHealth?NodesHealthStateFilter=1&ApplicationsHealthStateFilter=1&EventsHealthStateFilter=1&api-version=3.0"
    HEALTH_RESULT="$(wget --timeout=1 -qO - "$HEALTHURL" | jq -r .AggregatedHealthState)"
    NODE_COUNT="$(wget --timeout=1 -qO - "$HEALTHURL" | jq -r .HealthStatistics.HealthStateCountList[0].HealthStateCount.OkCount)"
    echo "Current Status $HEALTH_RESULT Nodes: $NODE_COUNT"
    if [ "$HEALTH_RESULT" = "Ok" ] && [ "$NODE_COUNT" = "3" ]; then
        return 1
    else
        echo "Waiting for health with 3 Nodes..." 
        return 0
    fi  
}; 

echo "######## Remove previous containers if they exist ###########"
docker rm -f sftestcluster 
docker rm -f sfsampleinstaller 
docker rm -f sfappinstaller

echo "######## Starting onebox cluster docker container ###########"
docker run --name sftestcluster -d -p 19080:19080 -p 19000:19000 -p 25100-25200:25100-25200 $DOCKERLOCATION/sfoneboxwithnode 

echo "Waiting for the cluster to start"
RESULT=0
until [[ $RESULT = 1 ]]
do
    sleep 5
    isClusterHealthy
    RESULT=$?
done

echo "######## Deploying sample node apps to cluster ###########"
if [ ! -f "./uploadtestapp.sh" ]
then
	echo "Cannot find 'uploadtestapp.sh' script must run under '/integration' folder."
    exit 1
fi
docker run --name sfappinstaller -d --network=host -v ${PWD}:/src $DOCKERLOCATION/sfctl -f ./uploadtestapp.sh

# Note: Previously attemted to use 'docker commit' on sftestcluster to capture the state so apps didn't need to be installed each time 
# however, this caused an issue with the SF cluster so have worked around this by installing apps each time.  

# This is required as -it fails when invoked from golang
# TODO: Investigate workarounds
docker wait sfappinstaller
docker logs sfappinstaller
docker rm -f sfappinstaller