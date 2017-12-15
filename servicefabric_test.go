package servicefabric

import (
	"context"
	"encoding/json"
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
		{ //Include a failed service in test data
			ReplicaItemBase: &sf.ReplicaItemBase{
				Address:                      `{"Endpoints":{"":"http://anotheraddress:8081"}}`,
				HealthState:                  "Error",
				LastInBuildDurationInSeconds: "3",
				NodeName:                     "_Node_0",
				ReplicaStatus:                "Down", // status is currently down.
				ServiceKind:                  "Stateless",
			},
			ID: "131497042182378183",
		},
	},
}

var labels = map[string]string{
	label.SuffixEnable: "true",
}

// TestUpdateconfig - This test ensures the provider returns a configuration message to
// the configuration channel when run.
func TestUpdateConfig(t *testing.T) {

	client := &clientMock{
		applications: apps,
		services:     services,
		partitions:   partitions,
		replicas:     nil,
		instances:    instances,
		labels:       labels,
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
	case <-configurationChan:
		t.Log("Received configuration object")
	case <-timeout:
		t.Error("Provider failed to return configuration")
	}
}

// TestServicesPresentInConfig tests that the basic services provide by SF
// are return in the configuration object
func TestServicesPresentInConfig(t *testing.T) {
	provider := Provider{}
	client := &clientMock{
		applications: apps,
		services:     services,
		partitions:   partitions,
		replicas:     nil,
		instances:    instances,
		labels:       labels,
	}
	config, err := provider.buildConfiguration(client)

	if err != nil {
		t.Error(err)
	}

	testCases := []struct {
		desc  string
		check func(types.Configuration) bool
	}{
		{
			desc:  "Has 1 Frontend",
			check: func(c types.Configuration) bool { return len(c.Frontends) == 1 },
		},
		{
			desc:  "Has 1 backend",
			check: func(c types.Configuration) bool { return len(c.Backends) == 1 },
		},
		{
			desc: "Backend for 'fabric:/TestApplication/TestService' exists",
			check: func(c types.Configuration) bool {
				_, exists := config.Backends["fabric:/TestApplication/TestService"]
				return exists
			},
		},
		{
			desc: "Backend has 1 server",
			check: func(c types.Configuration) bool {
				backend := config.Backends["fabric:/TestApplication/TestService"]
				return len(backend.Servers) == 1
			},
		},
		{
			desc: "Backend server has url 'http://localhost:8081'",
			check: func(c types.Configuration) bool {
				backend := config.Backends["fabric:/TestApplication/TestService"]
				return backend.Servers["131497042182378182"].URL == "http://localhost:8081"
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			if !test.check(*config) {
				t.Errorf("Check failed: %v", getJSON(config))
			}
		})
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
					t.Log(getJSON(frontend))
					t.Fail()
				}
			}
		})
	}
}

func TestBackendLabelConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		labels   map[string]string
		validate func(types.Backend) bool
	}{
		{
			desc: "Has DRR Loadbalencer",
			labels: map[string]string{
				label.SuffixEnable:                    "true",
				label.SuffixBackendLoadBalancerMethod: "drr",
			},
			validate: func(d types.Backend) bool { return d.LoadBalancer.Method == "drr" },
		},
		{
			desc: "Has circuit breaker set",
			labels: map[string]string{
				label.SuffixEnable:                "true",
				label.SuffixBackendCircuitBreaker: "NetworkErrorRatio() > 0.5",
			},
			validate: func(d types.Backend) bool { return d.CircuitBreaker.Expression == "NetworkErrorRatio() > 0.5" },
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

			if len(config.Backends) != 1 {
				t.Error("No backends present in the config")
			}

			for _, backend := range config.Backends {
				if backend == nil {
					t.Error("backend is nil")
				}
				if !test.validate(*backend) {
					t.Log(getJSON(backend))
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

func getJSON(i interface{}) string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
