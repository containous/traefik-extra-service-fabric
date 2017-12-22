#!/bin/bash
echo "######## Upload app ###########"
sfctl application upload --path ./testapp
echo "######## Provision type ###########"
sfctl application provision --application-type-build-path testapp
echo "######## Create 200 instances ###########"
for i in {100..125}
do
   ( echo "Deploying instance $i"
   sfctl application create --app-type NodeAppType --app-version 1.0.0 --parameters "{\"PORT\":\"25$i\", \"Response\":\"Instance on port: 25$i\"}" --app-name "fabric:/node25$i" ) &
done

echo "Waiting for deployment to finish..."
wait