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
	"errors"
	"fmt"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
)

// ValidateKubevirtProviderSpec validates kubevirt spec to check if all fields are present and valid
func ValidateKubevirtProviderSpec(spec *api.KubeVirtProviderSpec) field.ErrorList {
	errs := field.ErrorList{}

	requestsPath := field.NewPath("resources").Child("requests")
	if spec.Resources.Requests.Memory().IsZero() {
		errs = append(errs, field.Required(requestsPath.Child("memory"), "cannot be zero"))
	}
	if spec.Resources.Requests.Cpu().IsZero() {
		errs = append(errs, field.Required(requestsPath.Child("cpu"), "cannot be zero"))
	}

	if spec.SourceURL == "" {
		errs = append(errs, field.Required(field.NewPath("sourceURL"), "cannot be empty"))
	}

	if spec.StorageClassName == "" {
		errs = append(errs, field.Required(field.NewPath("storageClassName"), "cannot be empty"))
	}

	if spec.PVCSize.IsZero() {
		errs = append(errs, field.Required(field.NewPath("pvcSize"), "cannot be zero"))
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
			errs = append(errs, field.Invalid(dnsPolicyPath, spec.DNSPolicy, "invalid dns policy"))
		}

		if spec.DNSPolicy == corev1.DNSNone {
			if spec.DNSConfig != nil {
				if len(spec.DNSConfig.Nameservers) == 0 {
					errs = append(errs, field.Required(dnsConfigPath.Child("nameservers"),
						fmt.Sprintf("cannot be empty when dns policy is %s", corev1.DNSNone)))
				}
			} else {
				errs = append(errs, field.Required(dnsConfigPath,
					fmt.Sprintf("cannot be empty when dns policy is %s", corev1.DNSNone)))
			}
		}
	}

	return errs
}

// ValidateKubevirtProviderSecrets validates kubevirt secrets
func ValidateKubevirtProviderSecrets(secret *corev1.Secret) []error {
	var errs []error

	if secret == nil {
		errs = append(errs, errors.New("secret object passed by the MCM is nil"))
	} else {
		kubeconfig, kubevirtKubeconifgCheck := secret.Data["kubeconfig"]
		_, userdataCheck := secret.Data["userData"]

		if !kubevirtKubeconifgCheck {
			errs = append(errs, fmt.Errorf("secret kubeconfig is required field"))
		} else {
			_, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to decode kubeconfig: %v", err))
			}
		}
		if !userdataCheck {
			errs = append(errs, fmt.Errorf("secret userData is required field"))
		}
	}
	return errs
}
