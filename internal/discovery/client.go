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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	kubeClient ctrlruntimeclient.Reader
}

func NewClient(kubeClient ctrlruntimeclient.Client) *Client {
	return &Client{
		kubeClient: kubeClient,
	}
}

func (c *Client) DiscoverResourceType(ctx context.Context, gk schema.GroupKind) (*apiextensionsv1.CustomResourceDefinition, error) {
	crds := &apiextensionsv1.CustomResourceDefinitionList{}
	if err := c.kubeClient.List(ctx, crds); err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	for _, crd := range crds.Items {
		if crd.Spec.Group != gk.Group {
			continue
		}

		if crd.Spec.Names.Kind != gk.Kind {
			continue
		}

		return &crd, nil
	}

	return nil, fmt.Errorf("CustomResourceDefinition for %s/%s does not exist", gk.Group, gk.Kind)
}
