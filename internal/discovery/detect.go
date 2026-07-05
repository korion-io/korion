/*
Copyright 2026 The Korion Authors.

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
	"errors"

	k8sdiscovery "k8s.io/client-go/discovery"
)

// ErrCRDNotInstalled is returned via DiscoveryResult.Err by an engine backed
// by an optional CRD (ArgoCD, Istio, Kyverno) when that CRD's group/version
// isn't installed in the target cluster. This is an expected, non-fatal
// condition -- the controller still surfaces it as a False "<Source>Detected"
// condition, but it never fails the whole reconcile, since most clusters
// won't have every one of these tools installed.
var ErrCRDNotInstalled = errors.New("required CRD group/version not installed in this cluster")

// GroupVersionAvailable reports whether groupVersion (e.g.
// "argoproj.io/v1alpha1") is currently served by the API server, using the
// discovery client rather than inferring absence from a failed list/get
// call. This is the one shared presence check every optional-CRD engine
// (argocd.go, istio.go, kyverno.go) uses before querying via the dynamic
// client.
func GroupVersionAvailable(disco k8sdiscovery.DiscoveryInterface, groupVersion string) bool {
	_, err := disco.ServerResourcesForGroupVersion(groupVersion)
	return err == nil
}
