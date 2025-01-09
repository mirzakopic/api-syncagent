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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/kontext"
)

// NewClusterAwareAPIReader returns a client.Reader that provides read-only access to the API server,
// and is configured to use the context to scope requests to the proper cluster. To scope requests,
// pass the request context with the cluster set.
// Example:
//
//	import (
//		"context"
//		kcpclient "github.com/kcp-dev/apimachinery/v2/pkg/client"
//		ctrl "sigs.k8s.io/controller-runtime"
//	)
//	func (r *reconciler)  Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//		ctx = kcpclient.WithCluster(ctx, req.ObjectKey.Cluster)
//		// from here on pass this context to all client calls
//		...
//	}
func NewClusterAwareAPIReader(config *rest.Config, opts ctrlruntimeclient.Options) (ctrlruntimeclient.Reader, error) {
	httpClient, err := ClusterAwareHTTPClient(config)
	if err != nil {
		return nil, err
	}
	opts.HTTPClient = httpClient
	return ctrlruntimeclient.NewAPIReader(config, opts)
}

// ClusterAwareHTTPClient returns an http.Client with a cluster aware round tripper.
func ClusterAwareHTTPClient(config *rest.Config) (*http.Client, error) {
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}

	httpClient.Transport = newClusterRoundTripper(httpClient.Transport)
	return httpClient, nil
}

// clusterRoundTripper is a cluster aware wrapper around http.RoundTripper.
type clusterRoundTripper struct {
	delegate http.RoundTripper
}

// newClusterRoundTripper creates a new cluster aware round tripper.
func newClusterRoundTripper(delegate http.RoundTripper) *clusterRoundTripper {
	return &clusterRoundTripper{
		delegate: delegate,
	}
}

func (c *clusterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cluster, ok := kontext.ClusterFrom(req.Context())
	if ok {
		req = req.Clone(req.Context())
		req.URL.Path = generatePath(req.URL.Path, cluster.Path())
		req.URL.RawPath = generatePath(req.URL.RawPath, cluster.Path())
	}
	return c.delegate.RoundTrip(req)
}

// apiRegex matches any string that has /api/ or /apis/ in it.
var apiRegex = regexp.MustCompile(`(/api/|/apis/)`)

var clustersRegex = regexp.MustCompile(`^/clusters/[^/]+`)

// generatePath formats the request path to target the specified cluster.
func generatePath(originalPath string, clusterPath logicalcluster.Path) string {
	// HACK: strip any pre-existing /clusters/.... prefix
	originalPath = clustersRegex.ReplaceAllString(originalPath, "")

	// If the originalPath already has cluster.Path() then the path was already modified and no change needed
	if strings.Contains(originalPath, clusterPath.RequestPath()) {
		return originalPath
	}
	// If the originalPath has /api/ or /apis/ in it, it might be anywhere in the path, so we use a regex to find and
	// replaces /api/ or /apis/ with $cluster/api/ or $cluster/apis/
	if apiRegex.MatchString(originalPath) {
		return apiRegex.ReplaceAllString(originalPath, fmt.Sprintf("%s$1", clusterPath.RequestPath()))
	}
	// Otherwise, we're just prepending /clusters/$name
	path := clusterPath.RequestPath()
	// if the original path is relative, add a / separator
	if len(originalPath) > 0 && originalPath[0] != '/' {
		path += "/"
	}
	// finally append the original path
	path += originalPath
	return path
}
