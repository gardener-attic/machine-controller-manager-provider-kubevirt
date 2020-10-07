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

// Package validation is used to validate cloud specific KubeVirtProviderSpec
package validation

import (
	"fmt"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

// ValidateKubevirtProviderSpec validates the given kubevirt provider spec.
func ValidateKubevirtProviderSpec(spec *api.KubeVirtProviderSpec) field.ErrorList {
	errs := field.ErrorList{}

	requestsPath := field.NewPath("resources").Child("requests")
	if spec.Resources.Requests.Memory().IsZero() {
		errs = append(errs, field.Required(requestsPath.Child("memory"), "cannot be zero"))
	}
	if spec.Resources.Requests.Cpu().IsZero() {
		errs = append(errs, field.Required(requestsPath.Child("cpu"), "cannot be zero"))
	}

	errs = append(errs, validateDataVolume(field.NewPath("rootVolume"), &spec.RootVolume)...)

	for i, volume := range spec.AdditionalVolumes {
		volumePath := field.NewPath("additionalVolumes").Index(i)

		switch {
		case volume.DataVolume != nil:
			errs = append(errs, validateDataVolume(volumePath.Child("dataVolume"), volume.DataVolume)...)
		case volume.VolumeSource != nil:
			break
		default:
			errs = append(errs, field.Invalid(volumePath, volume, "invalid volume, either dataVolume or volumeSource must be specified"))
		}
	}

	if spec.Region == "" {
		errs = append(errs, field.Required(field.NewPath("region"), "cannot be empty"))
	}

	if spec.Zone == "" {
		errs = append(errs, field.Required(field.NewPath("zone"), "cannot be empty"))
	}

	if spec.DNSPolicy != "" {
		dnsPolicyPath := field.NewPath("dnsPolicy")
		dnsConfigPath := field.NewPath("dnsConfig")

		switch spec.DNSPolicy {
		case corev1.DNSDefault, corev1.DNSClusterFirstWithHostNet, corev1.DNSClusterFirst, corev1.DNSNone:
			break
		default:
			errs = append(errs, field.Invalid(dnsPolicyPath, spec.DNSPolicy, "invalid DNS policy"))
		}

		if spec.DNSPolicy == corev1.DNSNone {
			if spec.DNSConfig == nil {
				errs = append(errs, field.Required(dnsConfigPath, fmt.Sprintf("cannot be empty when DNS policy is %s", corev1.DNSNone)))
			} else if len(spec.DNSConfig.Nameservers) == 0 {
				errs = append(errs, field.Required(dnsConfigPath.Child("nameservers"), fmt.Sprintf("cannot be empty when DNS policy is %s", corev1.DNSNone)))
			}
		}
	}

	return errs
}

// ValidateKubevirtProviderSecret validates the given kubevirt provider secret.
func ValidateKubevirtProviderSecret(secret *corev1.Secret) field.ErrorList {
	errs := field.ErrorList{}

	if kubeconfig, ok := secret.Data["kubeconfig"]; !ok || len(kubeconfig) == 0 {
		errs = append(errs, field.Required(field.NewPath("kubeconfig"), "cannot be empty"))
	} else if _, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig); err != nil {
		errs = append(errs, field.Invalid(field.NewPath("kubeconfig"), kubeconfig, fmt.Sprintf("could not get client config: %v", err)))
	}

	if userData, ok := secret.Data["userData"]; !ok || len(userData) == 0 {
		errs = append(errs, field.Required(field.NewPath("userData"), "cannot be empty"))
	}

	return errs
}

func validateDataVolume(path *field.Path, dataVolume *cdicorev1alpha1.DataVolumeSpec) field.ErrorList {
	errs := field.ErrorList{}

	pvcPath := path.Child("pvc")
	if dataVolume.PVC == nil {
		errs = append(errs, field.Required(pvcPath, "cannot be empty"))
	} else {
		if storage(&dataVolume.PVC.Resources.Requests).IsZero() {
			errs = append(errs, field.Required(pvcPath.Child("resources").Child("requests").Child("storage"), "cannot be zero"))
		}
	}

	return errs
}

func storage(resources *corev1.ResourceList) *resource.Quantity {
	if val, ok := (*resources)[corev1.ResourceStorage]; ok {
		return &val
	}
	return &resource.Quantity{}
}
