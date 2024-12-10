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

package crd

import (
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func APIResourceSchema(crd *apiextensionsv1.CustomResourceDefinition, prefix string) (*apisv1alpha1.APIResourceSchema, error) {
	return apisv1alpha1.CRDToAPIResourceSchema(crd, prefix)
}
