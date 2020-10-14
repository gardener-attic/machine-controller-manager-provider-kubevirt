package kubevirt

import (
	"encoding/json"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/core"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/validation"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// decodeProviderSpecAndSecret decodes the provider spec from the given machine class and validates it, together with the given secret.
func decodeProviderSpecAndSecret(machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*api.KubeVirtProviderSpec, error) {
	var spec *api.KubeVirtProviderSpec
	if err := json.Unmarshal(machineClass.ProviderSpec.Raw, &spec); err != nil {
		wrapped := errors.Wrap(err, "could not unmarshal provider spec from JSON")
		klog.V(2).Infof(wrapped.Error())
		return nil, status.Error(codes.Internal, wrapped.Error())
	}

	if errs := validation.ValidateKubevirtProviderSpec(spec); len(errs) > 0 {
		err := errors.Errorf("could not validate provider spec: %v", errs)
		klog.V(2).Infof(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	if secret == nil {
		err := errors.New("provider secret is nil")
		klog.V(2).Infof(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	if errs := validation.ValidateKubevirtProviderSecret(secret); len(errs) > 0 {
		err := errors.Errorf("could not validate provider secret: %v", errs)
		klog.V(2).Infof(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return spec, nil
}

// wrapf wraps the given error in a status.Error.
func wrapf(err error, format string, args ...interface{}) error {
	var (
		code    codes.Code
		wrapped error
	)
	switch err.(type) {
	case *core.MachineNotFoundError:
		code = codes.NotFound
		wrapped = err
	default:
		code = codes.Internal
		wrapped = errors.Wrapf(err, format, args...)
	}
	klog.V(2).Infof(wrapped.Error())
	return status.Error(code, wrapped.Error())
}
