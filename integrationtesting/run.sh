#!/bin/bash

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

echo "######## Starting onebox cluster docker container ###########"
docker run --name sftestcluster -d --rm -p 19080:19080 -p 19000:19000 -p 25100-25200:25100-25200 lawrencegripper/sfoneboxwithnode 

echo "Waiting for the cluster to start"
RESULT=0
until [[ $RESULT = 1 ]]
do
    sleep 5
    isClusterHealthy
    RESULT=$?
done

echo "######## Deploying sample node apps to cluster ###########"
docker run --name appinstaller -it --rm --network=host -v ${PWD}:/src lawrencegripper/sfctl -f ./uploadtestapp.sh

