package servicefabric

import (
	"encoding/json"
	"testing"

	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
	sf "github.com/jjcollinge/servicefabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServicesPresentInConfig tests that the basic services provide by SF
// are return in the configuration object
func TestBuildConfigurationServicesPresentInConfig(t *testing.T) {
	provider := Provider{}

	services := []ServiceItemExtended{
		{
			ServiceItem: sf.ServiceItem{
				HasPersistedState: true,
				HealthState:       "Ok",
				ID:                "TestApplication/TestService",
				IsServiceGroup:    false,
				ManifestVersion:   "1.0.0",
				Name:              "fabric:/TestApplication/TestService",
				ServiceKind:       kindStateless,
				ServiceStatus:     "Active",
				TypeName:          "TestServiceType",
			},
			Application: sf.ApplicationItem{
				HealthState: "Ok",
				ID:          "TestApplication",
				Name:        "fabric:/TestApplication",
				Parameters: []*sf.AppParameter{
					{
						Key:   "TraefikPublish",
						Value: "fabric:/TestApplication/TestService",
					},
				},
				Status:      "Ready",
				TypeName:    "TestApplicationType",
				TypeVersion: "1.0.0",
			},
			Partitions: []PartitionItemExtended{
				{
					PartitionItem: sf.PartitionItem{
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
						ServiceKind:          kindStateless,
						TargetReplicaSetSize: 1,
					},
					Replicas: nil,
					Instances: []sf.InstanceItem{
						{
							ReplicaItemBase: &sf.ReplicaItemBase{
								Address:                      "{\"Endpoints\":{\"\":\"http://localhost:8081\"}}",
								HealthState:                  "Ok",
								LastInBuildDurationInSeconds: "3",
								NodeName:                     "_Node_0",
								ReplicaRole:                  "",
								ReplicaStatus:                "Ready",
								ServiceKind:                  kindStateless,
							},
							ID: "1",
						},
					},
				},
			},
			Labels: map[string]string{
				"traefik.enable": "true",
			},
		},
	}

	config, err := provider.buildConfiguration(services)
	require.NoError(t, err)

	require.NotNil(t, config, "configuration")

	expected := &types.Configuration{
		Backends: map[string]*types.Backend{
			"fabric:/TestApplication/TestService": {
				Servers: map[string]types.Server{
					"1": {
						URL:    "http://localhost:8081",
						Weight: 1,
					},
				},
			},
		},
		Frontends: map[string]*types.Frontend{
			"frontend-fabric:/TestApplication/TestService": {
				Backend:        "fabric:/TestApplication/TestService",
				PassHostHeader: true,
			},
		},
	}
	assert.Equal(t, expected, config)
}

func TestBuildConfigurationStateful(t *testing.T) {
	provider := Provider{}

	testCases := []struct {
		desc     string
		labels   map[string]string
		expected *types.Configuration
	}{
		{
			desc: "without frontend.rule label",
			labels: map[string]string{
				label.TraefikEnable: "true",
			},
			expected: &types.Configuration{
				Backends: map[string]*types.Backend{
					"fabric-TestApplication-TestServicebce46a8c-b62d-4996-89dc-7ffc00a96902": {
						LoadBalancer: &types.LoadBalancer{
							Method: "drr",
						},
						Servers: map[string]types.Server{
							"131496928082309293": {
								URL:    "http://localhost:8081",
								Weight: 1,
							},
						},
					},
				},
				Frontends: map[string]*types.Frontend{},
			},
		},
		{
			desc: "with label frontend.rule.partition.$partitionId",
			labels: map[string]string{
				label.TraefikEnable:                                                    "true",
				"traefik.frontend.rule.partition.bce46a8c-b62d-4996-89dc-7ffc00a96902": "HeadersRegexp: username, ^b",
			},
			expected: &types.Configuration{
				Backends: map[string]*types.Backend{
					"fabric-TestApplication-TestServicebce46a8c-b62d-4996-89dc-7ffc00a96902": {
						LoadBalancer: &types.LoadBalancer{
							Method: "drr",
						},
						Servers: map[string]types.Server{
							"131496928082309293": {
								URL:    "http://localhost:8081",
								Weight: 1,
							},
						},
					},
				},
				Frontends: map[string]*types.Frontend{
					"fabric:/TestApplication/TestService/bce46a8c-b62d-4996-89dc-7ffc00a96902": {
						Backend: "fabric-TestApplication-TestServicebce46a8c-b62d-4996-89dc-7ffc00a96902",
						Routes: map[string]types.Route{
							"default": {
								Rule: "HeadersRegexp: username, ^b",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			services := []ServiceItemExtended{
				{
					ServiceItem: sf.ServiceItem{
						HasPersistedState: true,
						HealthState:       "Ok",
						ID:                "TestApplication/TestService",
						IsServiceGroup:    false,
						ManifestVersion:   "1.0.0",
						Name:              "fabric:/TestApplication/TestService",
						ServiceKind:       kindStateful,
						ServiceStatus:     "Active",
						TypeName:          "TestServiceType",
					},
					Partitions: []PartitionItemExtended{
						{
							PartitionItem: sf.PartitionItem{
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
								ServiceKind:          kindStateful,
								TargetReplicaSetSize: 1,
							},
							Replicas: []sf.ReplicaItem{
								{
									ReplicaItemBase: &sf.ReplicaItemBase{
										Address:                      `{"Endpoints":{"":"http://localhost:8081"}}`,
										HealthState:                  "Ok",
										LastInBuildDurationInSeconds: "1",
										NodeName:                     "_Node_0",
										ReplicaRole:                  "Primary",
										ReplicaStatus:                "Ready",
										ServiceKind:                  kindStateful,
									},
									ID: "131496928082309293",
								},
							},
						},
					},
					Labels: test.labels,
				},
			}

			config, err := provider.buildConfiguration(services)
			require.NoError(t, err)

			require.NotNil(t, config, "configuration")

			assert.Equal(t, test.expected, config)
		})
	}
}

// nolint: gocyclo
func TestBuildConfigurationFrontendLabelConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		labels   map[string]string
		validate func(*testing.T, *types.Frontend)
	}{
		{
			desc: "Has passHostHeader enabled",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendPassHostHeader: "true",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.True(t, f.PassHostHeader)
			},
		},
		{
			desc: "Has passHostHeader disabled",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendPassHostHeader: "false",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.False(t, f.PassHostHeader)
			},
		},
		{
			desc: "Has whitelistSourceRange set (deprecated)",
			labels: map[string]string{
				label.TraefikEnable:                       "true",
				label.TraefikFrontendWhitelistSourceRange: "10.0.0.1, 10.0.0.2",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				expected := &types.WhiteList{
					SourceRange: []string{"10.0.0.1", "10.0.0.2"},
				}
				assert.EqualValues(t, expected, f.WhiteList)
			},
		},
		{
			desc: "Has WhiteList set ",
			labels: map[string]string{
				label.TraefikEnable:                            "true",
				label.TraefikFrontendWhiteListSourceRange:      "10.0.0.1, 10.0.0.2",
				label.TraefikFrontendWhiteListUseXForwardedFor: "true",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				expected := &types.WhiteList{
					SourceRange:      []string{"10.0.0.1", "10.0.0.2"},
					UseXForwardedFor: true,
				}
				assert.EqualValues(t, expected, f.WhiteList)
			},
		},
		{
			desc: "Has priority set",
			labels: map[string]string{
				label.TraefikEnable:           "true",
				label.TraefikFrontendPriority: "13",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.Equal(t, f.Priority, 13)
			},
		},
		{
			desc: "Has basicAuth set",
			labels: map[string]string{
				label.TraefikEnable:            "true",
				label.TraefikFrontendAuthBasic: "USER1:HASH1, USER1:HASH1",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.Len(t, f.BasicAuth, 2)

				expected := []string{"USER1:HASH1", "USER1:HASH1"}
				assert.EqualValues(t, expected, f.BasicAuth)
			},
		},
		{
			desc: "Has frontend entry points set",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendEntryPoints: "Barry, Bob",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.Len(t, f.EntryPoints, 2)

				expected := []string{"Barry", "Bob"}
				assert.EqualValues(t, expected, f.EntryPoints)
			},
		},
		{
			desc: "Has passTLSCert enabled",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendPassTLSCert: "true",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.True(t, f.PassTLSCert)
			},
		},
		{
			desc: "Has passTLSCert disabled",
			labels: map[string]string{
				label.TraefikEnable:              "true",
				label.TraefikFrontendPassTLSCert: "false",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.False(t, f.PassTLSCert)
			},
		},
		{
			desc: "Has rule set",
			labels: map[string]string{
				label.TraefikEnable:                    "true",
				label.TraefikFrontendRule + ".default": "Path: /",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				assert.Len(t, f.Routes, 1)

				expected := map[string]types.Route{
					label.TraefikFrontendRule + ".default": {
						Rule: "Path: /",
					},
				}
				assert.Equal(t, expected, f.Routes)
			},
		},
		{
			desc: "Has Headers set",
			labels: map[string]string{
				label.TraefikEnable:                          "true",
				label.TraefikFrontendSSLRedirect:             "true",
				label.TraefikFrontendSSLTemporaryRedirect:    "true",
				label.TraefikFrontendSSLHost:                 "bob.bob.com",
				label.TraefikFrontendSTSSeconds:              "1337",
				label.TraefikFrontendSTSIncludeSubdomains:    "true",
				label.TraefikFrontendSTSPreload:              "true",
				label.TraefikFrontendForceSTSHeader:          "true",
				label.TraefikFrontendFrameDeny:               "true",
				label.TraefikFrontendCustomFrameOptionsValue: "SAMEORIGIN",
				label.TraefikFrontendContentTypeNosniff:      "true",
				label.TraefikFrontendBrowserXSSFilter:        "true",
				label.TraefikFrontendContentSecurityPolicy:   "plugin-types image/png application/pdf; sandbox",
				label.TraefikFrontendPublicKey:               "somekeydata",
				label.TraefikFrontendReferrerPolicy:          "same-origin",
				label.TraefikFrontendIsDevelopment:           "true",
				label.TraefikFrontendAllowedHosts:            "host1, host2",
				label.TraefikFrontendSSLProxyHeaders:         "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				expected := &types.Headers{
					SSLProxyHeaders: map[string]string{
						"X-Testing":  "testing",
						"X-Testing2": "testing2",
					},
					AllowedHosts:            []string{"host1", "host2"},
					HostsProxyHeaders:       nil,
					SSLRedirect:             true,
					SSLTemporaryRedirect:    true,
					SSLHost:                 "bob.bob.com",
					STSSeconds:              1337,
					STSIncludeSubdomains:    true,
					STSPreload:              true,
					ForceSTSHeader:          true,
					FrameDeny:               true,
					CustomFrameOptionsValue: "SAMEORIGIN",
					ContentTypeNosniff:      true,
					BrowserXSSFilter:        true,
					ContentSecurityPolicy:   "plugin-types image/png application/pdf; sandbox",
					PublicKey:               "somekeydata",
					ReferrerPolicy:          "same-origin",
					IsDevelopment:           true,
				}
				assert.Equal(t, expected, f.Headers)
			},
		},
		{
			desc: "Has RequestHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                 "true",
				label.TraefikFrontendRequestHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				require.NotNil(t, f.Headers, "headers")

				expected := map[string]string{
					"X-Testing":  "testing",
					"X-Testing2": "testing2",
				}
				assert.Equal(t, expected, f.Headers.CustomRequestHeaders)
			},
		},
		{
			desc: "Has ResponseHeaders set",
			labels: map[string]string{
				label.TraefikEnable:                  "true",
				label.TraefikFrontendResponseHeaders: "X-Testing:testing||X-Testing2:testing2",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				require.NotNil(t, f.Headers, "headers")

				expected := map[string]string{
					"X-Testing":  "testing",
					"X-Testing2": "testing2",
				}
				assert.Equal(t, expected, f.Headers.CustomResponseHeaders)
			},
		},
		{
			desc: "Has Redirect on entry point ",
			labels: map[string]string{
				label.TraefikEnable:                      "true",
				label.TraefikFrontendRedirectEntryPoint:  "foo",
				label.TraefikFrontendRedirectPermanent:   "true",
				label.TraefikFrontendRedirectRegex:       "nope",
				label.TraefikFrontendRedirectReplacement: "nope",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				expected := &types.Redirect{
					EntryPoint: "foo",
					Permanent:  true,
				}
				assert.Equal(t, expected, f.Redirect)
			},
		},
		{
			desc: "Has Redirect with regex ",
			labels: map[string]string{
				label.TraefikEnable:                      "true",
				label.TraefikFrontendRedirectPermanent:   "true",
				label.TraefikFrontendRedirectRegex:       "(.*)",
				label.TraefikFrontendRedirectReplacement: "$1",
			},
			validate: func(t *testing.T, f *types.Frontend) {
				expected := &types.Redirect{
					Regex:       "(.*)",
					Replacement: "$1",
					Permanent:   true,
				}
				assert.Equal(t, expected, f.Redirect)
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			provider := Provider{}

			services := []ServiceItemExtended{
				{
					ServiceItem: sf.ServiceItem{
						HasPersistedState: true,
						HealthState:       "Ok",
						ID:                "TestApplication/TestService",
						IsServiceGroup:    false,
						ManifestVersion:   "1.0.0",
						Name:              "fabric:/TestApplication/TestService",
						ServiceKind:       kindStateless,
						ServiceStatus:     "Active",
						TypeName:          "TestServiceType",
					},
					Application: sf.ApplicationItem{
						HealthState: "Ok",
						ID:          "TestApplication",
						Name:        "fabric:/TestApplication",
						Parameters: []*sf.AppParameter{
							{
								Key:   "TraefikPublish",
								Value: "fabric:/TestApplication/TestService",
							},
						},
						Status:      "Ready",
						TypeName:    "TestApplicationType",
						TypeVersion: "1.0.0",
					},
					Partitions: []PartitionItemExtended{
						{
							PartitionItem: sf.PartitionItem{
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
								ServiceKind:          kindStateless,
								TargetReplicaSetSize: 1,
							},
							Replicas: nil,
							Instances: []sf.InstanceItem{
								{
									ReplicaItemBase: &sf.ReplicaItemBase{
										Address:                      "{\"Endpoints\":{\"\":\"http://localhost:8081\"}}",
										HealthState:                  "Ok",
										LastInBuildDurationInSeconds: "3",
										NodeName:                     "_Node_0",
										ReplicaRole:                  "",
										ReplicaStatus:                "Ready",
										ServiceKind:                  kindStateless,
									},
									ID: "1",
								},
							},
						},
					},
					Labels: test.labels,
				},
			}

			config, err := provider.buildConfiguration(services)
			require.NoError(t, err)

			assert.NotEmpty(t, config.Frontends, "No frontends present in the configuration")

			for fname, frontend := range config.Frontends {
				require.NotNil(t, frontend, "Frontend %s is nil", fname)

				test.validate(t, frontend)
				if t.Failed() {
					t.Log(getJSON(frontend))
				}
			}
		})
	}
}

// nolint: gocyclo
func TestBuildConfigurationBackendLabelConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		labels   map[string]string
		validate func(*testing.T, *types.Backend)
	}{
		{
			desc: "Has DRR LoadBalancer",
			labels: map[string]string{
				label.TraefikEnable:                    "true",
				label.TraefikBackendLoadBalancerMethod: "drr",
			},
			validate: func(t *testing.T, b *types.Backend) {
				require.NotNil(t, b.LoadBalancer, "LoadBalancer")
				assert.Equal(t, "drr", b.LoadBalancer.Method)
			},
		},
		{
			desc: "Has health check set",
			labels: map[string]string{
				label.TraefikEnable:                     "true",
				label.TraefikBackendHealthCheckPath:     "/hc",
				label.TraefikBackendHealthCheckPort:     "9000",
				label.TraefikBackendHealthCheckInterval: "1337s",
			},
			validate: func(t *testing.T, b *types.Backend) {
				expected := &types.HealthCheck{
					Path:     "/hc",
					Port:     9000,
					Interval: "1337s",
				}
				assert.Equal(t, expected, b.HealthCheck)
			},
		},
		{
			desc: "Has circuit breaker set",
			labels: map[string]string{
				label.TraefikEnable:                          "true",
				label.TraefikBackendCircuitBreakerExpression: "NetworkErrorRatio() > 0.5",
			},
			validate: func(t *testing.T, b *types.Backend) {
				expected := &types.CircuitBreaker{
					Expression: "NetworkErrorRatio() > 0.5",
				}
				assert.Equal(t, expected, b.CircuitBreaker)
			},
		},
		{
			desc: "Has stickiness cookie set",
			labels: map[string]string{
				label.TraefikEnable:                        "true",
				label.TraefikBackendLoadBalancerStickiness: "true",
			},
			validate: func(t *testing.T, b *types.Backend) {
				require.NotNil(t, b.LoadBalancer, "LoadBalancer")
				assert.NotNil(t, b.LoadBalancer.Stickiness, "Stickiness")
			},
		},
		{
			desc: "Has maxconn amount and extractor func",
			labels: map[string]string{
				label.TraefikEnable:                      "true",
				label.TraefikBackendMaxConnAmount:        "1337",
				label.TraefikBackendMaxConnExtractorFunc: "request.header.TEST_HEADER",
			},
			validate: func(t *testing.T, b *types.Backend) {
				expected := &types.MaxConn{
					Amount:        1337,
					ExtractorFunc: "request.header.TEST_HEADER",
				}
				assert.Equal(t, expected, b.MaxConn)
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			provider := Provider{}

			services := []ServiceItemExtended{
				{
					ServiceItem: sf.ServiceItem{
						HasPersistedState: true,
						HealthState:       "Ok",
						ID:                "TestApplication/TestService",
						IsServiceGroup:    false,
						ManifestVersion:   "1.0.0",
						Name:              "fabric:/TestApplication/TestService",
						ServiceKind:       kindStateless,
						ServiceStatus:     "Active",
						TypeName:          "TestServiceType",
					},
					Application: sf.ApplicationItem{
						HealthState: "Ok",
						ID:          "TestApplication",
						Name:        "fabric:/TestApplication",
						Parameters: []*sf.AppParameter{
							{
								Key:   "TraefikPublish",
								Value: "fabric:/TestApplication/TestService",
							},
						},
						Status:      "Ready",
						TypeName:    "TestApplicationType",
						TypeVersion: "1.0.0",
					},
					Partitions: []PartitionItemExtended{
						{
							PartitionItem: sf.PartitionItem{
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
								ServiceKind:          kindStateless,
								TargetReplicaSetSize: 1,
							},
							Replicas: nil,
							Instances: []sf.InstanceItem{
								{
									ReplicaItemBase: &sf.ReplicaItemBase{
										Address:                      "{\"Endpoints\":{\"\":\"http://localhost:8081\"}}",
										HealthState:                  "Ok",
										LastInBuildDurationInSeconds: "3",
										NodeName:                     "_Node_0",
										ReplicaRole:                  "",
										ReplicaStatus:                "Ready",
										ServiceKind:                  kindStateless,
									},
									ID: "1",
								},
							},
						},
					},
					Labels: test.labels,
				},
			}

			config, err := provider.buildConfiguration(services)
			require.NoError(t, err)

			assert.NotEmpty(t, config.Backends, "No backends present in the configuration")

			for bname, backend := range config.Backends {
				require.NotNil(t, backend, "Backend %s is nil", bname)

				test.validate(t, backend)
				if t.Failed() {
					t.Log(getJSON(backend))
				}
			}
		})
	}
}

func TestBuildConfigurationGroupedServicesFrontends(t *testing.T) {
	services := []ServiceItemExtended{
		{
			ServiceItem: sf.ServiceItem{
				HasPersistedState: true,
				HealthState:       "Ok",
				ID:                "TestApplication/TestService",
				IsServiceGroup:    false,
				ManifestVersion:   "1.0.0",
				Name:              "fabric:/TestApplication/TestService",
				ServiceKind:       kindStateless,
				ServiceStatus:     "Active",
				TypeName:          "TestServiceType",
			},
			Application: sf.ApplicationItem{
				HealthState: "Ok",
				ID:          "TestApplication",
				Name:        "fabric:/TestApplication",
				Parameters: []*sf.AppParameter{
					{
						Key:   "TraefikPublish",
						Value: "fabric:/TestApplication/TestService",
					},
				},
				Status:      "Ready",
				TypeName:    "TestApplicationType",
				TypeVersion: "1.0.0",
			},
			Partitions: []PartitionItemExtended{
				{
					PartitionItem: sf.PartitionItem{
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
						ServiceKind:          kindStateless,
						TargetReplicaSetSize: 1,
					},
					Replicas: nil,
					Instances: []sf.InstanceItem{
						{
							ReplicaItemBase: &sf.ReplicaItemBase{
								Address:                      "{\"Endpoints\":{\"\":\"http://localhost:8081\"}}",
								HealthState:                  "Ok",
								LastInBuildDurationInSeconds: "3",
								NodeName:                     "_Node_0",
								ReplicaRole:                  "",
								ReplicaStatus:                "Ready",
								ServiceKind:                  kindStateless,
							},
							ID: "1",
						},
					},
				},
			},
			Labels: map[string]string{
				label.TraefikEnable:  "true",
				traefikSFGroupName:   "groupedbackends",
				traefikSFGroupWeight: "154",
			},
		},
	}

	provider := Provider{}

	config, err := provider.buildConfiguration(services)
	require.NoError(t, err)

	require.NotNil(t, config, "configuration")

	expectedFrontends := map[string]*types.Frontend{
		"frontend-fabric:/TestApplication/TestService": {
			Backend:        "fabric:/TestApplication/TestService",
			PassHostHeader: true,
		},
		"groupedbackends": {
			Backend:  "groupedbackends",
			Priority: 50,
		},
	}

	assert.Equal(t, expectedFrontends, config.Frontends)
}

func TestBuildConfigurationGroupedServicesBackends(t *testing.T) {
	services := []ServiceItemExtended{
		{
			ServiceItem: sf.ServiceItem{
				HasPersistedState: true,
				HealthState:       "Ok",
				ID:                "TestApplication/TestService",
				IsServiceGroup:    false,
				ManifestVersion:   "1.0.0",
				Name:              "fabric:/TestApplication/TestService",
				ServiceKind:       kindStateless,
				ServiceStatus:     "Active",
				TypeName:          "TestServiceType",
			},
			Application: sf.ApplicationItem{
				HealthState: "Ok",
				ID:          "TestApplication",
				Name:        "fabric:/TestApplication",
				Parameters: []*sf.AppParameter{
					{
						Key:   "TraefikPublish",
						Value: "fabric:/TestApplication/TestService",
					},
				},
				Status:      "Ready",
				TypeName:    "TestApplicationType",
				TypeVersion: "1.0.0",
			},
			Partitions: []PartitionItemExtended{
				{
					PartitionItem: sf.PartitionItem{
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
						ServiceKind:          kindStateless,
						TargetReplicaSetSize: 1,
					},
					Replicas: nil,
					Instances: []sf.InstanceItem{
						{
							ReplicaItemBase: &sf.ReplicaItemBase{
								Address:                      "{\"Endpoints\":{\"\":\"http://localhost:8081\"}}",
								HealthState:                  "Ok",
								LastInBuildDurationInSeconds: "3",
								NodeName:                     "_Node_0",
								ReplicaRole:                  "",
								ReplicaStatus:                "Ready",
								ServiceKind:                  kindStateless,
							},
							ID: "1",
						},
					},
				},
			},
			Labels: map[string]string{
				label.TraefikEnable:  "true",
				traefikSFGroupName:   "groupedbackends",
				traefikSFGroupWeight: "154",
			},
		},
	}

	provider := Provider{}

	config, err := provider.buildConfiguration(services)
	require.NoError(t, err)

	require.NotNil(t, config, "configuration")

	expected := map[string]*types.Backend{
		"fabric:/TestApplication/TestService": {
			Servers: map[string]types.Server{
				"1": {
					URL:    "http://localhost:8081",
					Weight: 1,
				},
			},
		},
		"groupedbackends": {
			Servers: map[string]types.Server{
				"TestApplication/TestService-1": {
					URL:    "http://localhost:8081",
					Weight: 154,
				},
			},
		},
	}
	assert.Equal(t, expected, config.Backends)
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
					ServiceKind:                  kindStateful,
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
					ServiceKind:                  kindStateful,
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

func getJSON(i interface{}) string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
