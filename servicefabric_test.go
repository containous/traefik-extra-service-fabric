package servicefabric

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	sf "github.com/jjcollinge/servicefabric"
)

func TestUpdateConfig(t *testing.T) {
	apps := &sf.ApplicationItemsPage{
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
	services := &sf.ServiceItemsPage{
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
	partitions := &sf.PartitionItemsPage{
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
	instances := &sf.InstanceItemsPage{
		ContinuationToken: nil,
		Items: []sf.InstanceItem{
			{
				ReplicaItemBase: &sf.ReplicaItemBase{
					Address:                      "{\"Endpoints\":{\"\":\"http:\\/\\/localhost:8081\"}}",
					HealthState:                  "Ok",
					LastInBuildDurationInSeconds: "3",
					NodeName:                     "_Node_0",
					ReplicaStatus:                "Ready",
					ServiceKind:                  "Stateless",
				},
				ID: "131497042182378182",
			},
		},
	}

	labels := map[string]string{
		"expose":                      "true",
		"frontend.rule.default":       "Path: /",
		"backend.loadbalancer.method": "wrr",
		"backend.circuitbreaker":      "NetworkErrorRatio() > 0.5",
	}

	client := &clientMock{
		applications: apps,
		services:     services,
		partitions:   partitions,
		replicas:     nil,
		instances:    instances,
		labels:       labels,
	}
	expected := types.ConfigMessage{
		ProviderName: "servicefabric",
		Configuration: &types.Configuration{
			Frontends: map[string]*types.Frontend{
				"fabric:/TestApplication/TestService": {
					EntryPoints: []string{},
					Backend:     "fabric:/TestApplication/TestService",
					Routes: map[string]types.Route{
						"frontend.rule.default": {
							Rule: "Path: /",
						},
					},
				},
			},
			Backends: map[string]*types.Backend{
				"fabric:/TestApplication/TestService": {
					LoadBalancer: &types.LoadBalancer{
						Method: "wrr",
					},
					CircuitBreaker: &types.CircuitBreaker{
						Expression: "NetworkErrorRatio() > 0.5",
					},
					Servers: map[string]types.Server{
						"131497042182378182": {
							URL:    "http://localhost:8081",
							Weight: 1,
						},
					},
				},
			},
		},
	}

	provider := Provider{}
	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)
	defer pool.Stop()

	err := provider.updateConfig(configurationChan, pool, client, time.Millisecond*100)
	if err != nil {
		t.Fatal(err)
	}

	timeout := make(chan string, 1)
	go func() {
		time.Sleep(time.Second * 2)
		timeout <- "Timeout triggered"
	}()

	select {
	case actual := <-configurationChan:
		err := compareConfigurations(actual, expected)
		if err != nil {
			res, _ := json.Marshal(actual)
			t.Log(string(res))
			t.Error(err)
		}
	case <-timeout:
		t.Error("Provider failed to return configuration")
	}
}

func TestIsPrimary(t *testing.T) {
	replica := &sf.ReplicaItem{
		ReplicaItemBase: &sf.ReplicaItemBase{
			Address:                      "{\"Endpoints\":{\"\":\"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293\"}}",
			HealthState:                  "Ok",
			LastInBuildDurationInSeconds: "1",
			NodeName:                     "_Node_0",
			ReplicaRole:                  "Primary",
			ReplicaStatus:                "Ready",
			ServiceKind:                  "Stateful",
		},
		ID: "131496928082309293",
	}
	isPrimary := isPrimary(replica)
	if !isPrimary {
		t.Error("Failed to identify replica as primary")
	}
}

func TestIsPrimaryWhenSecondary(t *testing.T) {
	replica := &sf.ReplicaItem{
		ReplicaItemBase: &sf.ReplicaItemBase{
			Address:                      "{\"Endpoints\":{\"\":\"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293\"}}",
			HealthState:                  "Ok",
			LastInBuildDurationInSeconds: "1",
			NodeName:                     "_Node_0",
			ReplicaRole:                  "Secondary",
			ReplicaStatus:                "Ready",
			ServiceKind:                  "Stateful",
		},
		ID: "131496928082309293",
	}
	isPrimary := isPrimary(replica)
	if isPrimary {
		t.Error("Incorrectly identified replica as primary")
	}
}

func TestIsHealthy(t *testing.T) {
	replica := &sf.ReplicaItem{
		ReplicaItemBase: &sf.ReplicaItemBase{
			Address:                      "{\"Endpoints\":{\"\":\"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293\"}}",
			HealthState:                  "Ok",
			LastInBuildDurationInSeconds: "1",
			NodeName:                     "_Node_0",
			ReplicaRole:                  "Primary",
			ReplicaStatus:                "Ready",
			ServiceKind:                  "Stateful",
		},
		ID: "131496928082309293",
	}
	isHealthy := isHealthy(replica.ReplicaItemBase)
	if !isHealthy {
		t.Error("Failed to identify replica as healthy")
	}
}

func TestIsHealthyWhenError(t *testing.T) {
	replica := &sf.ReplicaItem{
		ReplicaItemBase: &sf.ReplicaItemBase{
			Address:                      "{\"Endpoints\":{\"\":\"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293\"}}",
			HealthState:                  "Error",
			LastInBuildDurationInSeconds: "1",
			NodeName:                     "_Node_0",
			ReplicaRole:                  "Primary",
			ReplicaStatus:                "Error",
			ServiceKind:                  "Stateful",
		},
		ID: "131496928082309293",
	}
	isHealthy := isHealthy(replica.ReplicaItemBase)
	if isHealthy {
		t.Error("Incorrectly identified replica as healthy")
	}
}

func compareConfigurations(actual, expected types.ConfigMessage) error {
	if actual.ProviderName == expected.ProviderName {
		if len(actual.Configuration.Frontends) == len(expected.Configuration.Frontends) {
			if len(actual.Configuration.Backends) == len(expected.Configuration.Backends) {
				actualFrontends, err := json.Marshal(actual.Configuration.Frontends)
				if err != nil {
					return err
				}
				actualFrontendsStr := string(actualFrontends)

				expectedFrontends, err := json.Marshal(expected.Configuration.Frontends)
				if err != nil {
					return err
				}
				expectedFrontendsStr := string(expectedFrontends)

				if actualFrontendsStr != expectedFrontendsStr {
					return fmt.Errorf("backend configuration differs from expected configuration: got %q, want %q", actualFrontendsStr, expectedFrontendsStr)
				}

				actualBackends, err := json.Marshal(actual.Configuration.Backends)
				if err != nil {
					return err
				}
				actualBackendsStr := string(actualBackends)
				expectedBackends, err := json.Marshal(expected.Configuration.Backends)
				if err != nil {
					return err
				}
				expectedBackendsStr := string(expectedBackends)

				if actualBackendsStr != expectedBackendsStr {
					return err
				}
				return nil
			}
			return fmt.Errorf("backends count differs from expected: got %+v, want %+v", actual.Configuration.Backends, expected.Configuration.Backends)
		}
		return fmt.Errorf("frontends count differs from expected: got %+v, want %+v", actual.Configuration.Frontends, expected.Configuration.Frontends)
	}
	return fmt.Errorf("provider name differs from expected: got %q, want %q", actual.ProviderName, expected.ProviderName)
}
