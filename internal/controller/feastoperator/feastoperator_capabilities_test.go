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
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	componentApi "github.com/opendatahub-io/feast-module-operator/api/components/v1alpha1"
	moduleconfig "github.com/opendatahub-io/feast-module-operator/pkg/config"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/cluster"
	odhtypes "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/metadata/labels"
)

func initCapabilitiesTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(componentApi.AddToScheme(scheme))
	return scheme
}

func newCapabilitiesTestModule(t *testing.T, featureStoreEnabled, dataRegistryEnabled bool) *Module {
	t.Helper()

	repoRoot := filepath.Join("..", "..", "..")
	cfg := &moduleconfig.Config{
		PlatformName:          string(cluster.OpenDataHub),
		PlatformVersion:       "1.0.0",
		ManifestsPath:         filepath.Join(repoRoot, "config", "manifests"),
		ApplicationsNamespace: "test-ns",
		FeatureStoreEnabled:   featureStoreEnabled,
		DataRegistryEnabled:   dataRegistryEnabled,
	}

	m, err := NewModule(cfg)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	return m
}

func newCapabilitiesRR(t *testing.T, cl client.Client, obj *componentApi.FeastOperator, m *Module) *odhtypes.ReconciliationRequest {
	t.Helper()
	g := NewWithT(t)

	repoRoot := filepath.Join("..", "..", "..")
	rr := &odhtypes.ReconciliationRequest{
		Instance:          obj,
		Client:            cl,
		ManifestsBasePath: filepath.Join(repoRoot, "config", "manifests"),
		Release: (&moduleconfig.Config{
			PlatformName:    string(cluster.OpenDataHub),
			PlatformVersion: "1.0.0",
		}).Release(),
	}
	g.Expect(m.initialize(context.Background(), rr)).To(Succeed())
	return rr
}

func TestReconcileCapabilitiesConfigMapCreatesConfigMap(t *testing.T) {
	g := NewWithT(t)

	scheme := initCapabilitiesTestScheme()
	feast := newTestFeastOperator()
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(feast).Build()

	m := newCapabilitiesTestModule(t, true, false)
	rr := newCapabilitiesRR(t, cl, feast, m)

	g.Expect(m.reconcileCapabilitiesConfigMap(context.Background(), rr)).To(Succeed())

	cm := &corev1.ConfigMap{}
	g.Expect(cl.Get(context.Background(), client.ObjectKey{
		Name:      capabilitiesConfigMapName,
		Namespace: "test-ns",
	}, cm)).To(Succeed())
	g.Expect(cm.Data[capabilitiesKeyFeatureStoreEnabled]).To(Equal("true"))
	g.Expect(cm.Data[capabilitiesKeyDataRegistryEnabled]).To(Equal("false"))
	g.Expect(cm.Labels[labels.ODH.Component(componentName)]).To(Equal(labels.True))
	g.Expect(cm.OwnerReferences).To(HaveLen(1))
	g.Expect(cm.OwnerReferences[0].Name).To(Equal(componentApi.FeastOperatorInstanceName))
}

func TestReconcileCapabilitiesConfigMapUpdatesExistingConfigMap(t *testing.T) {
	g := NewWithT(t)

	scheme := initCapabilitiesTestScheme()
	feast := newTestFeastOperator()
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      capabilitiesConfigMapName,
			Namespace: "test-ns",
		},
		Data: map[string]string{
			capabilitiesKeyFeatureStoreEnabled: "true",
			capabilitiesKeyDataRegistryEnabled: "true",
		},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(feast, existing).Build()

	m := newCapabilitiesTestModule(t, false, true)
	rr := newCapabilitiesRR(t, cl, feast, m)

	g.Expect(m.reconcileCapabilitiesConfigMap(context.Background(), rr)).To(Succeed())

	cm := &corev1.ConfigMap{}
	g.Expect(cl.Get(context.Background(), client.ObjectKey{
		Name:      capabilitiesConfigMapName,
		Namespace: "test-ns",
	}, cm)).To(Succeed())
	g.Expect(cm.Data[capabilitiesKeyFeatureStoreEnabled]).To(Equal("false"))
	g.Expect(cm.Data[capabilitiesKeyDataRegistryEnabled]).To(Equal("true"))
}
