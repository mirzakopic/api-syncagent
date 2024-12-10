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
	"testing"

	"github.com/kcp-dev/logicalcluster/v3"

	"github.com/kcp-dev/kcp/sdk/apis/core"
	tenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/kcp/test/e2e/framework"
)

func NewOrganizationFixture(t *testing.T, server framework.RunningServer, options ...framework.UnprivilegedWorkspaceOption) (logicalcluster.Path, *tenancyv1alpha1.Workspace) {
	t.Helper()
	return framework.NewWorkspaceFixture(t, server, core.RootCluster.Path(), append(options, framework.WithType(core.RootCluster.Path(), "kdp-organization"))...)
}
