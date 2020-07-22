package core

import (
	"context"
	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"testing"
)

var testCase = struct {
	name                  string
	machineName           string
	providerSpec          *api.KubeVirtProviderSpec
	secret                *corev1.Secret
	expectedMachinesCount int
}{
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
	secret: &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-machine-secret",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"kubeconfig": []byte("kubeconfig-test"),
			"userData":   []byte("userdata-test"),
		},
	},
	expectedMachinesCount: 1,
}

func TestPluginSPIImpl_CreateMachine(t *testing.T) {
	t.Run("create kubvirt machine", func(t *testing.T) {
		plugin := &PluginSPIImpl{
			client: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		_, err := plugin.CreateMachine(context.Background(), testCase.machineName, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) != testCase.expectedMachinesCount {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_GetMachineStatus(t *testing.T) {
	t.Run("get kubvirt machine status", func(t *testing.T) {
		plugin := &PluginSPIImpl{
			client: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		providerID, err := plugin.CreateMachine(context.Background(), testCase.machineName, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		fetchedProviderID, err := plugin.GetMachineStatus(context.Background(), testCase.machineName, providerID, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		if providerID != fetchedProviderID {
			t.Fatal("failed to fetch the right machine status")
		}
	})
}

func TestPluginSPIImpl_ListMachines(t *testing.T) {
	t.Run("list kubvirt machines", func(t *testing.T) {
		plugin := &PluginSPIImpl{
			client: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		_, err := plugin.CreateMachine(context.Background(), testCase.machineName, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) != testCase.expectedMachinesCount {
			t.Fatal("unexpected machine count")
		}
	})
}

func TestPluginSPIImpl_ShutDownMachine(t *testing.T) {
	t.Run("shutdown kubvirt machine", func(t *testing.T) {
		plugin := &PluginSPIImpl{
			client: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		providerID, err := plugin.CreateMachine(context.Background(), testCase.machineName, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		_, err = plugin.ShutDownMachine(context.Background(), testCase.machineName, providerID, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("failed to test kubevirt machine shutdown: %v", err)
		}

		vm, err := plugin.getVM(testCase.secret, testCase.machineName, testCase.providerSpec.Namespace)
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
		plugin := &PluginSPIImpl{
			client: fake.NewFakeClientWithScheme(scheme.Scheme),
		}

		providerID, err := plugin.CreateMachine(context.Background(), testCase.machineName, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while testing machine creation: %v", err)
		}

		_, err = plugin.DeleteMachine(context.Background(), testCase.machineName, providerID, testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("failed to delete kubevrit machine: %v", err)
		}

		machineList, err := plugin.ListMachines(context.Background(), testCase.providerSpec, testCase.secret)
		if err != nil {
			t.Fatalf("error has occurred while listing machines: %v", err)
		}

		if len(machineList) > 0 {
			t.Fatalf("unexpected machine count! failed to delete machine")
		}
	})
}
