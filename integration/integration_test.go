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

func init() {
	flag.BoolVar(&isVerbose, "sfintegration.verbose", false, "Show the full ouput of cluster creation scripts")
}

func TestMain(m *testing.M) {
	flag.Parse()
	startTestCluster()

	if !isVerbose {
		log.SetOutput(ioutil.Discard)
	}

	retCode := m.Run()

	stopTestCluster()
	os.Exit(retCode)
}

func TestServiceDiscovery(t *testing.T) {
	defer resetTestCluster(t)
	provider := servicefabric.Provider{
		BaseProvider:         provider.BaseProvider{},
		ClusterManagementURL: "http://localhost:19080",
		RefreshSeconds:       3,
	}
	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)
	defer pool.Stop()

	err := provider.Provide(configurationChan, pool, types.Constraints{})
	if err != nil {
		t.Error(err)
		return
	}

	timeout := make(chan string, 1)
	go func() {
		time.Sleep(time.Second * 12)
		timeout <- "Timeout triggered"
	}()

	select {
	case actual := <-configurationChan:
		t.Log(toJSON(actual))
		t.Log("Configuration received", actual)
		//todo: Do some checks!
	case <-timeout:
		t.Error("Provider failed to return configuration")
	}
}
