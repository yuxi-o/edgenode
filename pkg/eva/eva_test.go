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
	"github.com/smartedgemec/appliance-ce/pkg/config"
	"github.com/smartedgemec/appliance-ce/pkg/ela/pb"
	"github.com/smartedgemec/appliance-ce/pkg/eva"
	"os"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
)

var cfgFile = "../../configs/eva.json"

func TestEva(t *testing.T) {
	var cfg eva.Config
	var wg sync.WaitGroup
	dockerTestOn := false

	// Automated tests do not have the application image binaries
	// so we can only run basic tests there.
	// To manually test when you have those images, use the following:
	// go test -args d
	if len(os.Args) == 2 && os.Args[1] == "d" {
		dockerTestOn = true
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

	if err := config.LoadJSONConfig(cfgFile, &cfg); err != nil {
		t.Errorf("LoadJSONConfig() failed: %v", err)
	}
	conn, err := grpc.Dial(cfg.Endpoint, grpc.WithInsecure())
	if err != nil {
		t.Errorf("failed to dial EVA: %v", err)
		cancel()
		return
	}

	if dockerTestOn {
		callDeployAPI(t, conn, "app-test-1", "http://localhost/hello-world.img")
		callDeployAPI(t, conn, "app-test-2", "/var/www/html/busybox.tar.gz")
		callUndeployAPI(t, conn, "app-test-1")
		callUndeployAPI(t, conn, "app-test-2")
	}

	conn.Close()
	cancel()  // stop the EVA running in other thread
	wg.Wait() // wait for the other thread to terminate!
}

func callDeployAPI(t *testing.T, conn *grpc.ClientConn, id string,
	file string) {
	client := pb.NewApplicationDeploymentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	uri := pb.Application_HttpUri{
		HttpUri: &pb.Application_HTTPSource{HttpUri: file},
	}
	app := pb.Application{Id: id, Cores: 1, Memory: 40960, Source: &uri}

	_, err := client.DeployContainer(ctx, &app, grpc.WaitForReady(true))
	if err != nil {
		t.Errorf("DeployContainer failed: %v", err)
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
