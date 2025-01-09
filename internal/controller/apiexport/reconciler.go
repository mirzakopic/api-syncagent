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

package apiexport

import (
	"cmp"
	"slices"

	"github.com/kcp-dev/api-syncagent/internal/resources/reconciling"
	"github.com/kcp-dev/api-syncagent/sdk/apis/services"

	kcpdevv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"

	"k8s.io/apimachinery/pkg/util/sets"
)

// createAPIExportReconciler creates the reconciler for the APIExport.
// WARNING: The APIExport in this is NOT created by the Sync Agent, it's created
// by a controller in kcp. Make sure you don't create a reconciling conflict!
func (r *Reconciler) createAPIExportReconciler(availableResourceSchemas sets.Set[string], claimedResourceKinds sets.Set[string], agentName string, apiExportName string) reconciling.NamedAPIExportReconcilerFactory {
	return func() (string, reconciling.APIExportReconciler) {
		return apiExportName, func(existing *kcpdevv1alpha1.APIExport) (*kcpdevv1alpha1.APIExport, error) {
			known := sets.New[string](existing.Spec.LatestResourceSchemas...)

			if existing.Annotations == nil {
				existing.Annotations = map[string]string{}
			}
			existing.Annotations[services.AgentNameAnnotation] = agentName

			// we only ever add new schemas
			result := known.Union(availableResourceSchemas)
			existing.Spec.LatestResourceSchemas = sets.List(result)

			// To allow admins to configure additional permission claims, sometimes
			// useful for debugging, we do not override the permission claims, but
			// only ensure the ones originating from the published resources;
			// step 1 is to collect all existing claims with the same properties
			// as ours.
			existingClaims := sets.New[string]()
			for _, claim := range existing.Spec.PermissionClaims {
				if claim.All && claim.Group == "" && len(claim.ResourceSelector) == 0 {
					existingClaims.Insert(claim.Resource)
				}
			}

			missingClaims := claimedResourceKinds.Difference(existingClaims)

			// add our missing claims
			for _, claimed := range sets.List(missingClaims) {
				existing.Spec.PermissionClaims = append(existing.Spec.PermissionClaims, kcpdevv1alpha1.PermissionClaim{
					GroupResource: kcpdevv1alpha1.GroupResource{
						Group:    "",
						Resource: claimed,
					},
					All: true,
				})
			}

			// prevent reconcile loops by ensuring a stable order
			slices.SortFunc(existing.Spec.PermissionClaims, func(a, b kcpdevv1alpha1.PermissionClaim) int {
				if a.Group != b.Group {
					return cmp.Compare(a.Group, b.Group)
				}

				if a.Resource != b.Resource {
					return cmp.Compare(a.Resource, b.Resource)
				}

				return 0
			})

			return existing, nil
		}
	}
}
