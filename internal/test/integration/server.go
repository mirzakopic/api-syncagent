/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/stretchr/testify/require"

	kcpapisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	"github.com/kcp-dev/kcp/sdk/apis/core"
	conditionsv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/third_party/conditions/apis/conditions/v1alpha1"
	"github.com/kcp-dev/kcp/sdk/apis/third_party/conditions/util/conditions"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	"github.com/kcp-dev/kcp/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func NewKDP(t *testing.T) (server framework.RunningServer, rootCluster logicalcluster.Path) {
	t.Helper()

	// setup kcp
	kcpServer := framework.SharedKcpServer(t)
	rootCluster = core.RootCluster.Path()

	framework.WithRootShard()

	// setup KDP base configuration
	resourcesDir := filepath.Join(RepositoryDir(), "..", "deploy", "resources")
	for _, filename := range []string{
		"apiresourceschema-core.kdp.k8c.io_services.yaml",
		"apiresourceschema-docs.kdp.k8c.io_pages.yaml",
		"apiexport-core.kdp.k8c.io.yaml",
		"apiexport-docs.kdp.k8c.io.yaml",
		"workspacetype-kdp-organization.yaml",
		"clusterroles.yaml",
	} {
		framework.Kubectl(t, kcpServer.KubeconfigPath(), "apply", "--filename", filepath.Join(resourcesDir, filename))
	}

	return kcpServer, rootCluster
}

func BindAPIExport(t *testing.T, server framework.RunningServer, cluster logicalcluster.Path, apiExportCluster logicalcluster.Path, apiExportName string) *kcpapisv1alpha1.APIBinding {
	t.Helper()

	cfg := server.BaseConfig(t)
	clusterClient, err := kcpclientset.NewForConfig(cfg)
	require.NoError(t, err, "failed to construct client for server")

	apiBinding := &kcpapisv1alpha1.APIBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", apiExportName),
		},
		Spec: kcpapisv1alpha1.APIBindingSpec{
			Reference: kcpapisv1alpha1.BindingReference{
				Export: &kcpapisv1alpha1.ExportBindingReference{
					Path: apiExportCluster.String(),
					Name: apiExportName,
				},
			},
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	apisClient := clusterClient.ApisV1alpha1().APIBindings().Cluster(cluster)

	created, err := apisClient.Create(ctx, apiBinding, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create APIBinding")

	framework.Eventually(t, func() (bool, string) {
		current, err := apisClient.Get(ctx, created.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("waiting on APIBinding to be ready: %v", err.Error())
		}

		if !conditions.IsTrue(current, conditionsv1alpha1.ReadyCondition) {
			return false, fmt.Sprintf("no %s=True condition yet", conditionsv1alpha1.ReadyCondition)
		}

		return true, ""
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting on APIBinding to be ready")

	t.Logf("Bound APIExport %s:%s in %s.", apiExportCluster, apiExportName, cluster)

	return created
}

// RepositoryDir returns the absolute path of <repo-dir>.
//
// This is copied from kcp's testing framework, sadly the use of runtime.Caller
// makes the function return the root to the kcp pkg path.
func RepositoryDir() string {
	// Caller(0) returns the path to the calling test file rather than the path to this framework file. That
	// precludes assuming how many directories are between the file and the repo root. It's therefore necessary
	// to search in the hierarchy for an indication of a path that looks like the repo root.
	_, sourceFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(sourceFile)
	for {
		// go.mod should always exist in the repo root
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			break
		} else if errors.Is(err, os.ErrNotExist) {
			currentDir, err = filepath.Abs(filepath.Join(currentDir, ".."))
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	return currentDir
}
