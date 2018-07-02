#!/bin/bash
echo "######## Connect to cluster ###########"
sfctl cluster select --endpoint http://localhost:19080
echo "######## Clear down existing apps ###########"
sfctl application list --query items[].id -o tsv | xargs -n 1 sfctl application delete --application-id
sfctl application unprovision --application-type-name NodeAppType --application-type-version 1.0.0
sfctl application unprovision --application-type-name TraefikType --application-type-version 1.0.0

echo "Waiting for deployment to finish..."
wait