///*
//Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//*/
//
//// Package validation - validation is used to validate cloud specific KubeVirtProviderSpec
package validation

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/clientcmd"
)

// ValidateProviderSpecNSecret validates kubevirt spec and secret to check if all fields are present and valid
func ValidateKubevirtSecret(spec *api.KubeVirtProviderSpec, secrets *corev1.Secret) []error {
	var validationErrors []error

	if spec.CPUs == "" {
		validationErrors = append(validationErrors, errors.New("cpus field cannot be empty"))
	}
	if spec.Memory == "" {
		validationErrors = append(validationErrors, errors.New("memory field cannot be empty"))
	}
	if _, err := util.ParseResources(spec.CPUs, spec.Memory); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("invalid cpus/memory values: %v", err))
	}
	if spec.SourceURL == "" {
		validationErrors = append(validationErrors, errors.New("sourceURL field cannot be empty"))
	}
	if spec.StorageClassName == "" {
		validationErrors = append(validationErrors, errors.New("storageClassName field cannot be empty"))
	}
	if spec.PVCSize == "" {
		validationErrors = append(validationErrors, errors.New("memory field cannot be empty"))
	}
	if _, err := resource.ParseQuantity(spec.PVCSize); err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("failed to parse value of pvcSize field: %v", err))
	}
	if spec.DNSPolicy != "" {
		dnsPolicy, err := util.DNSPolicy(spec.DNSPolicy)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("invalid dns policy: %v", err))
		}

		if dnsPolicy == corev1.DNSNone {
			if spec.DNSConfig != "" {
				dnsConfig := &corev1.PodDNSConfig{}
				if err := yaml.Unmarshal([]byte(spec.DNSConfig), &dnsConfig); err != nil {
					validationErrors = append(validationErrors, fmt.Errorf("failed to unmarshal dnsConfig field %v", err))
				}
				if len(dnsConfig.Nameservers) == 0 {
					validationErrors = append(validationErrors, errors.New("dns config must specify nameservers when dns policy is None"))
				}
			} else {
				validationErrors = append(validationErrors, errors.New("dns config must be specified when dns policy is None"))
			}
		}
	}

	validationErrors = append(validationErrors, validateSecrets(secrets)...)

	return validationErrors
}

func validateSecrets(secret *corev1.Secret) []error {
	var validationErrors []error

	if secret == nil {
		validationErrors = append(validationErrors, errors.New("secret object that has been passed by the MCM is nil"))
	} else {
		kubeconfig, kubevirtKubeconifgCheck := secret.Data["kubeconfig"]
		_, userdataCheck := secret.Data["userData"]

		if !kubevirtKubeconifgCheck {
			validationErrors = append(validationErrors, fmt.Errorf("secret kubeconfig is required field"))
		} else {
			_, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Errorf("failed to decode kubeconfig: %v", err))
			}
		}
		if !userdataCheck {
			validationErrors = append(validationErrors, fmt.Errorf("secret userData is required field"))
		}
	}
	return validationErrors
}
