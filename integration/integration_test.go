package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	servicefabric "github.com/containous/traefik-extra-service-fabric"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
)

func TestMain(m *testing.M) {
	startTestCluster()
	retCode := m.Run()
	stopTestCluster()
	os.Exit(retCode)
}

func startTestCluster() {
	err := runScript("run.sh", time.Second*180)
	if err != nil {
		panic("Failed to start cluster")
	}
}

func stopTestCluster() {
	err := runScript("stop.sh", time.Second*30)
	if err != nil {
		panic("Failed to stop cluster")
	}
}

func resetTestCluster() {
	err := runScript("reset.sh", time.Second*60)
	if err != nil {
		panic("Failed to reset cluster")
	}
}

func TestServiceDiscovery(t *testing.T) {
	defer resetTestCluster()
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

func runScript(scriptName string, timeout time.Duration) error {
	resultChan := make(chan int, 1)

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", filepath.Join(dir, scriptName))

	go func() {
		log.Infof("Running commands in directory: %v", dir)

		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()

		if err != nil {
			log.Infof("Failed running script: %v", err)
			panic(err)
		}

		resultChan <- 1
	}()

	timeoutChan := make(chan int, 1)
	go func() {
		time.Sleep(timeout)
		timeoutChan <- 1
	}()

	select {
	case <-resultChan:
		return nil
	case <-timeoutChan:
		cmd.Process.Kill()
		return fmt.Errorf("Timeout waiting for script after: %v", timeout)
	}
}

func toJSON(i interface{}) string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		panic("Failed to marshal json")
	}

	return string(jsonBytes)
}
