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
	testCase = []struct {
		name                  string
		machineName           string
		providerSpec          *api.KubeVirtProviderSpec
		expectedMachinesCount int
		expectedProviderID    string
		fakeClient            client.Client
	}{
		{
			name:        "test kubevirt machine creation",
			machineName: "kubevirt-machine",
			providerSpec: &api.KubeVirtProviderSpec{
				SourceURL:        "http://test-image.com",
				StorageClassName: "test-sc",
				PVCSize:          "10Gi",
				CPUs:             "1",
				Memory:           "4096M",
				Namespace:        "kube-system",
			},
			expectedMachinesCount: 1,
			fakeClient:            fake.NewFakeClientWithScheme(scheme.Scheme),
		},
		{
			name:                  "test kubevirt get machine status",
			machineName:           "kubevirt-machine",
			expectedMachinesCount: 1,
			providerSpec: &api.KubeVirtProviderSpec{
				Namespace: "kube-system",
			},
			fakeClient: fake.NewFakeClientWithScheme(scheme.Scheme),
		},
		{
			name:                  "test kubevirt list machines",
			machineName:           "kubevirt-machine",
			expectedMachinesCount: 1,
			providerSpec: &api.KubeVirtProviderSpec{
				Namespace: "kube-system",
			},
			fakeClient: fake.NewFakeClientWithScheme(scheme.Scheme),
		},
		{
			name:                  "test kubevirt shutdown machine",
			machineName:           "kubevirt-machine",
			expectedMachinesCount: 1,
			providerSpec: &api.KubeVirtProviderSpec{
				Namespace: "kube-system",
			},
			fakeClient: fake.NewFakeClientWithScheme(scheme.Scheme),
		},
		{
			name:                  "test kubevirt delete machine",
			machineName:           "kubevirt-machine",
			expectedMachinesCount: 1,
			providerSpec: &api.KubeVirtProviderSpec{
				Namespace: "kube-system",
			},
			fakeClient: fake.NewFakeClientWithScheme(scheme.Scheme),
		},
	}
)

func TestPluginSPIImpl_CreateMachine(t *testing.T) {
	t.Run("create kubvirt machine", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(mockClient(testCase[0].fakeClient))
		if err != nil {
			t.Fatalf("failed to create a mock client: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), testCase[0].machineName, testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) != testCase[0].expectedMachinesCount {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_GetMachineStatus(t *testing.T) {
	t.Run("get kubvirt machine status", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(mockClient(testCase[1].fakeClient))
		if err != nil {
			t.Fatalf("failed to create a mock client: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), testCase[0].machineName, testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		providerID, err := plugin.GetMachineStatus(context.Background(), testCase[1].machineName, testCase[1].expectedProviderID, testCase[1].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		if providerID != testCase[1].expectedProviderID {
			t.Fatal("failed to fetch the right machine status")
		}
	})
}

func TestPluginSPIImpl_ListMachines(t *testing.T) {
	t.Run("list kubvirt machines", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(mockClient(testCase[2].fakeClient))
		if err != nil {
			t.Fatalf("failed to create a mock client: %v", err)
		}

		_, err = plugin.CreateMachine(context.Background(), testCase[0].machineName, testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase[2].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) != testCase[2].expectedMachinesCount {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_ShutDownMachine(t *testing.T) {
	t.Run("shutdown kubvirt machine", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(mockClient(testCase[3].fakeClient))
		if err != nil {
			t.Fatalf("failed to create a mock client: %v", err)
		}

		providerID, err := plugin.CreateMachine(context.Background(), testCase[0].machineName, testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		_, err = plugin.ShutDownMachine(context.Background(), testCase[3].machineName, providerID, testCase[3].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to test kubevirt machine shutdown: %v", err)
		}

		vm, err := plugin.getVM(context.Background(), &corev1.Secret{}, testCase[3].machineName, testCase[3].providerSpec.Namespace)
		if err != nil {
			t.Fatalf("failed to fetch kubevirt vm: %v", err)
		}

		if *vm.Spec.Running {
			t.Fatal("machine is still running! kubevirt machine shutdown failed")
		}
	})
}

func TestPluginSPIImpl_DeleteMachine(t *testing.T) {
	t.Run("delete kubvirt machine", func(t *testing.T) {
		plugin, err := NewPluginSPIImpl(mockClient(testCase[4].fakeClient))
		if err != nil {
			t.Fatalf("failed to create a mock client: %v", err)
		}

		providerID, err := plugin.CreateMachine(context.Background(), testCase[4].machineName, testCase[0].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		_, err = plugin.DeleteMachine(context.Background(), testCase[4].machineName, providerID, testCase[4].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("failed to delete kubevrit machine: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase[4].providerSpec, &corev1.Secret{})
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) > 0 {
			t.Fatalf("unexpected machine count! failed to delete machine")
		}
	})
}

func mockClient(c client.Client) ClientFunc {
	return func(secret *corev1.Secret) (client.Client, error) {
		return c, nil
	}
}
