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

package kcp

import (
	"context"
	"fmt"
	"net/url"

	kcpdevv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// toRootClusterConfig will ensure that the given host (URL)
// for the kcp apiserver points to the root workspace.
func toRootClusterConfig(cfg *rest.Config) (*rest.Config, error) {
	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}

	u.Path = "/clusters/root"

	newCfg := rest.CopyConfig(cfg)
	newCfg.Host = u.String()

	return newCfg, nil
}

// RestConfigForAPIExport returns a *rest.Config properly configured to communicate
// with the endpoint for the APIExport's virtual workspace.
func RestConfigForAPIExport(ctx context.Context, cfg *rest.Config, apiExportName string, hostOverride string) (*rest.Config, error) {
	scheme := runtime.NewScheme()
	if err := kcpdevv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding apis.kcp.dev/v1alpha1 to scheme: %w", err)
	}

	newCfg, err := toRootClusterConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL %q: %w", cfg.Host, err)
	}

	apiExportClient, err := ctrlruntimeclient.New(newCfg, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("error creating APIExport client: %w", err)
	}

	var apiExport kcpdevv1alpha1.APIExport

	if err := apiExportClient.Get(ctx, types.NamespacedName{Name: apiExportName}, &apiExport); err != nil {
		return nil, fmt.Errorf("error getting APIExport %q: %w", apiExportName, err)
	}

	//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
	if len(apiExport.Status.VirtualWorkspaces) < 1 {
		return nil, fmt.Errorf("APIExport %q status.virtualWorkspaces is empty", apiExportName)
	}

	//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
	vwUrl, err := url.Parse(apiExport.Status.VirtualWorkspaces[0].URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing VirtualWorkspace URL: %w", err)
	}

	if hostOverride != "" {
		vwUrl.Host = hostOverride
	}

	newCfg.Host = vwUrl.String()

	return newCfg, nil
}
