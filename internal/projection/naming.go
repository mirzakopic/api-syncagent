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

package projection

import (
	"fmt"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"

	"github.com/kcp-dev/api-syncagent/internal/crypto"
	syncagentv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/syncagent/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var DefaultNamingScheme = syncagentv1alpha1.ResourceNaming{
	Namespace: syncagentv1alpha1.PlaceholderRemoteClusterName,
	Name:      fmt.Sprintf("%s-%s", syncagentv1alpha1.PlaceholderRemoteNamespaceHash, syncagentv1alpha1.PlaceholderRemoteNameHash),
}

func GenerateLocalObjectName(pr *syncagentv1alpha1.PublishedResource, object metav1.Object, clusterName logicalcluster.Name) types.NamespacedName {
	naming := pr.Spec.Naming
	if naming == nil {
		naming = &syncagentv1alpha1.ResourceNaming{}
	}

	replacer := strings.NewReplacer(
		// order of elements is important here, "$fooHash" needs to be defined before "$foo"
		syncagentv1alpha1.PlaceholderRemoteClusterName, clusterName.String(),
		syncagentv1alpha1.PlaceholderRemoteNamespaceHash, crypto.ShortHash(object.GetNamespace()),
		syncagentv1alpha1.PlaceholderRemoteNamespace, object.GetNamespace(),
		syncagentv1alpha1.PlaceholderRemoteNameHash, crypto.ShortHash(object.GetName()),
		syncagentv1alpha1.PlaceholderRemoteName, object.GetName(),
	)

	result := types.NamespacedName{}

	pattern := naming.Namespace
	if pattern == "" {
		pattern = DefaultNamingScheme.Namespace
	}

	result.Namespace = replacer.Replace(pattern)

	pattern = naming.Name
	if pattern == "" {
		pattern = DefaultNamingScheme.Name
	}

	result.Name = replacer.Replace(pattern)

	return result
}
