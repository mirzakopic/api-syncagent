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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

type delegatedTransport struct {
	delegate *genericapiserver.APIServerHandler
}

func (t *delegatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	writer := httptest.NewRecorder()
	t.delegate.ServeHTTP(writer, req)

	return writer.Result(), nil
}

// NewDelegatedHTTPClient returns a new HTTP client that forwards all requests
// to the given server's Handler by calling it's ServeHTTP() instead of doing an
// actual network request.
func NewDelegatedHTTPClient(t *testing.T, restConfig *rest.Config, server *genericapiserver.GenericAPIServer) *http.Client {
	t.Helper()

	httpClient, err := rest.HTTPClientFor(restConfig)
	require.NoError(t, err, "failed to construct custom HTTP client")

	transportCfg, err := restConfig.TransportConfig()
	require.NoError(t, err, "failed to get transport config")

	transportCfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &delegatedTransport{
			delegate: server.Handler,
		}
	})

	rt, err := transport.New(transportCfg)
	require.NoError(t, err, "failed to create transport")

	httpClient.Transport = rt

	return httpClient
}
