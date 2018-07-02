package integration

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	// "strings"
	"testing"
	"time"

	servicefabric "github.com/containous/traefik-extra-service-fabric"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
)

const numberOfNodeTestServices = 6
const numberOfJavaTestServices = 1
const numberOfJavaStatefulBackends = 1 //Currently don't have a frontend but backend is exposed

var expectedNumberOfFrontends = numberOfJavaTestServices + numberOfNodeTestServices
var expectedNumberOfBackends = numberOfNodeTestServices + numberOfJavaTestServices + numberOfJavaStatefulBackends

var isVerbose bool
var isClusterAlreadyRunning bool
var removeCluster bool

func init() {
	flag.BoolVar(&isVerbose, "sfintegration.verbose", false, "Show the full output of cluster creation scripts")
	flag.BoolVar(&isClusterAlreadyRunning, "sfintegration.clusterrunning", false, "Will skip cluster creation and teardown")
	flag.BoolVar(&removeCluster, "sfintegration.removecluster", false, "Should test leave the cluster running")
}

func TestMain(m *testing.M) {
	flag.Parse()

	if !isClusterAlreadyRunning {
		startTestCluster()
	}

	if !isVerbose {
		log.SetOutput(ioutil.Discard)
	}

	retCode := m.Run()

	if removeCluster {
		stopTestCluster()
	}
	os.Exit(retCode)
}

func TestServiceDiscovery(t *testing.T) {
	provider := servicefabric.Provider{
		BaseProvider:         provider.BaseProvider{},
		ClusterManagementURL: "http://localhost:19080",
		RefreshSeconds:       1,
	}
	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)

	err := provider.Provide(configurationChan, pool, types.Constraints{})
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case actual := <-configurationChan:
		t.Log("Configuration received", toJSON(actual))
		if len(actual.Configuration.Frontends) != expectedNumberOfFrontends {
			t.Errorf("Expected to see %v frontends enabled in the cluster", expectedNumberOfFrontends)
		}
		if len(actual.Configuration.Backends) != expectedNumberOfBackends {
			t.Errorf("Expected to see %v backends enabled in the cluster", expectedNumberOfBackends)
		}
	case <-time.After(time.Second * 12):
		log.Info("Timeout occurred")
		t.Error("Provider failed to return configuration")
	}
}

func TestLabelOverrides(t *testing.T) {
	provider := servicefabric.Provider{
		BaseProvider:         provider.BaseProvider{},
		ClusterManagementURL: "http://localhost:19080",
		RefreshSeconds:       1,
	}

	//Disable the first service
	req, err := http.NewRequest(
		"PUT",
		"http://localhost:19080/Names/node25100/WebService/$/GetProperty?api-version=6.0&IncludeValues=true",
		bytes.NewBufferString(`{
			"PropertyName": "traefik.enable",
			"Value": {
			  "Kind": "String",
			  "Data": "false"
			},
			"CustomTypeId": "LabelType"
		  }`),
	)

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer enableService()
	client := &http.Client{}
	result, err := client.Do(req)

	if err != nil || result.StatusCode != 200 {
		t.Error(err)
		t.FailNow()
	}

	resultString, _ := ioutil.ReadAll(result.Body)
	t.Logf("Disable service response code: %v body: %v", result.StatusCode, string(resultString))

	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)

	err = provider.Provide(configurationChan, pool, types.Constraints{})
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case actual := <-configurationChan:
		t.Log("Configuration received", toJSON(actual))
		if len(actual.Configuration.Frontends) != expectedNumberOfFrontends-1 {
			t.Errorf("Expected to see %v frontends enabled in the cluster", expectedNumberOfFrontends-1)
		}
		if len(actual.Configuration.Backends) != expectedNumberOfBackends-1 {
			t.Errorf("Expected to see %v backends enabled in the cluster", expectedNumberOfBackends-1)
		}
	case <-time.After(time.Second * 12):
		log.Info("Timeout occurred")
		t.Error("Provider failed to return configuration")
	}
}

func TestBackendUrlCorrect(t *testing.T) {
	provider := servicefabric.Provider{
		BaseProvider:         provider.BaseProvider{},
		ClusterManagementURL: "http://localhost:19080",
		RefreshSeconds:       1,
	}

	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)

	err := provider.Provide(configurationChan, pool, types.Constraints{})
	if err != nil {
		t.Error(err)
		return
	}

	select {
	case actual := <-configurationChan:
		t.Log("Configuration received", toJSON(actual))

		client := &http.Client{}
		for backendName, backend := range actual.Configuration.Backends {
			if strings.Contains(backendName, "fabric-testapp-java") {
				t.Log("Skipping java stateful service as currently unsupported, waiting on plugin model see: https://github.com/containous/traefik/pull/3239")
				continue
			}
			for serverName, server := range backend.Servers {
				result, err := client.Get(server.URL)

				if err != nil || result.StatusCode != 200 {
					t.Logf("Server failed to return 200 at %v", server.URL)
					t.Error(err)
				}
				resultString, _ := ioutil.ReadAll(result.Body)
				t.Logf("Success server:%v responded with:%v", serverName, string(resultString))
			}
		}
	case <-time.After(time.Second * 12):
		log.Info("Timeout occurred")
		t.Error("Provider failed to return configuration")
	}
}

func enableService() {
	//Disable the first service
	req, _ := http.NewRequest(
		"PUT",
		"http://localhost:19080/Names/node25100/WebService/$/GetProperty?api-version=6.0&IncludeValues=true",
		bytes.NewBufferString(`{
			"PropertyName": "traefik.enable",
			"Value": {
			  "Kind": "String",
			  "Data": "true"
			},
			"CustomTypeId": "LabelType"
		  }`),
	)

	client := &http.Client{}
	_, _ = client.Do(req)
}
