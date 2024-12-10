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

package projection

import (
	"testing"

	"github.com/kcp-dev/logicalcluster/v3"

	kdpservicesv1alpha1 "k8c.io/servlet/sdk/apis/services/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func createNewObject(name, namespace string) metav1.Object {
	obj := &unstructured.Unstructured{}
	obj.SetName(name)
	obj.SetNamespace(namespace)

	return obj
}

func TestGenerateLocalObjectName(t *testing.T) {
	testcases := []struct {
		name         string
		clusterName  string
		remoteObject metav1.Object
		namingConfig *kdpservicesv1alpha1.ResourceNaming
		expected     types.NamespacedName
	}{
		{
			name:         "follow default naming rules",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: nil,
			expected:     types.NamespacedName{Namespace: "testcluster", Name: "e75ee3d444e238331f6a-8b09d63c82efb771a2c5"},
		},
		{
			name:         "custom static namespace pattern",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: &kdpservicesv1alpha1.ResourceNaming{Namespace: "foobar"},
			expected:     types.NamespacedName{Namespace: "foobar", Name: "e75ee3d444e238331f6a-8b09d63c82efb771a2c5"},
		},
		{
			name:         "custom dynamic namespace pattern",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: &kdpservicesv1alpha1.ResourceNaming{Namespace: "foobar-$remoteClusterName"},
			expected:     types.NamespacedName{Namespace: "foobar-testcluster", Name: "e75ee3d444e238331f6a-8b09d63c82efb771a2c5"},
		},
		{
			name:         "plain, unhashed values should be available in patterns",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: &kdpservicesv1alpha1.ResourceNaming{Namespace: "$remoteNamespace"},
			expected:     types.NamespacedName{Namespace: "objnamespace", Name: "e75ee3d444e238331f6a-8b09d63c82efb771a2c5"},
		},
		{
			name:         "configured but empty patterns",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: &kdpservicesv1alpha1.ResourceNaming{Namespace: "", Name: ""},
			expected:     types.NamespacedName{Namespace: "testcluster", Name: "e75ee3d444e238331f6a-8b09d63c82efb771a2c5"},
		},
		{
			name:         "custom dynamic name pattern",
			clusterName:  "testcluster",
			remoteObject: createNewObject("objname", "objnamespace"),
			namingConfig: &kdpservicesv1alpha1.ResourceNaming{Name: "foobar-$remoteName"},
			expected:     types.NamespacedName{Namespace: "testcluster", Name: "foobar-objname"},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			pubRes := &kdpservicesv1alpha1.PublishedResource{
				Spec: kdpservicesv1alpha1.PublishedResourceSpec{
					Naming: testcase.namingConfig,
				},
			}

			generatedName := GenerateLocalObjectName(pubRes, testcase.remoteObject, logicalcluster.Name(testcase.clusterName))

			if generatedName.String() != testcase.expected.String() {
				t.Errorf("Expected %q, but got %q.", testcase.expected, generatedName)
			}
		})
	}
}
