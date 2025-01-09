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

package kcp

import (
	"context"
	"fmt"

	"github.com/kcp-dev/logicalcluster/v3"
	"go.uber.org/zap"

	kcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/kontext"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type FilterFunc func(o unstructured.Unstructured) bool

func GetInWorkspaceHandler(client ctrlruntimeclient.Client, gvk schema.GroupVersionKind, log *zap.SugaredLogger, filterFunc FilterFunc) handler.TypedEventHandler[*kcptenancyv1alpha1.Workspace, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj *kcptenancyv1alpha1.Workspace) []reconcile.Request {
		cluster := logicalcluster.From(obj)

		logger := log.With("cluster", string(logicalcluster.From(obj)))
		wsCtx := kontext.WithCluster(ctx, cluster)

		var objects unstructured.UnstructuredList
		objects.SetGroupVersionKind(gvk)

		if err := client.List(wsCtx, &objects); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list objects when eqnqueing from workspace: %w", err))
			return []reconcile.Request{}
		}

		requests := []reconcile.Request{}

		for _, obj := range objects.Items {
			if filterFunc(obj) {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: obj.GetName()}, ClusterName: string(cluster)})
			}
		}

		logger.Debugf("found %d resources to enqueue", len(requests))

		return requests
	})
}
