#!/bin/bash
echo "######## Connect to cluster ###########"
sfctl cluster select --endpoint http://localhost:19080
echo "######## Clear down existing apps ###########"
sfctl application list --query items[].id -o tsv | grep node | xargs -n 1 -P 12 sfctl application delete --application-id

echo "######## Create instances ###########"
for i in {100..105}
do
   ( echo "Deploying instance $i"
   sfctl application create --app-type NodeAppType --app-version 1.0.0 --parameters "{\"PORT\":\"25$i\", \"Response\":\"Instance on port: 25$i\"}" --app-name "fabric:/node25$i" ) &
done

echo "Waiting for deployment to finish..."
wait