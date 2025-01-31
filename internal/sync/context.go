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

package sync

import (
	"context"

	"github.com/kcp-dev/logicalcluster/v3"
	"sigs.k8s.io/controller-runtime/pkg/kontext"
)

type Context struct {
	clusterName logicalcluster.Name
	clusterPath logicalcluster.Path
	local       context.Context
	remote      context.Context
}

func NewContext(local, remote context.Context) Context {
	clusterName, ok := kontext.ClusterFrom(remote)
	if !ok {
		panic("Provided remote context does not contain cluster name.")
	}

	return Context{
		clusterName: clusterName,
		local:       local,
		remote:      remote,
	}
}

func (c *Context) WithClusterPath(path logicalcluster.Path) Context {
	return Context{
		clusterName: c.clusterName,
		clusterPath: path,
		local:       c.local,
		remote:      c.remote,
	}
}
