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
			ID: "1",
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
				return backend.Servers["1"].URL == "http://localhost:8081"
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

// nolint: gocyclo
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
				label.TraefikFrontendWhitelistSourceRange: "10.0.0.1, 10.0.0.2",
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
				label.TraefikFrontendAuthBasic: "USER1:HASH1, USER2:HASH2",
			},
			validate: func(f types.Frontend) bool {
				return len(f.BasicAuth) == 2 && f.BasicAuth[0] == "USER1:HASH1"
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
			desc: "Has rule set",
			labels: map[string]string{
				label.TraefikEnable:                    "true",
				label.TraefikFrontendRule + ".default": "Path: /",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Routes) == 1 && f.Routes[label.TraefikFrontendRule+".default"].Rule == "Path: /"
			},
		},
		{
			desc: "Has SSLRedirectHeaders set",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendSSLRedirect: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.SSLRedirect
			},
		},
		{
			desc: "Has Temporary SSLRedirectHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                       "true",
				label.TraefikFrontendSSLTemporaryRedirect: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.SSLTemporaryRedirect
			},
		},
		//Todo: Is this behaviour correct "bob.bob.com" => "[bob.bob.com]"?
		{
			desc: "Has SSLHostHeaders set",
			labels: map[string]string{
				label.TraefikEnable:          "true",
				label.TraefikFrontendSSLHost: "bob.bob.com",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.SSLHost == "[bob.bob.com]"
			},
		},
		{
			desc: "Has STSSecondsHeaders set",
			labels: map[string]string{
				label.TraefikEnable:             "true",
				label.TraefikFrontendSTSSeconds: "1337",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.STSSeconds == 1337
			},
		},
		{
			desc: "Has STSIncludeSubdomainsHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                       "true",
				label.TraefikFrontendSTSIncludeSubdomains: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.STSIncludeSubdomains
			},
		},
		{
			desc: "Has STSPreloadHeaders set",
			labels: map[string]string{
				label.TraefikEnable:             "true",
				label.TraefikFrontendSTSPreload: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.STSPreload
			},
		},
		{
			desc: "Has hasForceSTSHeaderHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendForceSTSHeader: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.ForceSTSHeader
			},
		},
		{
			desc: "Has hasForceSTSHeaderHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendForceSTSHeader: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.ForceSTSHeader
			},
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
		//Todo: Is this behaviour correct "SAMEORIGIN" => "[SAMEORIGIN]"?
		{
			desc: "hasCustomFrameOptionsValueHeaders",
			labels: map[string]string{
				label.TraefikEnable:                          "true",
				label.TraefikFrontendCustomFrameOptionsValue: "SAMEORIGIN",
			},
			validate: func(f types.Frontend) bool { return f.Headers.CustomFrameOptionsValue == "[SAMEORIGIN]" },
		},
		{
			desc: "Has ContentTypeNosniffHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                     "true",
				label.TraefikFrontendContentTypeNosniff: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.ContentTypeNosniff
			},
		},
		{
			desc: "Has BrowserXSSFilterHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                   "true",
				label.TraefikFrontendBrowserXSSFilter: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.BrowserXSSFilter
			},
		},
		{
			desc: "Has ContentSecurityPolicyHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                        "true",
				label.TraefikFrontendContentSecurityPolicy: "plugin-types image/png application/pdf; sandbox",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.ContentSecurityPolicy == "[plugin-types image/png application/pdf; sandbox]"
			},
		},
		{
			desc: "Has PublicKeyHeaders set",
			labels: map[string]string{
				label.TraefikEnable:            "true",
				label.TraefikFrontendPublicKey: "somekeydata",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.PublicKey == "[somekeydata]"
			},
		},
		{
			desc: "Has ReferrerPolicyHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendReferrerPolicy: "same-origin",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.ReferrerPolicy == "[same-origin]"
			},
		},
		{
			desc: "Has IsDevelopmentHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                "true",
				label.TraefikFrontendIsDevelopment: "true",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.IsDevelopment
			},
		},
		{
			desc: "Has AllowedhostHeaders set",
			labels: map[string]string{
				label.TraefikEnable:               "true",
				label.TraefikFrontendAllowedHosts: "host1, host2",
			},
			validate: func(f types.Frontend) bool {
				return f.Headers.AllowedHosts[0] == "host1"
			},
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
			desc: "Has ResponseHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                  "true",
				label.TraefikFrontendResponseHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Headers.CustomResponseHeaders) == 2 && f.Headers.CustomResponseHeaders["X-Testing"] == "testing"
			},
		},
		{
			desc: "Has SSLProxyHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                  "true",
				label.TraefikFrontendSSLProxyHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(f types.Frontend) bool {
				return len(f.Headers.SSLProxyHeaders) == 2 && f.Headers.SSLProxyHeaders["X-Testing"] == "testing"
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

// nolint: gocyclo
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
				label.TraefikBackendHealthCheckPort:     "9000",
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
				label.TraefikEnable:                          "true",
				label.TraefikBackendCircuitBreakerExpression: "NetworkErrorRatio() > 0.5",
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

func TestGroupedServicesFrontends(t *testing.T) {
	groupName := "groupedbackends"
	groupWeight := "154"

	provider := Provider{}
	client := &clientMock{
		applications: apps,
		services:     services,
		partitions:   partitions,
		replicas:     nil,
		instances:    instances,
		labels: map[string]string{
			label.TraefikEnable:  "true",
			TraefikSFGroupName:   groupName,
			TraefikSFGroupWeight: groupWeight,
		},
	}
	config, err := requestConfig(provider, client)
	if err != nil {
		t.Error(err)
	}
	if err != nil {
		t.Error(err)
	}

	if len(config.Frontends) != 2 {
		t.Log(getJSON(config))
		t.Log("Incorrect count of frontends present in the config")
		t.FailNow()
	}

	if len(config.Backends) != 2 {
		t.Log(getJSON(config))
		t.Log("Incorrect count of backends present in the config")
		t.FailNow()
	}

	frontend, exists := config.Frontends[groupName]

	if !exists {
		t.Log(getJSON(config))
		t.Log("Missing frontend for grouped service")
		t.FailNow()
	}

	if frontend.Priority == 50 && frontend.Backend == groupName {
		t.Log("Frontend exists for group")
	}
}

func TestGroupedServicesBackends(t *testing.T) {
	groupName := "groupedbackends"
	groupWeight := "154"

	provider := Provider{}
	client := &clientMock{
		applications: apps,
		services:     services,
		partitions:   partitions,
		replicas:     nil,
		instances:    instances,
		labels: map[string]string{
			label.TraefikEnable:  "true",
			TraefikSFGroupName:   groupName,
			TraefikSFGroupWeight: groupWeight,
		},
	}
	config, err := requestConfig(provider, client)
	if err != nil {
		t.Error(err)
	}
	if err != nil {
		t.Error(err)
	}

	if len(config.Backends) != 2 {
		t.Log(getJSON(config))
		t.Log("Incorrect count of backends present in the config")
		t.FailNow()
	}

	backend, exists := config.Backends[groupName]
	if !exists {
		t.Log(getJSON(config))
		t.Log("Missing backend for grouped service")
		t.FailNow()
	}

	if len(backend.Servers) != 1 {
		t.Log(getJSON(config))
		t.Log("Incorrect number of backend servers on grouped service")
		t.FailNow()
	}

	for _, server := range backend.Servers {
		if server.Weight != 154 {
			t.Log(getJSON(config))
			t.Log("Incorrect weight on grouped service")
			t.FailNow()
		}
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

func getJSON(i interface{}) string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
