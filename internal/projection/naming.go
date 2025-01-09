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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"

	servicesv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/services/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var DefaultNamingScheme = servicesv1alpha1.ResourceNaming{
	Namespace: servicesv1alpha1.PlaceholderRemoteClusterName,
	Name:      fmt.Sprintf("%s-%s", servicesv1alpha1.PlaceholderRemoteNamespaceHash, servicesv1alpha1.PlaceholderRemoteNameHash),
}

func GenerateLocalObjectName(pr *servicesv1alpha1.PublishedResource, object metav1.Object, clusterName logicalcluster.Name) types.NamespacedName {
	naming := pr.Spec.Naming
	if naming == nil {
		naming = &servicesv1alpha1.ResourceNaming{}
	}

	replacer := strings.NewReplacer(
		// order of elements is important here, "$fooHash" needs to be defined before "$foo"
		servicesv1alpha1.PlaceholderRemoteClusterName, clusterName.String(),
		servicesv1alpha1.PlaceholderRemoteNamespaceHash, shortSha1Hash(object.GetNamespace()),
		servicesv1alpha1.PlaceholderRemoteNamespace, object.GetNamespace(),
		servicesv1alpha1.PlaceholderRemoteNameHash, shortSha1Hash(object.GetName()),
		servicesv1alpha1.PlaceholderRemoteName, object.GetName(),
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

func shortSha1Hash(value string) string {
	hash := sha1.New()
	if _, err := hash.Write([]byte(value)); err != nil {
		// This is not something that should ever happen at runtime and is also not
		// something we can really gracefully handle, so crashing and restarting might
		// be a good way to signal the service owner that something is up.
		panic(fmt.Sprintf("Failed to hash string: %v", err))
	}

	encoded := hex.EncodeToString(hash.Sum(nil))

	return encoded[:20]
}
