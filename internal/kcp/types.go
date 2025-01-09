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

package kcp

const (
	// RootWorkspace is kcp's root workspace name. This should only be changed
	// if it changes in kcp.
	RootWorkspace = "root"

	// IdentityClusterName is the name of the logicalcluster that is backing the
	// current kcp workspace. Within each kcp workspace one can query for this
	// logicalcluster to resolve the workspace's path (e.g. "root:org1:teamx")
	// the logicalcluster name (e.g. "984235jkhwfowt45").
	IdentityClusterName = "cluster"
)
