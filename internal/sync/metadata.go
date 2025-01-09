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
	"maps"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func stripMetadata(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetFinalizers(nil)
	obj.SetGeneration(0)
	obj.SetOwnerReferences(nil)
	obj.SetResourceVersion("")
	obj.SetManagedFields(nil)
	obj.SetUID("")
	obj.SetSelfLink("")

	stripAnnotations(obj)
	stripLabels(obj)

	return obj
}

func stripAnnotations(obj *unstructured.Unstructured) *unstructured.Unstructured {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return obj
	}

	delete(annotations, "kcp.io/cluster")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	maps.DeleteFunc(annotations, func(annotation string, _ string) bool {
		return strings.HasPrefix(annotation, relatedObjectAnnotationPrefix)
	})

	obj.SetAnnotations(annotations)

	return obj
}

func stripLabels(obj *unstructured.Unstructured) *unstructured.Unstructured {
	labels := obj.GetLabels()
	if labels == nil {
		return obj
	}

	for _, label := range ignoredRemoteLabels.UnsortedList() {
		delete(labels, label)
	}

	obj.SetLabels(labels)

	return obj
}

// ignoredRemoteLabels are labels we never want to copy from the remote to local objects.
var ignoredRemoteLabels = sets.New[string](
	remoteObjectClusterLabel,
	remoteObjectNamespaceLabel,
	remoteObjectNameLabel,
)

// filterRemoteLabels removes all unwanted remote labels and returns a new label set.
func filterRemoteLabels(remoteLabels labels.Set) labels.Set {
	filteredLabels := labels.Set{}

	for k, v := range remoteLabels {
		if !ignoredRemoteLabels.Has(k) {
			filteredLabels[k] = v
		}
	}

	return filteredLabels
}

func RemoteNameForLocalObject(localObj ctrlruntimeclient.Object) *reconcile.Request {
	labels := localObj.GetLabels()
	clusterName := labels[remoteObjectClusterLabel]
	namespace := labels[remoteObjectNamespaceLabel]
	name := labels[remoteObjectNameLabel]

	// reject/ignore invalid/badly labelled object
	if clusterName == "" || name == "" {
		return nil
	}

	return &reconcile.Request{
		ClusterName: clusterName,
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}
