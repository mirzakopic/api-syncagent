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

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/kcp"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewClusterAwareCluster(config *rest.Config) (cluster.Cluster, error) {
	return cluster.New(config, func(o *cluster.Options) {
		// o.MapperProvider = kcp.NewClusterAwareMapperProvider
		o.NewClient = kcp.NewClusterAwareClient
		o.NewCache = kcp.NewClusterAwareCache
		o.NewAPIReader = kcp.NewClusterAwareAPIReader
	})
}

func ConnectToVirtualWorkspace(ctx context.Context, mgr manager.Manager, restConfig *rest.Config, apiExport string, hostOverride string) (cluster.Cluster, error) {
	exportConfig, err := RestConfigForAPIExport(ctx, restConfig, apiExport, hostOverride)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual workspace URL for %q: %w", apiExport, err)
	}

	kcpCluster, err := NewClusterAwareCluster(exportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster aware cluster: %w", err)
	}

	if err := mgr.Add(kcpCluster); err != nil {
		return nil, fmt.Errorf("failed to add cluster to root manager: %w", err)
	}

	return kcpCluster, nil
}
