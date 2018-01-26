

#!/bin/bash

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


echo "Waiting for the cluster to be healthy"
ATTEMPTS=0
RESULT=0
until [[ $RESULT = 1 || $ATTEMPTS -gt 30 ]]
do
    sleep 5
    isClusterHealthy
    RESULT=$?
    ATTEMPTS=$((ATTEMPTS + 1))
done
