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

const (
	// deletionFinalizer is the finalizer put on remote objects to prevent
	// them from being deleted before the local objects can be cleaned up.
	deletionFinalizer = "servlet.kdp.k8c.io/cleanup"

	// The following 3 labels are put on local objects to link them to their
	// origin remote objects.

	remoteObjectClusterLabel   = "servlet.kdp.k8c.io/remote-object-cluster"
	remoteObjectNamespaceLabel = "servlet.kdp.k8c.io/remote-object-namespace"
	remoteObjectNameLabel      = "servlet.kdp.k8c.io/remote-object-name"

	// objectStateLabelName is put on object state Secrets to allow for easier mass deletions
	// if ever necessary.
	objectStateLabelName = "servlet.kdp.k8c.io/object-state"

	// objectStateLabelValue is the value of the objectStateLabelName label.
	objectStateLabelValue = "true"

	// relatedObjectAnnotationPrefix is the prefix for the annotation that is placed on
	// objects in the kcp workspaces, informing the user about the existence of a related
	// object. The identifier of the related object is appended to this to form the
	// full annotation name, the annotation value is a JSON string containing GVK and
	// metadata of the related object.
	relatedObjectAnnotationPrefix = "related-resources.kdp.k8c.io/"
)
