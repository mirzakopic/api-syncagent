//go:build e2e

/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiresourceschema

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"

	"github.com/kcp-dev/api-syncagent/test/utils"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

func TestARSAreCreated(t *testing.T) {
	ctx := context.Background()
	ctrlruntime.SetLogger(logr.Discard())

	// setup a test environment in kcp
	fooKubconfig := utils.CreateHomeWorkspace(t, ctx, "foo", "my-api")

	// start a service cluster
	envtestKubeconfig, _, _ := utils.RunEnvtest(t)

	t.Run("my subtest", func(t *testing.T) {
		utils.RunAgent(
			ctx,
			t,
			"bob",
			fooKubconfig,
			envtestKubeconfig,
			"my-api",
		)

		time.Sleep(5 * time.Second)
	})

}
