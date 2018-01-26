package integration

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"

	servicefabric "github.com/containous/traefik-extra-service-fabric"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
)

var isVerbose bool
var isClusterAlreadyRunning bool

func init() {
	flag.BoolVar(&isVerbose, "sfintegration.verbose", false, "Show the full output of cluster creation scripts")
	flag.BoolVar(&isClusterAlreadyRunning, "sfintegration.clusterrunning", false, "Will skip cluster creation and teardown")
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

	if !isClusterAlreadyRunning {
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
		if len(actual.Configuration.Frontends) != 6 {
			t.Error("Expected to see 5 frontends enabled in the cluster")
		}
		if len(actual.Configuration.Backends) != 6 {
			t.Error("Expected to see 5 backends enabled in the cluster")
		}
	case <-time.After(time.Second * 12):
		log.Info("Timeout occurred")
		t.Error("Provider failed to return configuration")
	}
}
