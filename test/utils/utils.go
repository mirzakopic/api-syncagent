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

package utils

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKcpClient(t *testing.T) ctrlruntimeclient.Client {
	t.Helper()

	sc := runtime.NewScheme()
	if err := scheme.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KCP_KUBECONFIG"))
	if err != nil {
		t.Fatalf("Failed to get kcp kubeconfig: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{
		Scheme: sc,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}
