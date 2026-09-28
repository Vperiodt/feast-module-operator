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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	componentApi "github.com/opendatahub-io/feast-module-operator/api/components/v1alpha1"
	odhtypes "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
)

const (
	dataRegistryNamespaceName     = "rhoai-data-registry"
	dataRegistryEnabledLabelKey   = "dataregistry.opendatahub.io/enabled"
	dataRegistryEnabledLabelValue = "true"
)

// reconcileDataRegistryNamespace provisions the dedicated Data Registry namespace
// when the capability is enabled.
//
// Resolution order for namespace name:
//  1. spec.capabilities.dataRegistry.namespace (when spec.capabilities is present)
//  2. Default: rhoai-data-registry
func (m *Module) reconcileDataRegistryNamespace(ctx context.Context, rr *odhtypes.ReconciliationRequest) error {
	feast, ok := rr.Instance.(*componentApi.FeastOperator)
	if !ok {
		return errors.New("instance is not a FeastOperator")
	}

	_, drEnabled, drNamespace := m.resolveCapabilities(feast)
	if !drEnabled {
		return nil
	}

	log := logf.FromContext(ctx)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: drNamespace,
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
		return fmt.Errorf("failed to reconcile Data Registry namespace %s: %w", drNamespace, err)
	}

	log.V(1).Info("Reconciled Data Registry namespace",
		"namespace", drNamespace,
		"operation", op,
	)

	return nil
}
