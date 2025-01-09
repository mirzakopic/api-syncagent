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

/*
Package apiexport contains a controller that watches for PublishedResources
and then maintains a singular APIExport for all found PR's. The controller
only includes PR's that already have an APIResourceSchema name attached,
created by the accompanying controller in the Sync Agent.

Note that for the time being, to prevent data loss, only new ARS will be added to
the APIExport. Once an ARS is listed in the APIExport, it is supposed to remain
until an administrator/other process performs garbage collection in the platform.
*/
package apiexport
