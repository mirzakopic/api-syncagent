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

	syncagentv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/syncagent/v1alpha1"
	"github.com/kcp-dev/api-syncagent/test/utils"

	kcpapisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

func TestARSAreCreated(t *testing.T) {
	const (
		apiExportName = "example.com"
	)

	ctx := context.Background()
	ctrlruntime.SetLogger(logr.Discard())

	// setup a test environment in kcp
	orgKubconfig := utils.CreateOrganization(t, ctx, "ars-are-created", apiExportName)

	// start a service cluster
	envtestKubeconfig, envtestClient, _ := utils.RunEnvtest(t, []string{
		"test/crds/crontab.yaml",
	})

	// publish Crontabs
	t.Logf("Publishing CronTabs…")
	pr := &syncagentv1alpha1.PublishedResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "publish-crontabs",
		},
		Spec: syncagentv1alpha1.PublishedResourceSpec{
			Resource: syncagentv1alpha1.SourceResourceDescriptor{
				APIGroup: "example.com",
				Version:  "v1",
				Kind:     "CronTab",
			},
		},
	}

	if err := envtestClient.Create(ctx, pr); err != nil {
		t.Fatalf("Failed to create PublishedResource: %v", err)
	}

	// let the agent do its thing
	utils.RunAgent(ctx, t, "bob", orgKubconfig, envtestKubeconfig, apiExportName)

	// wait for the APIExport to be updated
	t.Logf("Waiting for APIExport to be updated…")
	orgClient := utils.GetClient(t, orgKubconfig)
	apiExportKey := types.NamespacedName{Name: apiExportName}

	var arsName string
	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		apiExport := &kcpapisv1alpha1.APIExport{}
		err = orgClient.Get(ctx, apiExportKey, apiExport)
		if err != nil {
			return false, err
		}

		if len(apiExport.Spec.LatestResourceSchemas) == 0 {
			return false, nil
		}

		arsName = apiExport.Spec.LatestResourceSchemas[0]

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for APIExport to be updated: %v", err)
	}

	// check the APIResourceSchema
	ars := &kcpapisv1alpha1.APIResourceSchema{}
	err = orgClient.Get(ctx, types.NamespacedName{Name: arsName}, ars)
	if err != nil {
		t.Fatalf("APIResourceSchema does not exist: %v", err)
	}
}

func TestARSAreNotUpdated(t *testing.T) {
	const (
		apiExportName = "example.com"
	)

	ctx := context.Background()
	ctrlruntime.SetLogger(logr.Discard())

	// setup a test environment in kcp
	orgKubconfig := utils.CreateOrganization(t, ctx, "ars-are-not-updated", apiExportName)

	// start a service cluster
	envtestKubeconfig, envtestClient, _ := utils.RunEnvtest(t, []string{
		"test/crds/crontab.yaml",
	})

	// publish Crontabs
	t.Logf("Publishing CronTabs…")
	pr := &syncagentv1alpha1.PublishedResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "publish-crontabs",
		},
		Spec: syncagentv1alpha1.PublishedResourceSpec{
			Resource: syncagentv1alpha1.SourceResourceDescriptor{
				APIGroup: "example.com",
				Version:  "v1",
				Kind:     "CronTab",
			},
		},
	}

	if err := envtestClient.Create(ctx, pr); err != nil {
		t.Fatalf("Failed to create PublishedResource: %v", err)
	}

	// let the agent do its thing
	utils.RunAgent(ctx, t, "bob", orgKubconfig, envtestKubeconfig, apiExportName)

	// wait for the APIExport to be updated
	t.Logf("Waiting for APIExport to be updated…")
	orgClient := utils.GetClient(t, orgKubconfig)
	apiExportKey := types.NamespacedName{Name: apiExportName}

	var arsName string
	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		apiExport := &kcpapisv1alpha1.APIExport{}
		err = orgClient.Get(ctx, apiExportKey, apiExport)
		if err != nil {
			return false, err
		}

		if len(apiExport.Spec.LatestResourceSchemas) == 0 {
			return false, nil
		}

		arsName = apiExport.Spec.LatestResourceSchemas[0]

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for APIExport to be updated: %v", err)
	}

	// update the CRD
	t.Logf("Updating CRD (same version, but new schema)…")
	utils.ApplyCRD(t, ctx, envtestClient, "test/crds/crontab-improved.yaml")

	// give the agent some time to do nothing
	time.Sleep(3 * time.Second)

	// validate that the APIExport has *not* changed
	apiExport := &kcpapisv1alpha1.APIExport{}
	err = orgClient.Get(ctx, apiExportKey, apiExport)
	if err != nil {
		t.Fatalf("APIExport disappeared: %v", err)
	}

	if l := len(apiExport.Spec.LatestResourceSchemas); l != 1 {
		t.Fatalf("APIExport should still have 1 resource schema, but has %d.", l)
	}

	if currentName := apiExport.Spec.LatestResourceSchemas[0]; currentName != arsName {
		t.Fatalf("APIExport should still refer to the original ARS %q, but now contains %q.", arsName, currentName)
	}
}

func TestARSDropsAllVersionsExceptTheSelectedOne(t *testing.T) {
	const (
		apiExportName = "example.com"
		theVersion    = "v1"
	)

	ctx := context.Background()
	ctrlruntime.SetLogger(logr.Discard())

	// setup a test environment in kcp
	orgKubconfig := utils.CreateOrganization(t, ctx, "ars-drops-crd-versions", apiExportName)

	// start a service cluster
	envtestKubeconfig, envtestClient, _ := utils.RunEnvtest(t, []string{
		"test/crds/crontab-multi-versions.yaml",
	})

	// publish Crontabs
	t.Logf("Publishing CronTabs…")
	pr := &syncagentv1alpha1.PublishedResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "publish-crontabs",
		},
		Spec: syncagentv1alpha1.PublishedResourceSpec{
			Resource: syncagentv1alpha1.SourceResourceDescriptor{
				APIGroup: "example.com",
				Version:  theVersion,
				Kind:     "CronTab",
			},
		},
	}

	if err := envtestClient.Create(ctx, pr); err != nil {
		t.Fatalf("Failed to create PublishedResource: %v", err)
	}

	// let the agent do its thing
	utils.RunAgent(ctx, t, "bob", orgKubconfig, envtestKubeconfig, apiExportName)

	// wait for the APIExport to be updated
	t.Logf("Waiting for APIExport to be updated…")
	orgClient := utils.GetClient(t, orgKubconfig)
	apiExportKey := types.NamespacedName{Name: apiExportName}

	var arsName string
	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		apiExport := &kcpapisv1alpha1.APIExport{}
		err = orgClient.Get(ctx, apiExportKey, apiExport)
		if err != nil {
			return false, err
		}

		if len(apiExport.Spec.LatestResourceSchemas) == 0 {
			return false, nil
		}

		arsName = apiExport.Spec.LatestResourceSchemas[0]

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for APIExport to be updated: %v", err)
	}

	// check the APIResourceSchema
	ars := &kcpapisv1alpha1.APIResourceSchema{}
	err = orgClient.Get(ctx, types.NamespacedName{Name: arsName}, ars)
	if err != nil {
		t.Fatalf("APIResourceSchema does not exist: %v", err)
	}

	if len(ars.Spec.Versions) != 1 {
		t.Fatalf("Expected only one version to remain in ARS, but found %d.", len(ars.Spec.Versions))
	}

	if name := ars.Spec.Versions[0].Name; name != theVersion {
		t.Fatalf("Expected ARS to contain %q, but contains %q.", theVersion, name)
	}
}
