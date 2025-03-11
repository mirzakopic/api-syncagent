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

package discovery

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kcp-dev/kcp/pkg/crdpuller"

	"k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/utils/ptr"
)

type Client struct {
	discoveryClient discovery.DiscoveryInterface
	crdClient       apiextensionsv1client.ApiextensionsV1Interface
}

func NewClient(config *rest.Config) (*Client, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	crdClient, err := apiextensionsv1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		discoveryClient: discoveryClient,
		crdClient:       crdClient,
	}, nil
}

func (c *Client) RetrieveCRD(ctx context.Context, gvk schema.GroupVersionKind) (*apiextensionsv1.CustomResourceDefinition, error) {
	// Most of this code follows the logic in kcp's crd-puller, but is slimmed down
	// to extract a specific version, not necessarily the preferred version.

	////////////////////////////////////
	// Resolve GVK into GVR, because we need the resource name to construct
	// the full CRD name.

	_, resourceLists, err := c.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	var resource *metav1.APIResource
	allResourceNames := sets.New[string]()
	for _, resList := range resourceLists {
		for _, res := range resList.APIResources {
			allResourceNames.Insert(res.Name)

			// find the requested resource based on the Kind, but ensure that subresources
			// are not misinterpreted as the main resource by checking for "/"
			if resList.GroupVersion == gvk.GroupVersion().String() && res.Kind == gvk.Kind && !strings.Contains(res.Name, "/") {
				resource = &res
			}
		}
	}

	if resource == nil {
		return nil, fmt.Errorf("could not find %v in APIs", gvk)
	}

	////////////////////////////////////
	// If possible, retrieve the GVK as its original CRD, which is always preferred
	// because it's much more precise than what we can retrieve from the OpenAPI.
	// If no CRD can be found, fallback to the OpenAPI schema.

	crdName := resource.Name
	if gvk.Group == "" {
		crdName += ".core"
	} else {
		crdName += "." + gvk.Group
	}

	crd, err := c.crdClient.CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})

	// Hooray, we found a CRD! There is so much goodness on a real CRD that instead
	// of re-creating it later on based on the openapi schema, we take the original
	// CRD and just strip it down to what we need.
	if err == nil {
		// remove all but the requested version
		crd.Spec.Versions = slices.DeleteFunc(crd.Spec.Versions, func(ver apiextensionsv1.CustomResourceDefinitionVersion) bool {
			return ver.Name != gvk.Version
		})

		if len(crd.Spec.Versions) == 0 {
			return nil, fmt.Errorf("CRD %s does not contain version %s", crdName, gvk.Version)
		}

		crd.Spec.Versions[0].Served = true
		crd.Spec.Versions[0].Storage = true

		if apihelpers.IsCRDConditionTrue(crd, apiextensionsv1.NonStructuralSchema) {
			crd.Spec.Versions[0].Schema = &apiextensionsv1.CustomResourceValidation{
				OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
					Type:                   "object",
					XPreserveUnknownFields: ptr.To(true),
				},
			}
		}

		crd.APIVersion = apiextensionsv1.SchemeGroupVersion.Identifier()
		crd.Kind = "CustomResourceDefinition"

		// cleanup object meta
		oldMeta := crd.ObjectMeta
		crd.ObjectMeta = metav1.ObjectMeta{
			Name:        oldMeta.Name,
			Annotations: filterAnnotations(oldMeta.Annotations),
		}

		// There is only ever one version, so conversion rules do not make sense
		// (and even if they did, the conversion webhook from the service cluster
		// would not be available in kcp anyway).
		crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{
			Strategy: apiextensionsv1.NoneConverter,
		}

		return crd, nil
	}

	// any non-404 error is permanent
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// CRD not found, so fall back to using the OpenAPI schema
	openapiSchema, err := c.discoveryClient.OpenAPISchema()
	if err != nil {
		return nil, err
	}

	models, err := proto.NewOpenAPIData(openapiSchema)
	if err != nil {
		return nil, err
	}
	modelsByGKV, err := openapi.GetModelsByGKV(models)
	if err != nil {
		return nil, err
	}

	protoSchema := modelsByGKV[gvk]
	if protoSchema == nil {
		return nil, fmt.Errorf("no models for %v", gvk)
	}

	var schemaProps apiextensionsv1.JSONSchemaProps
	errs := crdpuller.Convert(protoSchema, &schemaProps)
	if len(errs) > 0 {
		return nil, utilerrors.NewAggregate(errs)
	}

	hasSubResource := func(subResource string) bool {
		return allResourceNames.Has(resource.Name + "/" + subResource)
	}

	var statusSubResource *apiextensionsv1.CustomResourceSubresourceStatus
	if hasSubResource("status") {
		statusSubResource = &apiextensionsv1.CustomResourceSubresourceStatus{}
	}

	var scaleSubResource *apiextensionsv1.CustomResourceSubresourceScale
	if hasSubResource("scale") {
		scaleSubResource = &apiextensionsv1.CustomResourceSubresourceScale{
			SpecReplicasPath:   ".spec.replicas",
			StatusReplicasPath: ".status.replicas",
		}
	}

	scope := apiextensionsv1.ClusterScoped
	if resource.Namespaced {
		scope = apiextensionsv1.NamespaceScoped
	}

	out := &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: apiextensionsv1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: gvk.Group,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name: gvk.Version,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &schemaProps,
					},
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: statusSubResource,
						Scale:  scaleSubResource,
					},
					Served:  true,
					Storage: true,
				},
			},
			Scope: scope,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:     resource.Name,
				Kind:       resource.Kind,
				Categories: resource.Categories,
				ShortNames: resource.ShortNames,
				Singular:   resource.SingularName,
			},
		},
	}

	apiextensionsv1.SetDefaults_CustomResourceDefinition(out)

	if apihelpers.IsProtectedCommunityGroup(gvk.Group) {
		out.Annotations = map[string]string{
			apiextensionsv1.KubeAPIApprovedAnnotation: "https://github.com/kcp-dev/kubernetes/pull/4",
		}
	}

	return out, nil
}

func filterAnnotations(ann map[string]string) map[string]string {
	allowlist := []string{
		apiextensionsv1.KubeAPIApprovedAnnotation,
	}

	out := map[string]string{}
	for k, v := range ann {
		if slices.Contains(allowlist, k) {
			out[k] = v
		}
	}

	return out
}
