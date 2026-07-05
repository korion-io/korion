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
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

func TestGroupVersionAvailable(t *testing.T) {
	// A fake clientset with no registered API resources reports every
	// groupVersion as unavailable -- this is the "optional tool not
	// installed" case every engine must tolerate.
	clientset := fake.NewClientset()

	if GroupVersionAvailable(clientset.Discovery(), "argoproj.io/v1alpha1") {
		t.Error("expected argoproj.io/v1alpha1 to be unavailable against an empty fake discovery client")
	}
}
