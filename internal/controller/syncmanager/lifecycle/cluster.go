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

package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/kcp"
)

// Cluster is a controller-runtime cluster
// that can be stopped by cancelling its root context.
type Cluster struct {
	// a Cluster representing the virtual workspace for the APIExport
	obj cluster.Cluster

	// a signal that is closed when the vwCluster has stopped
	stopped chan struct{}

	// a function that is used to stop the vwCluster
	cancelFunc context.CancelCauseFunc
}

func NewCluster(address string, baseRestConfig *rest.Config) (*Cluster, error) {
	// note that this cluster and all its components are kcp-aware
	config := rest.CopyConfig(baseRestConfig)
	config.Host = address

	clusterObj, err := cluster.New(config, func(o *cluster.Options) {
		o.NewCache = kcp.NewClusterAwareCache
		o.NewAPIReader = kcp.NewClusterAwareAPIReader
		o.NewClient = kcp.NewClusterAwareClient
		// o.MapperProvider = kcp.NewClusterAwareMapperProvider
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cluster: %w", err)
	}

	return &Cluster{
		obj: clusterObj,
	}, nil
}

// Start starts a goroutine for the underlying cluster object; make sure to use
// a long-lived context here.
func (c *Cluster) Start(ctx context.Context, log *zap.SugaredLogger) error {
	if c.obj == nil {
		return errors.New("cannot restart a stopped cluster")
	}

	if c.stopped != nil {
		return errors.New("cluster is already running")
	}

	clusterCtx, cancel := context.WithCancelCause(ctx)

	c.cancelFunc = cancel
	c.stopped = make(chan struct{})

	// start the cluster in a new goroutine
	go func() {
		defer close(c.stopped)

		// this call blocks until clusterCtx is done; Start() never returns an error
		// in real-life scenarios, as the cluster just waits for the cache to
		// end and caches only end (cleanly) when the context is closed.
		// Since this "cannot fail" at runtime, we do not need to somehow trigger
		// a full reconciliation when this fails (like recreating a new cluster,
		// stopping and restarting all sync controllers, ...).
		if err := c.obj.Start(clusterCtx); err != nil {
			log.Errorw("Virtual workspace cluster has failed", zap.Error(err))
		}

		cancel(errors.New("closing to prevent leakage"))

		c.obj = nil
		c.cancelFunc = nil
	}()

	// wait for the cluster to be up (context can be anything here)
	if !c.obj.GetCache().WaitForCacheSync(ctx) {
		err := errors.New("failed to wait for caches to sync")

		// stop the cluster
		cancel(err)

		// wait for cleanup to be completed
		<-c.stopped

		return err
	}

	return nil
}

func (c *Cluster) Running() bool {
	if c.obj == nil {
		return false
	}

	if c.stopped == nil {
		return false
	}

	select {
	case <-c.stopped:
		return false

	default:
		return true
	}
}

func (c *Cluster) Stop(log *zap.SugaredLogger) error {
	if !c.Running() {
		return errors.New("cluster is not running")
	}

	c.cancelFunc(errors.New("virtual workspace URL has changed"))
	log.Info("Waiting for virtual workspace cluster to shut down…")
	<-c.stopped
	log.Info("Virtual workspace cluster has finished shutting down.")

	return nil
}

func (c *Cluster) GetCluster() cluster.Cluster {
	return c.obj
}

func (c *Cluster) GetClient() (ctrlruntimeclient.Client, error) {
	if !c.Running() {
		return nil, errors.New("cluster is not running")
	}

	return c.obj.GetClient(), nil
}

func (c *Cluster) GetCache() (cache.Cache, error) {
	if !c.Running() {
		return nil, errors.New("cluster is not running")
	}

	return c.obj.GetCache(), nil
}
