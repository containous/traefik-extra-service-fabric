#!/bin/bash
echo "######## Connect to cluster ###########"
sfctl cluster select --endpoint http://localhost:19080
echo "######## Upload app ###########"
sfctl application upload --path ./traefik
echo "######## Provision type ###########"
sfctl application provision --application-type-build-path traefik
echo "######## Create instances ###########"
sfctl application create --app-type TraefikType --app-version 1.0.0 --app-name fabric:/traefik

echo "Waiting for deployment to finish..."
wait
sleep 60