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

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	korionv1alpha1 "github.com/korion-io/korion/api/v1alpha1"
)

func TestServer_GetPlatformMap_Found(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := korionv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}
	pm := &korionv1alpha1.PlatformMap{
		ObjectMeta: metav1.ObjectMeta{Name: "superheros-platform", Namespace: "superheros"},
		Spec:       korionv1alpha1.PlatformMapSpec{Namespace: "superheros"},
		Status: korionv1alpha1.PlatformMapStatus{
			Topology: &runtime.RawExtension{Raw: []byte(`{"nodes":[],"edges":[]}`)},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pm).Build()

	s := &Server{Client: c}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platformmaps/superheros/superheros-platform", nil)
	req.SetPathValue("namespace", "superheros")
	req.SetPathValue("name", "superheros-platform")
	rec := httptest.NewRecorder()

	s.getPlatformMap(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got korionv1alpha1.PlatformMap
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if got.Name != "superheros-platform" || got.Namespace != "superheros" {
		t.Errorf("got %+v", got.ObjectMeta)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
}

func TestServer_GetPlatformMap_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := korionv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	s := &Server{Client: c}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platformmaps/superheros/missing", nil)
	req.SetPathValue("namespace", "superheros")
	req.SetPathValue("name", "missing")
	rec := httptest.NewRecorder()

	s.getPlatformMap(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServer_CORSHeaders(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := korionv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	s := &Server{Client: c}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/platformmaps/{namespace}/{name}", s.getPlatformMap)
	handler := withCORS(mux)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/platformmaps/ns/name", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing CORS header")
	}
}
