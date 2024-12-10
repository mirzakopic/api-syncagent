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

package rbac

import (
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetClusterRoleReconciler(source *rbacv1.ClusterRole) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return source.Name, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			setMetadata(source, cr)

			cr.AggregationRule = source.AggregationRule
			// If rules are aggregated, this field will be managed by Kubernetes.
			// we must not override it because the source rules are not necessarily
			// the same rules aggregated in the target workspace.
			if cr.AggregationRule == nil {
				cr.Rules = source.Rules
			}

			return cr, nil
		}
	}
}

func GetClusterRoleBindingReconciler(source *rbacv1.ClusterRoleBinding) reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return source.Name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			setMetadata(source, crb)

			crb.RoleRef = source.RoleRef
			crb.Subjects = source.Subjects

			return crb, nil
		}
	}
}

func setMetadata(src metav1.Object, dest metav1.Object) {
	annotations := dest.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	for key, value := range src.GetAnnotations() {
		// Don't sync annotations created by kcp.
		if !strings.HasPrefix(key, "kcp.io") && !strings.HasPrefix(key, "internal.kcp.io") {
			annotations[key] = value
		}
	}

	dest.SetAnnotations(annotations)

	labels := dest.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for key, value := range src.GetLabels() {
		labels[key] = value
	}

	dest.SetLabels(labels)
}
