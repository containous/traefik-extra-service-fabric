package servicefabric

import (
	"context"
	"encoding/json"
	"errors"
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
	label.TraefikEnable: "true",
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

func requestConfig(provider Provider, client *clientMock) (types.Configuration, error) {
	config, err := provider.buildConfiguration(client)

	if err != nil {
		return types.Configuration{}, err
	}

	if config == nil {
		return types.Configuration{}, errors.New("Returned nil config")
	}

	return *config, nil
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

	config, err := requestConfig(provider, client)
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
			if !test.check(config) {
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
			desc: "Has passHostHeader enabled",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendPassHostHeader: "true",
			},
			validate: func(f types.Frontend) bool { return f.PassHostHeader },
		},
		{
			desc: "Has passHostHeader disabled",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendPassHostHeader: "false",
			},
			validate: func(f types.Frontend) bool { return !f.PassHostHeader },
		},
		{
			desc: "Has whitelistSourceRange set",
			labels: map[string]string{
				label.TraefikEnable:                       "true",
				label.TraefikFrontendWhitelistSourceRange: "[\"10.0.0.1\", \"10.0.0.2\"]",
			},
			validate: func(f types.Frontend) bool {
				if len(f.WhitelistSourceRange) != 2 {
					return false
				}
				return f.WhitelistSourceRange[0] == "10.0.0.1" && f.WhitelistSourceRange[1] == "10.0.0.2"
			},
		},
		{
			desc: "Has priority set",
			labels: map[string]string{
				label.TraefikEnable:           "true",
				label.TraefikFrontendPriority: "13",
			},
			validate: func(f types.Frontend) bool { return f.Priority == 13 },
		},
		{
			desc: "Has basicAuth set",
			labels: map[string]string{
				label.TraefikEnable:            "true",
				label.TraefikFrontendAuthBasic: "[\"USER:HASH\"]",
			},
			validate: func(f types.Frontend) bool {
				return len(f.BasicAuth) == 1 && f.BasicAuth[0] == "USER:HASH"
			},
		},
		{
			desc: "Has entrypoints set",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendEntryPoints: "Barry, Bob",
			},
			validate: func(f types.Frontend) bool {
				return len(f.EntryPoints) == 2 && f.EntryPoints[0] == "Barry" && f.EntryPoints[1] == "Bob"
			},
		},
		{
			desc: "Has passTLSCert enabled",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendPassTLSCert: "true",
			},
			validate: func(f types.Frontend) bool { return f.PassTLSCert },
		},
		{
			desc: "Has passTLSCert disabled",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendPassTLSCert: "false",
			},
			validate: func(f types.Frontend) bool { return !f.PassTLSCert },
		},
		{
			desc: "Has FrameDeny enabled",
			labels: map[string]string{
				label.TraefikEnable:            "true",
				label.TraefikFrontendFrameDeny: "true",
			},
			validate: func(f types.Frontend) bool { return f.Headers.FrameDeny },
		},
		{
			desc: "Has FrameDeny disabled",
			labels: map[string]string{
				label.TraefikEnable:            "true",
				label.TraefikFrontendFrameDeny: "false",
			},
			validate: func(f types.Frontend) bool { return !f.Headers.FrameDeny },
		},
		{
			desc: "Has RequestHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendRequestHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Headers.CustomRequestHeaders) == 2 && f.Headers.CustomRequestHeaders["X-Testing"] == "testing"
			},
		},
		{
			desc: "Has rule set",
			labels: map[string]string{
				label.TraefikEnable:                    "true",
				label.TraefikFrontendRule + ".default": "Path: /",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Routes) == 1 && f.Routes["frontend.rule.default"].Rule == "Path: /"
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

			config, err := requestConfig(provider, client)
			if err != nil {
				t.Error(err)
			}

			if config.Frontends == nil || len(config.Frontends) != 1 {
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
				label.TraefikEnable:                    "true",
				label.TraefikBackendLoadBalancerMethod: "drr",
			},
			validate: func(b types.Backend) bool { return b.LoadBalancer.Method == "drr" },
		},
		{
			desc: "Has healthcheck set",
			labels: map[string]string{
				label.TraefikEnable:                     "true",
				label.TraefikBackendHealthCheckPath:     "/hc",
				label.TraefikBackendHealthCheckInterval: "1337s",
			},
			validate: func(b types.Backend) bool {
				if b.HealthCheck == nil {
					return false
				}
				return b.HealthCheck.Path == "/hc" && b.HealthCheck.Interval == "1337s"
			},
		},
		{
			desc: "Has circuit breaker set",
			labels: map[string]string{
				label.TraefikEnable:                "true",
				label.TraefikBackendCircuitBreaker: "NetworkErrorRatio() > 0.5",
			},
			validate: func(b types.Backend) bool {
				if b.CircuitBreaker == nil {
					return false
				}
				return b.CircuitBreaker.Expression == "NetworkErrorRatio() > 0.5"
			},
		},
		// {
		// 	desc: "Has stickiness loadbalencer set with cookie name",
		// 	labels: map[string]string{
		// 		label.TraefikEnable:                                  "true",
		// 		label.TraefikBackendLoadBalancerStickiness:           "true",
		// 		label.TraefikBackendLoadBalancerStickinessCookieName: "stickycookie",
		// 	},
		// 	validate: func(b types.Backend) bool {
		// 		return b.LoadBalancer.Stickiness != nil && b.LoadBalancer.Stickiness.CookieName == "stickycookie"
		// 	},
		// },
		{
			desc: "Has stickiness cookie set",
			labels: map[string]string{
				label.TraefikEnable:                        "true",
				label.TraefikBackendLoadBalancerStickiness: "true",
			},
			validate: func(b types.Backend) bool { return b.LoadBalancer.Stickiness != nil },
		},
		{
			desc: "Has maxconn amount and extractor func",
			labels: map[string]string{
				label.TraefikEnable:                      "true",
				label.TraefikBackendMaxConnAmount:        "1337",
				label.TraefikBackendMaxConnExtractorFunc: "request.header.TEST_HEADER",
			},
			validate: func(b types.Backend) bool {
				if b.MaxConn == nil {
					return false
				}
				return b.MaxConn.Amount == 1337 && b.MaxConn.ExtractorFunc == "request.header.TEST_HEADER"
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
			config, err := requestConfig(provider, client)
			if err != nil {
				t.Error(err)
			}
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
