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

package main

import (
	"context"
	"flag"
	"fmt"
	golog "log"
	"strings"

	"github.com/go-logr/zapr"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/kcp-dev/api-syncagent/internal/controller/apiexport"
	"github.com/kcp-dev/api-syncagent/internal/controller/apiresourceschema"
	"github.com/kcp-dev/api-syncagent/internal/controller/syncmanager"
	"github.com/kcp-dev/api-syncagent/internal/kcp"
	syncagentlog "github.com/kcp-dev/api-syncagent/internal/log"
	"github.com/kcp-dev/api-syncagent/internal/version"

	kcpdevv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpdevcore "github.com/kcp-dev/kcp/sdk/apis/core"
	kcpdevcorev1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	ctx := context.Background()

	opts := NewOptions()
	opts.AddFlags(pflag.CommandLine)

	// ctrl-runtime will have added its --kubeconfig to Go's flag set
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if err := opts.Validate(); err != nil {
		golog.Fatalf("Invalid command line: %v", err)
	}

	log := syncagentlog.NewFromOptions(opts.LogOptions)

	if err := opts.Complete(); err != nil {
		log.With(zap.Error(err)).Fatal("Invalid command line")
	}

	sugar := log.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(log.WithOptions(zap.AddCallerSkip(1))))

	if err := run(ctx, sugar, opts); err != nil {
		sugar.Fatalw("Sync Agent has encountered an error", zap.Error(err))
	}
}

func run(ctx context.Context, log *zap.SugaredLogger, opts *Options) error {
	v := version.NewAppVersion()
	log.With(
		"version", v.GitVersion,
		"name", opts.AgentName,
		"apiexport", opts.APIExportRef,
	).Info("Moin, I'm the kcp Sync Agent")

	// create the ctrl-runtime manager
	mgr, err := setupLocalManager(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to setup local manager: %w", err)
	}

	// load the kcp kubeconfig
	platformRestConfig, err := loadKubeconfig(opts.PlatformKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load platform kubeconfig: %w", err)
	}

	// sanity check
	if !strings.Contains(platformRestConfig.Host, "/clusters/") {
		return fmt.Errorf("platform kubeconfig does not point to a specific workspace")
	}

	// We check if the APIExport exists and extract information we need to set up our platformCluster.
	apiExport, lcPath, lcName, err := resolveAPIExport(ctx, platformRestConfig, opts.APIExportRef)
	if err != nil {
		return fmt.Errorf("failed to resolve APIExport: %w", err)
	}

	log.Infow("Resolved APIExport", "apigroup", opts.APIExportRef, "workspace", lcPath, "logicalcluster", lcName)

	// init the "permanent" platform cluster connection
	platformCluster, err := setupPlatformCluster(platformRestConfig, opts.APIExportRef)
	if err != nil {
		return fmt.Errorf("failed to initialize platform cluster: %w", err)
	}

	// start the platform cluster caches when the manager boots up
	// (happens regardless of leader election status)
	if err := mgr.Add(platformCluster); err != nil {
		return fmt.Errorf("failed to add platform cluster runnable: %w", err)
	}

	if err := apiresourceschema.Add(mgr, platformCluster, lcName, log, 4, opts.AgentName, opts.APIExportRef, opts.PublishedResourceSelector); err != nil {
		return fmt.Errorf("failed to add apiresourceschema controller: %w", err)
	}

	if err := apiexport.Add(mgr, platformCluster, lcName, log, opts.APIExportRef, opts.AgentName, opts.PublishedResourceSelector); err != nil {
		return fmt.Errorf("failed to add apiexport controller: %w", err)
	}

	if err := syncmanager.Add(ctx, mgr, platformCluster, platformRestConfig, log, apiExport, opts.PublishedResourceSelector); err != nil {
		return fmt.Errorf("failed to add syncmanager controller: %w", err)
	}

	log.Info("Starting kcp Sync Agent…")

	return mgr.Start(ctx)
}

func setupLocalManager(ctx context.Context, opts *Options) (manager.Manager, error) {
	scheme := runtime.NewScheme()

	mgr, err := manager.New(ctrlruntime.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		BaseContext: func() context.Context {
			return ctx
		},
		LeaderElection:          opts.EnableLeaderElection,
		LeaderElectionID:        "syncagent." + opts.AgentName,
		LeaderElectionNamespace: opts.Namespace,
	})
	if err != nil {
		return nil, err
	}

	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register local scheme %s: %w", corev1.SchemeGroupVersion, err)
	}

	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register local scheme %s: %w", apiextensionsv1.SchemeGroupVersion, err)
	}

	return mgr, nil
}

func resolveAPIExport(ctx context.Context, restConfig *rest.Config, apiExportRef string) (*kcpdevv1alpha1.APIExport, logicalcluster.Path, logicalcluster.Name, error) {
	// construct temporary, uncached client
	scheme := runtime.NewScheme()
	if err := kcpdevcorev1alpha1.AddToScheme(scheme); err != nil {
		return nil, logicalcluster.None, "", fmt.Errorf("failed to register scheme %s: %w", kcpdevcorev1alpha1.SchemeGroupVersion, err)
	}
	if err := kcpdevv1alpha1.AddToScheme(scheme); err != nil {
		return nil, logicalcluster.None, "", fmt.Errorf("failed to register scheme %s: %w", kcpdevv1alpha1.SchemeGroupVersion, err)
	}

	client, err := ctrlruntimeclient.New(restConfig, ctrlruntimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, logicalcluster.None, "", fmt.Errorf("failed to create service reader: %w", err)
	}

	apiExport := &kcpdevv1alpha1.APIExport{}
	key := types.NamespacedName{Name: apiExportRef}
	if err := client.Get(ctx, key, apiExport); err != nil {
		return nil, logicalcluster.None, "", fmt.Errorf("failed to get APIExport %q: %w", apiExportRef, err)
	}

	// kcp's controller-runtime fork always caches objects including their logicalcluster names.
	// Our app technically doesn't care about workspaces / logical clusters, but we still need to
	// supply the correct logicalcluster when querying for objects.
	// We could take the cluster name from the Service itself, but since we want to log the nicer
	// looking cluster _path_, we fetch the workspace's own logicalcluster object.
	lc := &kcpdevcorev1alpha1.LogicalCluster{}
	key = types.NamespacedName{Name: kcp.IdentityClusterName}
	if err := client.Get(ctx, key, lc); err != nil {
		return nil, logicalcluster.None, "", fmt.Errorf("failed to resolve current workspace: %w", err)
	}

	lcName := logicalcluster.From(lc)
	lcPath := logicalcluster.NewPath(lc.Annotations[kcpdevcore.LogicalClusterPathAnnotationKey])

	return apiExport, lcPath, lcName, nil
}

// setupPlatformCluster sets up a plain, non-kcp-aware ctrl-runtime Cluster object
// that is solvely used to interact with the APIExport and APIResourceSchemas.
func setupPlatformCluster(restConfig *rest.Config, apiexportRef string) (cluster.Cluster, error) {
	scheme := runtime.NewScheme()

	if err := kcpdevv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register scheme %s: %w", kcpdevv1alpha1.SchemeGroupVersion, err)
	}

	if err := kcpdevcorev1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register scheme %s: %w", kcpdevcorev1alpha1.SchemeGroupVersion, err)
	}

	return cluster.New(restConfig, func(o *cluster.Options) {
		o.Scheme = scheme
		// RBAC on the platform cluster is very tight and does not allow to list/watch all objects;
		// restrict the cache's selectors accordingly so we can still make use of caching.
		o.Cache = cache.Options{
			Scheme: scheme,
			ByObject: map[ctrlruntimeclient.Object]cache.ByObject{
				&kcpdevv1alpha1.APIExport{}: {
					Field: fields.SelectorFromSet(fields.Set{"metadata.name": apiexportRef}),
				},
			},
		}
	})
}

func loadKubeconfig(filename string) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = filename

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil).ClientConfig()
}
