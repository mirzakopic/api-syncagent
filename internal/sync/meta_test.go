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

package sync

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func createNewObject(name, namespace string) metav1.Object {
	obj := &unstructured.Unstructured{}
	obj.SetName(name)
	obj.SetNamespace(namespace)

	return obj
}

func TestObjectKey(t *testing.T) {
	testcases := []struct {
		object      metav1.Object
		clusterName string
		expected    string
	}{
		{
			object:      createNewObject("test", ""),
			clusterName: "",
			expected:    "test",
		},
		{
			object:      createNewObject("test", "namespace"),
			clusterName: "",
			expected:    "namespace/test",
		},
		{
			object:      createNewObject("test", ""),
			clusterName: "abc123",
			expected:    "abc123|test",
		},
		{
			object:      createNewObject("test", "namespace"),
			clusterName: "abc123",
			expected:    "abc123|namespace/test",
		},
	}

	for _, testcase := range testcases {
		t.Run("", func(t *testing.T) {
			key := newObjectKey(testcase.object, testcase.clusterName)

			if stringified := key.String(); stringified != testcase.expected {
				t.Fatalf("Expected %q but got %q.", testcase.expected, stringified)
			}
		})
	}
}
