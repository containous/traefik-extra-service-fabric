#!/bin/bash
echo "######## Connect to cluster ###########"
sfctl cluster select --endpoint http://localhost:19080

echo "######## 1. NodeApps ###########"

echo "######## Upload app ###########"
sfctl application upload --path ./testapp
echo "######## Provision type ###########"
sfctl application provision --application-type-build-path testapp
echo "######## Create instances ###########"
for i in {100..105}
do
   ( echo "Deploying instance $i"
   sfctl application create --app-type NodeAppType --app-version 1.0.0 --parameters "{\"PORT\":\"25$i\", \"Response\":\"Instance on port: 25$i\"}" --app-name "fabric:/node25$i" ) &
done


echo "######## 2. Java Stateful App ###########"

echo "######## Upload app ###########"
sfctl application upload --path ./testapp_java
echo "######## Provision type ###########"
sfctl application provision --application-type-build-path testapp_java
echo "######## Create instances ###########"
sfctl application create --app-type VotingApplicationType --app-version 1.0.0 --app-name "fabric:/testapp_java"

echo "Waiting for deployment to finish..."
wait
sleep 60