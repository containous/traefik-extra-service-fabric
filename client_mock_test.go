package servicefabric

import (
	sf "github.com/jjcollinge/servicefabric"
)

type clientMock struct {
	applications *sf.ApplicationItemsPage
	services     *sf.ServiceItemsPage
	partitions   *sf.PartitionItemsPage
	replicas     *sf.ReplicaItemsPage
	instances    *sf.InstanceItemsPage
	labels       map[string]string
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

func (c *clientMock) GetServiceExtension(appType, applicationVersion, serviceTypeName, extensionKey string, response interface{}) error {
	return nil
}

func (c *clientMock) GetServiceLabels(service *sf.ServiceItem, app *sf.ApplicationItem, prefix string) (map[string]string, error) {
	return c.labels, nil
}
