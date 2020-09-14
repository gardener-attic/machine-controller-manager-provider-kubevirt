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
	"testing"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	providerSpec = &api.KubeVirtProviderSpec{
		SourceURL:        "http://test-image.com",
		StorageClassName: "test-sc",
		PVCSize:          "10Gi",
		CPUs:             "1",
		Memory:           "4096M",
	}
	machineName = "kubevirt-machine"
	namespace   = "default"
)

func TestPluginSPIImpl_CreateMachine(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	t.Run("CreateMachine", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(newMockClientFactory(fakeClient, namespace))
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
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
		plugin, err := NewPluginSPIImpl(newMockClientFactory(fakeClient, namespace))
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
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
		plugin, err := NewPluginSPIImpl(newMockClientFactory(fakeClient, namespace))
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), machineName, providerSpec, &corev1.Secret{})
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
		plugin, err := NewPluginSPIImpl(newMockClientFactory(fakeClient, namespace))
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

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
		plugin, err := NewPluginSPIImpl(newMockClientFactory(fakeClient, namespace))
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

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

type mockClientFactory struct {
	client    client.Client
	namespace string
}

func newMockClientFactory(client client.Client, namespace string) *mockClientFactory {
	return &mockClientFactory{
		client:    client,
		namespace: namespace,
	}
}

func (cf mockClientFactory) GetClient(secret *corev1.Secret) (client.Client, string, error) {
	return cf.client, cf.namespace, nil
}
