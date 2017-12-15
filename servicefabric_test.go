package servicefabric

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	sf "github.com/jjcollinge/servicefabric"
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
			ID: "131497042182378182",
		},
	},
}

var labels = map[string]string{
	label.SuffixEnable:            "true",
	"frontend.rule.default":       "Path: /",
	"backend.loadbalancer.method": "wrr",
	"backend.circuitbreaker":      "NetworkErrorRatio() > 0.5",
}

func TestUpdateConfig(t *testing.T) {

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
				"frontend-fabric:/TestApplication/TestService": {
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

func TestFrontendLabelConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		labels   map[string]string
		validate func(types.Frontend) bool
	}{
		{
			desc: "Has passTLSCert enabled",
			labels: map[string]string{
				label.SuffixEnable:              "true",
				label.SuffixFrontendPassTLSCert: "true",
			},
			validate: func(f types.Frontend) bool { return f.PassTLSCert },
		},
		{
			desc: "Has passTLSCert disabled",
			labels: map[string]string{
				label.SuffixEnable:              "true",
				label.SuffixFrontendPassTLSCert: "false",
			},
			validate: func(f types.Frontend) bool { return !f.PassTLSCert },
		},
		{
			desc: "Has FrameDeny enabled",
			labels: map[string]string{
				label.SuffixEnable:                   "true",
				label.SuffixFrontendHeadersFrameDeny: "true",
			},
			validate: func(f types.Frontend) bool { return f.Headers.FrameDeny },
		},
		{
			desc: "Has FrameDeny disabled",
			labels: map[string]string{
				label.SuffixEnable:                   "true",
				label.SuffixFrontendHeadersFrameDeny: "false",
			},
			validate: func(f types.Frontend) bool { return !f.Headers.FrameDeny },
		},
		{
			desc: "Has RequestHeaders set",
			labels: map[string]string{
				label.SuffixEnable:                 "true",
				label.SuffixFrontendRequestHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Headers.CustomRequestHeaders) == 2 && f.Headers.CustomRequestHeaders["X-Testing"] == "testing"
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			provider := Provider{}
			client := &clientMock{
				applications: apps,
				services:     services,
				partitions:   partitions,
				replicas:     nil,
				instances:    instances,
				labels:       test.labels,
			}
			config, err := provider.buildConfiguration(client)

			if err != nil {
				t.Error(err)
			}

			if len(config.Frontends) != 1 {
				t.Error("No frontends present in the config")
			}

			for _, frontend := range config.Frontends {
				if frontend == nil {
					t.Error("Frontend is nil")
				}
				if !test.validate(*frontend) {
					frontendJSON, _ := getJSON(frontend)
					t.Log(frontendJSON)
					t.Fail()
				}
			}
		})
	}
}

func TestIsPrimary(t *testing.T) {
	testCases := []struct {
		desc     string
		replica  *sf.ReplicaItem
		expected bool
	}{
		{
			desc: "when primary",
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
			desc: "When secondary",
			replica: &sf.ReplicaItem{
				ReplicaItemBase: &sf.ReplicaItemBase{
					Address:                      `{"Endpoints":{"":"localhost:30001+bce46a8c-b62d-4996-89dc-7ffc00a96902-131496928082309293"}}`,
					HealthState:                  "Ok",
					LastInBuildDurationInSeconds: "1",
					NodeName:                     "_Node_0",
					ReplicaRole:                  "Secondary",
					ReplicaStatus:                "Ready",
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

			primary := isPrimary(test.replica)

			if !primary && test.expected || primary && !test.expected {
				t.Errorf("Incorrectly identified primary state of a replica. Got %v, expected %v", primary, test.expected)
			}
		})
	}
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

func getJSON(i interface{}) (string, error) {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
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
					return fmt.Errorf("backend configuration differs from expected configuration: got %q, expected %q", actualFrontendsStr, expectedFrontendsStr)
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
			return fmt.Errorf("backends count differs from expected: got %+v, expected %+v", actual.Configuration.Backends, expected.Configuration.Backends)
		}
		return fmt.Errorf("frontends count differs from expected: got %+v, expected %+v", actual.Configuration.Frontends, expected.Configuration.Frontends)
	}
	return fmt.Errorf("provider name differs from expected: got %q, expected %q", actual.ProviderName, expected.ProviderName)
}
