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

const (
	// CreateDefaultClusterRolesLabel labels APIBindings for which a set of default ClusterRoles
	// (developer and viewer) should be generated and aggregated into the general KDP roles.
	CreateDefaultClusterRolesLabel = "rbac.kdp.k8c.io/create-default-clusterroles"

	// AggregateToDeveloperLabel labels ClusterRoles that should be aggregated into the
	// general KDP role "kdp:developer" (if present through RBAC sync).
	AggregateToDeveloperLabel = "rbac.kdp.k8c.io/aggregate-to-developer"
	// AggregateToMemberLabel labels ClusterRoles that should be aggregated into the
	// general KDP role "kdp:member" (if present through RBAC sync).
	AggregateToMemberLabel = "rbac.kdp.k8c.io/aggregate-to-member"

	// DisplayLabel labels ClusterRoles that should be visible from the KDP Dashboard for easy assignment.
	DisplayLabel = "rbac.kdp.k8c.io/display"
	// DisplayLabel annotates ClusterRoles with a human-readable name that will be shown in the KDP Dashboard.
	// If this is not set, the fallback is the object name.
	DisplayNameAnnotation = "rbac.kdp.k8c.io/display-name"
	// DescriptionAnnotation annotates ClusterRoles with a "help" text describing the permissions granted by
	// the ClusterRole.
	DescriptionAnnotation = "rbac.kdp.k8c.io/description"

	// SyncToWorkspacesAnnotation annotates ClusterRoles and ClusterRoleBindings that
	// should be synced to children workspaces.
	SyncToWorkspacesAnnotation = "kdp.k8c.io/sync-to-workspaces"
	// WildcardTarget is the value for SyncToWorkspacesAnnotation that denotes a sync to all workspaces.
	WildcardTarget = "*"
)
