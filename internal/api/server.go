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

// Package api exposes a small read-only HTTP API over the manager's
// controller-runtime cache. This is the only path by which the browser-based
// frontend ever learns about cluster state -- it never talks to the K8s API
// server directly, so no cluster credentials are exposed to client-side JS.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	korionv1alpha1 "github.com/korion-io/korion/api/v1alpha1"
)

// Server is a controller-runtime manager.Runnable serving the read-only
// PlatformMap API off the manager's client (backed by its in-memory cache,
// not a live API-server round trip per request).
type Server struct {
	Client client.Client
	// Addr is the address to listen on, e.g. ":8082".
	Addr string
}

// Start implements manager.Runnable. It blocks until ctx is cancelled, then
// shuts the HTTP server down gracefully.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/platformmaps/{namespace}/{name}", s.getPlatformMap)

	httpServer := &http.Server{Addr: s.Addr, Handler: withCORS(mux)}

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.ListenAndServe() }()

	select {
	case <-ctx.Done():
		return httpServer.Shutdown(context.Background())
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// withCORS allows the ui/ dev server (a different origin during local
// development) to call this API. In production the frontend is served
// same-origin behind the Helm-installed Service, so this is a dev
// convenience, not a security boundary -- the API is read-only and exposes
// no credentials.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getPlatformMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var pm korionv1alpha1.PlatformMap
	if err := s.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &pm); err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, fmt.Sprintf("platformmap %s/%s not found", namespace, name), http.StatusNotFound)
			return
		}
		log.FromContext(ctx).Error(err, "getting platformmap", "namespace", namespace, "name", name)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pm); err != nil {
		log.FromContext(ctx).Error(err, "encoding response")
	}
}
