package servicefabric

import (
	"fmt"

	sf "github.com/jjcollinge/servicefabric"
)

type clientMock struct {
	applications                 *sf.ApplicationItemsPage
	services                     *sf.ServiceItemsPage
	partitions                   *sf.PartitionItemsPage
	replicas                     *sf.ReplicaItemsPage
	instances                    *sf.InstanceItemsPage
	getServicelabelsResult       map[string]string
	expectedPropertyName         string
	getServiceExtensionMapResult map[string]string
	getPropertiesResult          map[string]string
}

func (c *clientMock) GetApplications() (*sf.ApplicationItemsPage, error) {
	return c.applications, nil
}

func (c *clientMock) GetServices(appName string) (*sf.ServiceItemsPage, error) {
	return c.services, nil
}

func (c *clientMock) GetPartitions(appName, serviceName string) (*sf.PartitionItemsPage, error) {
	return c.partitions, nil
}

func (c *clientMock) GetReplicas(appName, serviceName, partitionName string) (*sf.ReplicaItemsPage, error) {
	return c.replicas, nil
}

func (c *clientMock) GetInstances(appName, serviceName, partitionName string) (*sf.InstanceItemsPage, error) {
	return c.instances, nil
}

func (c *clientMock) GetServiceExtensionMap(service *sf.ServiceItem, app *sf.ApplicationItem, extensionKey string) (map[string]string, error) {
	if extensionKey != traefikServiceFabricExtensionKey {
		return nil, fmt.Errorf("extension key not expected value have: %s expect: %s", extensionKey, traefikServiceFabricExtensionKey)
	}
	return c.getServiceExtensionMapResult, nil
}

func (c *clientMock) GetServiceLabels(service *sf.ServiceItem, app *sf.ApplicationItem, prefix string) (map[string]string, error) {
	return c.getServicelabelsResult, nil
}

// Note this is dumb mock the `exists`.
func (c *clientMock) GetProperties(name string) (bool, map[string]string, error) {
	if c.expectedPropertyName == name {
		return true, c.getPropertiesResult, nil
	}
	return false, nil, nil
}
