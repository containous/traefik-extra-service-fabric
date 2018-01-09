package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	servicefabric "github.com/containous/traefik-extra-service-fabric"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
)

func TestServiceDiscovery(t *testing.T) {
	t.Log("Starting test")
	dir, err := os.Getwd()
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("Running commands in directory: %v", dir)

	cmd := exec.Command("/bin/sh", filepath.Join(dir, "run.sh"))
	cmd.Dir = dir
	t.Log("Starting test cluster")
	out, err := cmd.Output()

	t.Logf("Cluster script ran with output: %v", string(out))

	if err != nil {
		t.Error(err)
		return
	}

	provider := servicefabric.Provider{
		BaseProvider:         provider.BaseProvider{},
		ClusterManagementURL: "http://localhost:19080",
		RefreshSeconds:       3,
	}
	configurationChan := make(chan types.ConfigMessage)
	ctx := context.Background()
	pool := safe.NewPool(ctx)
	defer pool.Stop()

	err = provider.Provide(configurationChan, pool, types.Constraints{})
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
		t.Log("Configuration received", actual)
		//todo: Do some checks!
	case <-timeout:
		t.Error("Provider failed to return configuration")
	}

}
