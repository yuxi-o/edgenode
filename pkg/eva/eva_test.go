// Copyright 2019 Intel Corporation and Smart-Edge.com, Inc. All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE
// This test file uses the Go testing framework, while rest of the
// test code in OpenNESS uses Ginko / Gomeka.
// This file needs to be updated to match the other test files.
// (Or other test files updated to match this one)

package eva_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/smartedgemec/appliance-ce/internal/authtest"
	"github.com/smartedgemec/appliance-ce/pkg/config"
	"github.com/smartedgemec/appliance-ce/pkg/ela/pb"
	"github.com/smartedgemec/appliance-ce/pkg/eva"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	cfgFile        = "testdata/eva.json"
	transportCreds credentials.TransportCredentials
)

// TODO: refactor test to use ginkgo/gomega
func TestEva(t *testing.T) {
	var cfg eva.Config
	var wg sync.WaitGroup
	dockerTestOn := false
	libvirtTestOn := false

	if err := config.LoadJSONConfig(cfgFile, &cfg); err != nil {
		t.Errorf("LoadJSONConfig() failed: %v", err)
	}

	if err := os.MkdirAll(cfg.CertsDir, 0700); err != nil {
		t.Errorf("Creating temp directory for certs failed: %v", err)
	}
	defer os.RemoveAll(cfg.CertsDir)

	transportCreds = prepareCerts(t, cfg.CertsDir)

	// Automated tests do not have the application image binaries
	// so we can only run basic tests there.
	// To manually test when you have those images, use the following:
	// go test -args b
	if len(os.Args) == 2 {
		if os.Args[1] == "d" {
			dockerTestOn = true
		} else if os.Args[1] == "v" {
			libvirtTestOn = true
		} else if os.Args[1] == "b" {
			dockerTestOn = true
			libvirtTestOn = true
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		err := eva.Run(ctx, cfgFile)
		wg.Done()
		if err != nil {
			t.Errorf("eva.Run() failed: %v", err)
		}
	}()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(),
		10*time.Second)
	defer cancelTimeout()
	conn, err := grpc.DialContext(ctxTimeout, cfg.Endpoint,
		grpc.WithTransportCredentials(transportCreds), grpc.WithBlock())

	if err != nil {
		t.Errorf("failed to dial EVA: %v", err)
		cancel()
		return
	}
	defer conn.Close()

	if dockerTestOn {
		fmt.Println("-----------APP_TEST_1------------------------------------")
		callDockerDeploy(t, conn, "app-test-1",
			"http://localhost/hello-world.img")
		fmt.Println("-----------APP_TEST_2------------------------------------")
		callDockerDeploy(t, conn, "app-test-2", "/var/www/html/busybox.tar.gz")
		fmt.Println("-----------APP_TEST_1-U----------------------------------")
		callUndeployAPI(t, conn, "app-test-1")
		fmt.Println("-----------APP_TEST_2-U----------------------------------")
		callUndeployAPI(t, conn, "app-test-2")

		fmt.Println("---------LIFECYCLE---------------------------------------")
		testLifecycleAPI(t, conn, "hello-world-app",
			"/var/www/html/hello-world.tar.gz")
	}

	if libvirtTestOn {
		callLibvirtDeploy(t, conn, "app-test-vm-1",
			"http://localhost/freedos-1.0.7z")
		fmt.Println("--------------------------------------------------")
		callLibvirtDeploy(t, conn, "app-test-vm-2",
			"http://localhost/freedos-1.0.7z")
		fmt.Println("--------------------------------------------------")
		callUndeployAPI(t, conn, "app-test-vm-1")
		fmt.Println("--------------------------------------------------")
		callUndeployAPI(t, conn, "app-test-vm-2")
		fmt.Println("--------------------------------------------------")

	}

	cancel()  // stop the EVA running in other thread
	wg.Wait() // wait for the other thread to terminate!
}

// Prepare certificates for test
func prepareCerts(t *testing.T,
	certsDir string) credentials.TransportCredentials {

	err := authtest.EnrollStub(certsDir)
	if err != nil {
		t.Errorf("EnrollStub failed: %v", err)
	}
	transportCreds, err := authtest.ClientCredentialsStub()
	if err != nil {
		t.Errorf("ClientCredentialsStub failed: %v", err)
	}
	return transportCreds
}

func callDockerDeploy(t *testing.T, conn *grpc.ClientConn, id string,
	file string) {

	client := pb.NewApplicationDeploymentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	uri := pb.Application_HttpUri{
		HttpUri: &pb.Application_HTTPSource{HttpUri: file},
	}
	app := pb.Application{Id: id, Cores: 2, Memory: 40960, Source: &uri}

	_, err := client.DeployContainer(ctx, &app, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("DeployContainer failed: %v", err)
	}

	cancel()
}

func callLibvirtDeploy(t *testing.T, conn *grpc.ClientConn, id string,
	file string) {

	client := pb.NewApplicationDeploymentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	uri := pb.Application_HttpUri{
		HttpUri: &pb.Application_HTTPSource{HttpUri: file},
	}
	app := pb.Application{Id: id, Cores: 1, Memory: 40960, Source: &uri}

	_, err := client.DeployVM(ctx, &app, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("DeployVM failed: %v", err)
	}
	cancel()
}

func callUndeployAPI(t *testing.T, conn *grpc.ClientConn, id string) {
	client := pb.NewApplicationDeploymentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	app := pb.ApplicationID{Id: id}

	_, err := client.Undeploy(ctx, &app, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("Undeploy failed: %v", err)
	}
	cancel()
}

// Deploy application in container from given image; start, restart and stop
// container; undeploy application.
func testLifecycleAPI(t *testing.T, conn *grpc.ClientConn, id string,
	image string) {

	var err error

	callDockerDeploy(t, conn, id, image) //"/var/www/html/hello-world.tar.gz")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	alsClient := pb.NewApplicationLifecycleServiceClient(conn)

	lc := pb.LifecycleCommand{Id: id}

	_, err = alsClient.Start(ctx, &lc, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("StartContainer failed: %v", err)
	}

	_, err = alsClient.Restart(ctx, &lc, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("StartContainer failed: %v", err)
	}

	_, err = alsClient.Stop(ctx, &lc, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("StartContainer failed: %v", err)
	}

	callUndeployAPI(t, conn, id)
}
