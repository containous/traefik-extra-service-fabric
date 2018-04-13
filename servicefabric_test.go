package servicefabric

import (
	"context"
	"testing"
	"time"

	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	sf "github.com/jjcollinge/servicefabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var apps = &sf.ApplicationItemsPage{
	ContinuationToken: nil,
	Items: []sf.ApplicationItem{
		{
			HealthState: "Ok",
			ID:          "TestApplication",
			Name:        "fabric:/TestApplication",
			Parameters: []*sf.AppParameter{
				{Key: "TraefikPublish", Value: "fabric:/TestApplication/TestService"},
			},
			Status:      "Ready",
			TypeName:    "TestApplicationType",
			TypeVersion: "1.0.0",
		},
	},
}

var services = &sf.ServiceItemsPage{
	ContinuationToken: nil,
	Items: []sf.ServiceItem{
		{
			HasPersistedState: true,
			HealthState:       "Ok",
			ID:                "TestApplication/TestService",
			IsServiceGroup:    false,
			ManifestVersion:   "1.0.0",
			Name:              "fabric:/TestApplication/TestService",
			ServiceKind:       "Stateless",
			ServiceStatus:     "Active",
			TypeName:          "TestServiceType",
		},
	},
}

var partitions = &sf.PartitionItemsPage{
	ContinuationToken: nil,
	Items: []sf.PartitionItem{
		{
			CurrentConfigurationEpoch: sf.ConfigurationEpoch{
				ConfigurationVersion: "12884901891",
				DataLossVersion:      "131496928071680379",
			},
			HealthState:       "Ok",
			MinReplicaSetSize: 1,
			PartitionInformation: sf.PartitionInformation{
				HighKey:              "9223372036854775807",
				ID:                   "bce46a8c-b62d-4996-89dc-7ffc00a96902",
				LowKey:               "-9223372036854775808",
				ServicePartitionKind: "Int64Range",
			},
			PartitionStatus:      "Ready",
			ServiceKind:          "Stateless",
			TargetReplicaSetSize: 1,
		},
	},
}

var instances = &sf.InstanceItemsPage{
	ContinuationToken: nil,
	Items: []sf.InstanceItem{
		{
			ReplicaItemBase: &sf.ReplicaItemBase{
				Address:                      `{"Endpoints":{"":"http://localhost:8081"}}`,
				HealthState:                  "Ok",
				LastInBuildDurationInSeconds: "3",
				NodeName:                     "_Node_0",
				ReplicaStatus:                "Ready",
				ServiceKind:                  "Stateless",
			},
			ID: "1",
		},
		// Include a failed service in test data
		{
			ReplicaItemBase: &sf.ReplicaItemBase{
				Address:                      `{"Endpoints":{"":"http://anotheraddress:8081"}}`,
				HealthState:                  "Error",
				LastInBuildDurationInSeconds: "3",
				NodeName:                     "_Node_0",
				ReplicaStatus:                "Down", // status is currently down.
				ServiceKind:                  "Stateless",
			},
			ID: "2",
		},
	},
}

var labels = map[string]string{
	label.TraefikEnable: "true",
}

// TestUpdateconfig - This test ensures the provider returns a configuration message to
// the configuration channel when run.
func TestUpdateConfig(t *testing.T) {
	client := &clientMock{
		applications:           apps,
		services:               services,
		partitions:             partitions,
		replicas:               nil,
		instances:              instances,
		getServicelabelsResult: labels,
		expectedPropertyName:   services.Items[0].ID,
	}

	provider := Provider{}
	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)
	defer pool.Stop()

	err := provider.updateConfig(configurationChan, pool, client, time.Millisecond*100)
	require.NoError(t, err)

	timeout := make(chan string, 1)
	go func() {
		time.Sleep(time.Second * 2)
		timeout <- "Timeout triggered"
	}()

	select {
	case <-configurationChan:
		t.Log("Received configuration object")
	case <-timeout:
		t.Error("Provider failed to return configuration")
	}
}

func TestGetLabelsDisableLabelOverrides(t *testing.T) {
	extensionLabels := map[string]string{
		label.TraefikEnable:           "true",
		traefikSFEnableLabelOverrides: "false",
	}

	propertyLabels := map[string]string{
		"shouldnotexist": "true",
	}

	client := &clientMock{
		applications:                 apps,
		services:                     services,
		partitions:                   partitions,
		replicas:                     nil,
		instances:                    instances,
		getPropertiesResult:          propertyLabels,
		getServiceExtensionMapResult: extensionLabels,
		expectedPropertyName:         services.Items[0].ID,
	}

	res, err := getLabels(client, &services.Items[0], &apps.Items[0])
	require.NoError(t, err)

	_, exists := res["shouldnotexist"]
	assert.False(t, exists)
}

func TestIsHealthy(t *testing.T) {
	testCases := []struct {
		desc     string
		replica  *sf.ReplicaItem
		expected bool
	}{
		{
			desc: "when healthy",
			replica: &sf.ReplicaItem{
				ReplicaItemBase: &sf.ReplicaItemBase{
					Address:                      `{"Endpoints":{"":"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293"}}`,
					HealthState:                  "Ok",
					LastInBuildDurationInSeconds: "1",
					NodeName:                     "_Node_0",
					ReplicaRole:                  "Primary",
					ReplicaStatus:                "Ready",
					ServiceKind:                  "Stateful",
				},
				ID: "131496928082309293",
			},
			expected: true,
		},
		{
			desc: "When error",
			replica: &sf.ReplicaItem{
				ReplicaItemBase: &sf.ReplicaItemBase{
					Address:                      `{"Endpoints":{"":"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293"}}`,
					HealthState:                  "Error",
					LastInBuildDurationInSeconds: "1",
					NodeName:                     "_Node_0",
					ReplicaRole:                  "Primary",
					ReplicaStatus:                "Error",
					ServiceKind:                  "Stateful",
				},
				ID: "131496928082309293",
			},
			expected: false,
		},
		{
			desc: "When replica down but health only warning",
			replica: &sf.ReplicaItem{
				ReplicaItemBase: &sf.ReplicaItemBase{
					Address:                      `{"Endpoints":{"":"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293"}}`,
					HealthState:                  "Warning",
					LastInBuildDurationInSeconds: "1",
					NodeName:                     "_Node_0",
					ReplicaRole:                  "Primary",
					ReplicaStatus:                "Down",
					ServiceKind:                  "Stateful",
				},
				ID: "131496928082309293",
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			healthy := isHealthy(test.replica.ReplicaItemBase)

			if !healthy && test.expected || healthy && !test.expected {
				t.Errorf("Incorrectly identified healthy state of a replica. Got %v, expected %v", healthy, test.expected)
			}
		})
	}
}

func TestIsStateX(t *testing.T) {
	testCases := []struct {
		desc              string
		serviceItem       ServiceItemExtended
		expectedStateless bool
		expectedStateful  bool
	}{
		{
			desc: "With Stateful service",
			serviceItem: ServiceItemExtended{
				ServiceItem: sf.ServiceItem{
					ServiceKind: "Stateful",
				},
			},
			expectedStateless: false,
			expectedStateful:  true,
		},
		{
			desc: "With Stateless service",
			serviceItem: ServiceItemExtended{
				ServiceItem: sf.ServiceItem{
					ServiceKind: "Stateless",
				},
			},
			expectedStateless: true,
			expectedStateful:  false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			isStatefulResult := isStateful(test.serviceItem)
			isStatelessResult := isStateless(test.serviceItem)

			if isStatefulResult != test.expectedStateful {
				t.Errorf("Failed isStateful. Got %v, expected %v", isStatefulResult, test.expectedStateful)
			}

			if isStatelessResult != test.expectedStateless {
				t.Errorf("Failed isStateless. Got %v, expected %v", isStatelessResult, test.expectedStateless)
			}
		})
	}
}

func TestGetReplicaDefaultEndpoint(t *testing.T) {
	testCases := []struct {
		desc             string
		replicaData      *sf.ReplicaItemBase
		expectedEndpoint string
		errorExpected    bool
	}{
		{
			desc: "valid default endpoint",
			replicaData: &sf.ReplicaItemBase{
				Address: `{"Endpoints":{"":"http://localhost:8081"}}`,
			},
			expectedEndpoint: "http://localhost:8081",
		},
		{
			desc: "invalid default endpoint",
			replicaData: &sf.ReplicaItemBase{
				Address: `{"Endpoints":{"":"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293"}}`,
			},
			errorExpected: true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			defaultEndpoint, err := getReplicaDefaultEndpoint(test.replicaData)
			if test.errorExpected {
				if err == nil {
					t.Fatal("Expected an error, got no error")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

				if defaultEndpoint != test.expectedEndpoint {
					t.Errorf("Got %s, want %s", defaultEndpoint, test.expectedEndpoint)
				}
			}
		})
	}
}
