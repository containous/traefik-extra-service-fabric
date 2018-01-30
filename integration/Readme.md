### Integration Testing with Docker and Service Fabric

## Aims

Provide a quick and easy way to test Traefik Provider against a Service Fabric cluster

## Running the tests

Normal execution: `go test -v .`

This will just show test results. 

Verbose: `go test -v . -sfintegration.verbose`

This will show log output from the provider and script output from starting and resetting the scripts. 

## All Flags

- `sfintegration.verbose`: Shows full output from scripts and additional logging in `stdout` 
- `sfintegration.clusterrunning`: Skips starting and stopping cluster to enable fast local testing when a cluster is already running on your machine. For example: manually run `./scripts/run.sh` then use this to skip waiting for cluster start and stop when adding or developing tests.

## Structure

### Docker images: `/docker`

These build the necessary images to run the integration tests. Pre-built images are on dockerhub so only necessary for changes to images.

- `cluster.dockerfile`: Builds a SF cluster image which starts quickly
- `clusterwithnode.dockerfile`: Builds a SF cluster image with nodejs installed
- `sfctl.dockerfile`: Builds an image with the `sfctl` tool installed. Used to interact with the cluster
- `build.sh`: Builds the all docker images

### Sample app: `/testapp`

This is a simple ServiceFabric app running nodejs

- `WebServicePkg/ServiceManifest.xml`: Manifest in which labels are defined

### Managing the cluster: `/scripts`

These scripts create the cluster, check health metrics etc.

- `run.sh`: Pre-test - Creates a cluster listening on `http://localhost:19080` and installs instances of `testapp`
    - `upload_test_apps.sh`: Used with the `sfctl` container to install test apps in the cluster
- `reset.sh`: Between tests - Removes app instances and reinstalls to ensure tests can't affect each other
    - `reset_test_apps.sh`: Used with `sfctl` container to reset cluster state
- `stop.sh`: Post-test - Stops containers and cleans up

