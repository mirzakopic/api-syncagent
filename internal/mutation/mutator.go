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

package mutation

import (
	"fmt"

	syncagentv1alpha1 "github.com/kcp-dev/api-syncagent/sdk/apis/syncagent/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Mutator interface {
	// MutateSpec transform a remote object into a local one. On the first
	// mutation, otherObj will be nil. MutateSpec can modify all fields
	// except the status (i.e. "mutate spec" here means to mutate expected state,
	// which can be more than just the spec).
	MutateSpec(toMutate *unstructured.Unstructured, otherObj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	// MutateStatus transform a local object into a remote one. MutateStatus
	// must only modify the status field.
	MutateStatus(toMutate *unstructured.Unstructured, otherObj *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

type mutator struct {
	spec *syncagentv1alpha1.ResourceMutationSpec
}

var _ Mutator = &mutator{}

// NewMutator creates a new mutator, which will apply the mutation rules to a synced object, in
// both directions. A nil spec is supported and will simply make the mutator not do anything.
func NewMutator(spec *syncagentv1alpha1.ResourceMutationSpec) Mutator {
	return &mutator{
		spec: spec,
	}
}

func (m *mutator) MutateSpec(toMutate *unstructured.Unstructured, otherObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.spec == nil || m.spec.Spec == nil {
		return toMutate, nil
	}

	ctx := &TemplateMutationContext{
		RemoteObject: toMutate.Object,
	}

	if otherObj != nil {
		ctx.LocalObject = otherObj.Object
	}

	mutatedObj, err := ApplyResourceMutations(toMutate.Object, m.spec.Spec, ctx)
	if err != nil {
		return nil, err
	}

	obj, ok := mutatedObj.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mutations did not yield an object, but %T", mutatedObj)
	}

	toMutate.Object = obj

	return toMutate, nil
}

func (m *mutator) MutateStatus(toMutate *unstructured.Unstructured, otherObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.spec == nil || m.spec.Status == nil {
		return toMutate, nil
	}

	ctx := &TemplateMutationContext{
		LocalObject: toMutate.Object,
	}

	if otherObj != nil {
		ctx.RemoteObject = otherObj.Object
	}

	mutatedObj, err := ApplyResourceMutations(toMutate.Object, m.spec.Status, ctx)
	if err != nil {
		return nil, err
	}

	obj, ok := mutatedObj.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mutations did not yield an object, but %T", mutatedObj)
	}

	toMutate.Object = obj

	return toMutate, nil
}
