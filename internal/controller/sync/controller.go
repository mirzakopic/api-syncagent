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

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/kcp-dev/logicalcluster/v3"
	"go.uber.org/zap"

	"k8c.io/servlet/internal/discovery"
	"k8c.io/servlet/internal/mutation"
	"k8c.io/servlet/internal/projection"
	"k8c.io/servlet/internal/sync"
	kdpservicesv1alpha1 "k8c.io/servlet/sdk/apis/services/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/kontext"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "servlet-sync"
)

type Reconciler struct {
	localClient ctrlruntimeclient.Client
	vwClient    ctrlruntimeclient.Client
	log         *zap.SugaredLogger
	syncer      *sync.ResourceSyncer
	remoteDummy *unstructured.Unstructured
}

// Create creates a new controller and importantly does *not* add it to the manager,
// as this controller is started/stopped by the syncmanager controller instead.
func Create(
	ctx context.Context,
	localManager manager.Manager,
	virtualWorkspaceCluster cluster.Cluster,
	pubRes *kdpservicesv1alpha1.PublishedResource,
	discoveryClient *discovery.Client,
	apiExportName string,
	log *zap.SugaredLogger,
	numWorkers int,
) (controller.Controller, error) {
	log = log.Named(ControllerName)

	// create a dummy that represents the type used on the local service cluster
	localGVK := projection.PublishedResourceSourceGVK(pubRes)
	localDummy := &unstructured.Unstructured{}
	localDummy.SetGroupVersionKind(localGVK)

	// create a dummy unstructured object with the projected GVK inside the workspace
	remoteGVK := projection.PublishedResourceProjectedGVK(pubRes, apiExportName)
	remoteDummy := &unstructured.Unstructured{}
	remoteDummy.SetGroupVersionKind(remoteGVK)

	// find the local CRD so we know the actual local object scope
	localCRD, err := discoveryClient.DiscoverResourceType(ctx, localGVK.GroupKind())
	if err != nil {
		return nil, fmt.Errorf("failed to find local CRD: %w", err)
	}

	// create the syncer that holds the meat&potatoes of the synchronization logic
	mutator := mutation.NewMutator(nil) // pubRes.Spec.Mutation
	syncer, err := sync.NewResourceSyncer(log, localManager.GetClient(), virtualWorkspaceCluster.GetClient(), pubRes, localCRD, apiExportName, mutator)
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer: %w", err)
	}

	// setup the reconciler
	reconciler := &Reconciler{
		localClient: localManager.GetClient(),
		vwClient:    virtualWorkspaceCluster.GetClient(),
		log:         log,
		remoteDummy: remoteDummy,
		syncer:      syncer,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}

	// It doesn't really matter what manager is used here, as starting/stopping happens
	// outside of the manager's control anyway.
	c, err := controller.NewUnmanaged(ControllerName, localManager, ctrlOptions)
	if err != nil {
		return nil, err
	}

	// watch the target resource in the virtual workspace
	if err := c.Watch(source.Kind(virtualWorkspaceCluster.GetCache(), remoteDummy, &handler.TypedEnqueueRequestForObject[*unstructured.Unstructured]{})); err != nil {
		return nil, err
	}

	// watch the source resource in the local cluster, but enqueue the origin remote object
	enqueueRemoteObjForLocalObj := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, o *unstructured.Unstructured) []reconcile.Request {
		req := sync.RemoteNameForLocalObject(o)
		if req == nil {
			return nil
		}

		return []reconcile.Request{*req}
	})

	if err := c.Watch(source.Kind(localManager.GetCache(), localDummy, enqueueRemoteObjForLocalObj)); err != nil {
		return nil, err
	}

	return c, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request, "cluster", request.ClusterName)
	log.Debug("Processing")

	wsCtx := kontext.WithCluster(ctx, logicalcluster.Name(request.ClusterName))

	remoteObj := r.remoteDummy.DeepCopy()
	if err := r.vwClient.Get(wsCtx, request.NamespacedName, remoteObj); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, fmt.Errorf("failed to retrieve remote object: %w", err)
	}

	// object was not found anymore
	if remoteObj.GetName() == "" {
		return reconcile.Result{}, nil
	}

	// sync main object
	requeue, err := r.syncer.Process(sync.NewContext(ctx, wsCtx), remoteObj)
	if err != nil {
		return reconcile.Result{}, err
	}

	result := reconcile.Result{}
	if requeue {
		// 5s was chosen at random, winning narrowly against 6s and 4.7s
		result.RequeueAfter = 5 * time.Second
	}

	return result, nil
}
