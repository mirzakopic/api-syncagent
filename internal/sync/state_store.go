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

package sync

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectStateStore interface {
	Get(source syncSide) (*unstructured.Unstructured, error)
	Put(obj *unstructured.Unstructured, clusterName string, subresources []string) error
}

// objectStateStore is capable of creating/updating a target Kubernetes object
// based on a source object. It keeps track of the source's state so that fields
// that are changed after/outside of the Sync Agent are not undone by accident.
// This is the same logic as kubectl has using its last-known annotation.
type objectStateStore struct {
	backend backend
}

func newObjectStateStore(primaryObject, stateCluster syncSide) ObjectStateStore {
	kubernetes := newKubernetesBackend(primaryObject, stateCluster)

	return &objectStateStore{
		backend: kubernetes,
	}
}

func (op *objectStateStore) Get(source syncSide) (*unstructured.Unstructured, error) {
	data, err := op.backend.Get(source.object, source.clusterName)
	if err != nil {
		return nil, err
	}

	lastKnown := &unstructured.Unstructured{}
	if err := lastKnown.UnmarshalJSON(data); err != nil {
		// if no last-known-state annotation exists or it's defective, the destination object is
		// technically broken and we have to fall back to a full update
		return nil, nil
	}

	return lastKnown, nil
}

func (op *objectStateStore) Put(obj *unstructured.Unstructured, clusterName string, subresources []string) error {
	encoded, err := op.snapshotObject(obj, subresources)
	if err != nil {
		return err
	}

	return op.backend.Put(obj, clusterName, []byte(encoded))
}

func (op *objectStateStore) snapshotObject(obj *unstructured.Unstructured, subresources []string) (string, error) {
	obj = obj.DeepCopy()
	obj = stripMetadata(obj)

	// besides metadata, we also do not care about the object's subresources
	data := obj.UnstructuredContent()
	for _, key := range subresources {
		delete(data, key)
	}

	marshalled, err := obj.MarshalJSON()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(marshalled)), nil
}

type backend interface {
	Get(obj *unstructured.Unstructured, clusterName string) ([]byte, error)
	Put(obj *unstructured.Unstructured, clusterName string, data []byte) error
}

type kubernetesBackend struct {
	secretName   types.NamespacedName
	labels       labels.Set
	stateCluster syncSide
}

func hashObject(obj *unstructured.Unstructured) string {
	data := map[string]any{
		"apiVersion": obj.GetAPIVersion(),
		"namespace":  obj.GetNamespace(),
		"name":       obj.GetName(),
	}

	hash := sha1.New()

	if err := json.NewEncoder(hash).Encode(data); err != nil {
		// This is not something that should ever happen at runtime and is also not
		// something we can really gracefully handle, so crashing and restarting might
		// be a good way to signal the service owner that something is up.
		panic(fmt.Sprintf("Failed to hash object key: %v", err))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func newKubernetesBackend(primaryObject, stateCluster syncSide) *kubernetesBackend {
	keyHash := hashObject(primaryObject.object)

	secretLabels := newObjectKey(primaryObject.object, primaryObject.clusterName).Labels()
	secretLabels[objectStateLabelName] = objectStateLabelValue

	return &kubernetesBackend{
		secretName: types.NamespacedName{
			// trim hash down; 20 was chosen at random
			Name:      fmt.Sprintf("obj-state-%s-%s", primaryObject.clusterName, keyHash[:20]),
			Namespace: "kcp-system",
		},
		labels:       secretLabels,
		stateCluster: stateCluster,
	}
}

func (b *kubernetesBackend) Get(obj *unstructured.Unstructured, clusterName string) ([]byte, error) {
	secret := corev1.Secret{}
	if err := b.stateCluster.client.Get(b.stateCluster.ctx, b.secretName, &secret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return nil, err
	}

	sourceKey := newObjectKey(obj, clusterName).Key()
	data, ok := secret.Data[sourceKey]
	if !ok {
		return nil, nil
	}

	return data, nil
}

func (b *kubernetesBackend) Put(obj *unstructured.Unstructured, clusterName string, data []byte) error {
	secret := corev1.Secret{}
	if err := b.stateCluster.client.Get(b.stateCluster.ctx, b.secretName, &secret); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return err
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	sourceKey := newObjectKey(obj, clusterName).Key()
	secret.Data[sourceKey] = data
	secret.Labels = b.labels

	var err error

	if secret.Namespace == "" {
		secret.Name = b.secretName.Name
		secret.Namespace = b.secretName.Namespace

		err = b.stateCluster.client.Create(b.stateCluster.ctx, &secret)
	} else {
		err = b.stateCluster.client.Update(b.stateCluster.ctx, &secret)
	}

	return err
}
