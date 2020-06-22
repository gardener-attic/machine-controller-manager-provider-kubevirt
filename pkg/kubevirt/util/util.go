package util

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	api "github.com/moadqassem/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	clouderrors "github.com/moadqassem/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"
	"github.com/pkg/errors"
	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// DecodeProviderSpecAndSecret converts request parameters to api.ProviderSpec
func DecodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass) (*api.KubeVirtProviderSpec, error) {
	var (
		providerSpec *api.KubeVirtProviderSpec
	)

	// Extract providerSpec
	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return providerSpec, nil
}

func PrepareErrorf(err error, format string, args ...interface{}) error {
	var (
		code    codes.Code
		wrapped error
	)
	switch err.(type) {
	case *clouderrors.MachineNotFoundError:
		code = codes.NotFound
		wrapped = err
	default:
		code = codes.Internal
		wrapped = errors.Wrap(err, fmt.Sprintf(format, args...))
	}
	klog.V(2).Infof(wrapped.Error())
	return status.Error(code, wrapped.Error())
}

func DNSPolicy(policy string) (corev1.DNSPolicy, error) {
	switch policy {
	case string(corev1.DNSClusterFirstWithHostNet):
		return corev1.DNSClusterFirstWithHostNet, nil
	case string(corev1.DNSClusterFirst):
		return corev1.DNSClusterFirst, nil
	case string(corev1.DNSDefault):
		return corev1.DNSDefault, nil
	case string(corev1.DNSNone):
		return corev1.DNSNone, nil
	}

	return "", fmt.Errorf("unknown dns policy: %s", policy)
}

func ParseResources(cpus, memory string) (*corev1.ResourceList, error) {
	memoryResource, err := resource.ParseQuantity(memory)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory requests: %v", err)
	}
	cpuResource, err := resource.ParseQuantity(cpus)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpu request: %v", err)
	}
	return &corev1.ResourceList{
		corev1.ResourceMemory: memoryResource,
		corev1.ResourceCPU:    cpuResource,
	}, nil
}
