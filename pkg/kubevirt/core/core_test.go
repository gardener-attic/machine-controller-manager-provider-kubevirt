// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"os"
	"testing"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	providerSpec = &api.KubeVirtProviderSpec{
		Region: "default",
		Zone:   "default",
		Resources: kubevirtv1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("4096Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
		RootVolume: cdicorev1alpha1.DataVolumeSpec{
			PVC: &corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10Gi"),
					},
				},
				StorageClassName: pointer.StringPtr("test-sc"),
			},
			Source: cdicorev1alpha1.DataVolumeSource{
				HTTP: &cdicorev1alpha1.DataVolumeSourceHTTP{
					URL: "http://test-image.com",
				},
			},
		},
	}
	machineName   = "kubevirt-machine"
	namespace     = "default"
	serverVersion = "1.18"
)

func init() {
	if err := cdicorev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		klog.Errorf("could not execute tests: %v", err)
		os.Exit(1)
	}
}

func TestPluginSPIImpl_CreateMachine(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("CreateMachine", func(t *testing.T) {
		mf := newMockFactory(fakeClient, namespace, serverVersion)
		plugin := NewPluginSPIImpl(mf, mf)

		_, err := plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to create machine: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to list machines: %v", err)
		}

		if len(machineList) != 1 {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_GetMachineStatus(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("GetMachineStatus", func(t *testing.T) {
		mf := newMockFactory(fakeClient, namespace, serverVersion)
		plugin := NewPluginSPIImpl(mf, mf)

		_, err := plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to create machine: %v", err)
		}

		providerID, err := plugin.GetMachineStatus(context.Background(), machineName, "", providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to get machine status: %v", err)
		}

		if providerID != ProviderName+"://"+machineName {
			t.Fatal("provider id doesn't match the expected value")
		}
	})
}

func TestPluginSPIImpl_ListMachines(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("ListMachines", func(t *testing.T) {
		mf := newMockFactory(fakeClient, namespace, serverVersion)
		plugin := NewPluginSPIImpl(mf, mf)

		_, err := plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to create machine: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to list machines: %v", err)
		}

		if len(machineList) != 1 {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_ShutDownMachine(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("ShutDownMachine", func(t *testing.T) {
		mf := newMockFactory(fakeClient, namespace, serverVersion)
		plugin := NewPluginSPIImpl(mf, mf)

		providerID, err := plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to create machine: %v", err)
		}

		_, err = plugin.ShutDownMachine(context.Background(), machineName, providerID, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to shutdown machine: %v", err)
		}

		vm, err := plugin.getVM(context.Background(), fakeClient, machineName, namespace)
		if err != nil {
			t.Fatalf("failed to get VM: %v", err)
		}

		if *vm.Spec.Running {
			t.Fatal("machine is still running")
		}
	})
}

func TestPluginSPIImpl_DeleteMachine(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("DeleteMachine", func(t *testing.T) {
		mf := newMockFactory(fakeClient, namespace, serverVersion)
		plugin := NewPluginSPIImpl(mf, mf)

		providerID, err := plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to create machine: %v", err)
		}

		_, err = plugin.DeleteMachine(context.Background(), machineName, providerID, providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to delete machine: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to list machines: %v", err)
		}

		if len(machineList) != 0 {
			t.Fatalf("unexpected machine count")
		}
	})
}

type mockFactory struct {
	client        client.Client
	namespace     string
	serverVersion string
}

func newMockFactory(client client.Client, namespace, serverVersion string) *mockFactory {
	return &mockFactory{
		client:        client,
		namespace:     namespace,
		serverVersion: serverVersion,
	}
}

func (cf mockFactory) GetClient(secret *corev1.Secret) (client.Client, string, error) {
	return cf.client, cf.namespace, nil
}

func (cf mockFactory) GetServerVersion(secret *corev1.Secret) (string, error) {
	return cf.serverVersion, nil
}
