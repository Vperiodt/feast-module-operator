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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	componentApi "github.com/opendatahub-io/feast-module-operator/api/components/v1alpha1"
	odhtypes "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
	odhdeploy "github.com/opendatahub-io/opendatahub-operator/v2/pkg/deploy"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/metadata/labels"
)

const (
	capabilitiesConfigMapName = "feast-capabilities-config"

	capabilitiesKeyFeatureStoreEnabled = "featureStoreEnabled"
	capabilitiesKeyDataRegistryEnabled = "dataRegistryEnabled"

	paramsEnvKeyFeatureStoreEnabled = "FEATURE_STORE_ENABLED"
	paramsEnvKeyDataRegistryEnabled = "DATA_REGISTRY_ENABLED"
)

func boolString(value bool) string {
	return strconv.FormatBool(value)
}

// reconcileCapabilitiesConfigMap projects capability toggles into params.env and the
// feast-capabilities-config ConfigMap consumed by the upstream feast-operator.
func (m *Module) reconcileCapabilitiesConfigMap(ctx context.Context, rr *odhtypes.ReconciliationRequest) error {
	log := logf.FromContext(ctx)

	feast, ok := rr.Instance.(*componentApi.FeastOperator)
	if !ok {
		return errors.New("instance is not a FeastOperator")
	}

	if len(rr.Manifests) == 0 {
		return errors.New("no manifests initialized before reconcileCapabilitiesConfigMap")
	}

	capabilityParams := map[string]string{
		paramsEnvKeyFeatureStoreEnabled: boolString(m.cfg.FeatureStoreEnabled),
		paramsEnvKeyDataRegistryEnabled: boolString(m.cfg.DataRegistryEnabled),
	}

	if err := odhdeploy.ApplyParams(rr.Manifests[0].String(), "params.env", nil, capabilityParams); err != nil {
		return fmt.Errorf("failed to update params.env with capability parameters: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capabilitiesConfigMapName,
			Namespace: m.cfg.ApplicationsNamespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, rr.Client, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}
		cm.Labels[labels.ODH.Component(componentName)] = labels.True
		cm.Data = map[string]string{
			capabilitiesKeyFeatureStoreEnabled: boolString(m.cfg.FeatureStoreEnabled),
			capabilitiesKeyDataRegistryEnabled: boolString(m.cfg.DataRegistryEnabled),
		}
		return controllerutil.SetControllerReference(feast, cm, rr.Client.Scheme())
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile capabilities ConfigMap: %w", err)
	}

	log.V(1).Info("Reconciled capabilities ConfigMap",
		"configmap", capabilitiesConfigMapName,
		"namespace", m.cfg.ApplicationsNamespace,
		"operation", op,
	)

	return nil
}
