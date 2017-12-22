### Integration Testing with Docker and Service Fabric

## Aims

Provide a quick and easy way to test Traefik Provider against a Service Fabric cluster

## Structure

- `/testapp`: Contains a simple NodeJS app to run in the SF cluster
- `cluster.dockerfile`: Builds a SF cluster image which starts quickly
- `clusterwithnode.dockerfile`: Builds a SF cluster image with nodejs installed
- `sfctl.dockerfile`: Builds an image with the `sfctl` tool installed. Used to interact with the cluster
- `build.sh`: Builds the all docker images
- `run.sh`: Creates a cluster listening on `http://localhost:19080` and installs 25 instances of `testapp`

## Usage

> Prerequisites:
> All scripts expect to be executed in the `integrationtesting` folder. 
> You need to add the additional docker daemon config to run SF in a container. See [details here.](https://docs.microsoft.com/en-us/azure/service-fabric/service-fabric-get-started-mac#create-a-local-container-and-set-up-service-fabric)

Full build: Run `build.sh` to create docker images then run `run.sh` to start a cluster
Lite Build: Run `run.sh` and images will be pulled from docker hub

