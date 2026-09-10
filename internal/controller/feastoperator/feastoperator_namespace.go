/*
Copyright 2026.

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

package feastoperator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	odhtypes "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
)

const (
	dataRegistryNamespaceName      = "rhoai-data-registry"
	dataRegistryEnabledLabelKey      = "dataregistry.opendatahub.io/enabled"
	dataRegistryEnabledLabelValue    = "true"
)

// reconcileDataRegistryNamespace provisions the dedicated Data Registry namespace
// when the capability is enabled. EA2 hardcodes rhoai-data-registry.
func (m *Module) reconcileDataRegistryNamespace(ctx context.Context, rr *odhtypes.ReconciliationRequest) error {
	if !m.cfg.DataRegistryEnabled {
		return nil
	}

	log := logf.FromContext(ctx)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataRegistryNamespaceName,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, rr.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[dataRegistryEnabledLabelKey] = dataRegistryEnabledLabelValue
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile Data Registry namespace %s: %w", dataRegistryNamespaceName, err)
	}

	log.V(1).Info("Reconciled Data Registry namespace",
		"namespace", dataRegistryNamespaceName,
		"operation", op,
	)

	return nil
}
