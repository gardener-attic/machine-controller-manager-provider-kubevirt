package kubevirt

import (
	"encoding/json"
	"fmt"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	clouderrors "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/validation"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// decodeProviderSpecAndSecret converts request parameters to api.ProviderSpec
func decodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.KubeVirtProviderSpec, error) {
	var (
		providerSpec *api.KubeVirtProviderSpec
	)

	// Extract providerSpec
	err := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
	if err != nil {
		wrapped := errors.Wrap(err, "could not unmarshal provider spec from JSON")
		klog.V(2).Infof(wrapped.Error())
		return nil, status.Error(codes.Internal, wrapped.Error())
	}

	if errs := validation.ValidateKubevirtProviderSpec(providerSpec); errs != nil {
		err = fmt.Errorf("could not validate provider spec: %v", errs)
		klog.V(2).Infof(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	if errs := validation.ValidateKubevirtProviderSecrets(secret); errs != nil {
		err = fmt.Errorf("could not validate provider secrets: %v", errs)
		klog.V(2).Infof(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return providerSpec, nil
}

// prepareErrorf preapre, format and wrap an error on the machine server level.
func prepareErrorf(err error, format string, args ...interface{}) error {
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
		wrapped = errors.Wrapf(err, format, args...)
	}
	klog.V(2).Infof(wrapped.Error())
	return status.Error(code, wrapped.Error())
}
