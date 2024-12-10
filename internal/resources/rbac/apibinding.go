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
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

type roleType int

const (
	developer roleType = iota
	viewer
)

func GetClusterRoleServiceDeveloperReconciler(name string, apis map[string][]string) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return fmt.Sprintf("services:%s:developer", name), func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			return serviceClusterRole(cr, apis, developer, name)
		}
	}
}

func GetClusterRoleServiceViewerReconciler(name string, apis map[string][]string) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return fmt.Sprintf("services:%s:viewer", name), func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			return serviceClusterRole(cr, apis, viewer, name)
		}
	}
}

func serviceClusterRole(cr *rbacv1.ClusterRole, apis map[string][]string, t roleType, name string) (*rbacv1.ClusterRole, error) {
	var (
		aggregateLabel, displayName string
		verbs                       []string
	)

	switch t {
	case developer:
		aggregateLabel = AggregateToDeveloperLabel
		displayName = "Developer"
		verbs = []string{"get", "list", "watch", "create", "update", "patch", "delete"}
	case viewer:
		aggregateLabel = AggregateToMemberLabel
		displayName = "Viewer"
		verbs = []string{"get", "list", "watch"}
	default:
		return nil, fmt.Errorf("unknown role type ID passed: %d", t)
	}

	if cr.Labels == nil {
		cr.Labels = make(map[string]string)
	}

	cr.Labels[aggregateLabel] = "true"
	cr.Labels[DisplayLabel] = "true"

	if cr.Annotations == nil {
		cr.Annotations = make(map[string]string)
	}

	cr.Annotations[DisplayNameAnnotation] = fmt.Sprintf("%s %s", name, displayName)
	cr.Annotations[DescriptionAnnotation] = fmt.Sprintf("Allows %s access to resources managed by %s", strings.ToLower(displayName), name)

	cr.Rules = []rbacv1.PolicyRule{}

	for apiName, apiResources := range apis {
		cr.Rules = append(cr.Rules, rbacv1.PolicyRule{
			APIGroups: []string{apiName},
			Resources: apiResources,
			Verbs:     verbs,
		})
	}

	return cr, nil
}
