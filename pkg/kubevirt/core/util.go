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
	"errors"
	"fmt"
	"strings"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	"github.com/Masterminds/semver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClient creates a client from the kubeconfig saved in the "kubeconfig" field of the given secret.
// It also returns the namespace of the kubeconfig's current context.
func GetClient(secret *corev1.Secret) (client.Client, string, error) {
	clientConfig, err := getClientConfig(secret)
	if err != nil {
		return nil, "", err
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("could not get REST config from client config: %v", err)
	}
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, "", fmt.Errorf("could not create client from REST config: %v", err)
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("could not get namespace from client config: %v", err)
	}
	return c, namespace, nil
}

// GetServerVersion gets the server version from the kubeconfig saved in the "kubeconfig" field of the given secret.
func GetServerVersion(secret *corev1.Secret) (string, error) {
	clientConfig, err := getClientConfig(secret)
	if err != nil {
		return "", err
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return "", fmt.Errorf("could not get REST config from client config: %v", err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("could not create clientset from REST config: %v", err)
	}
	versionInfo, err := cs.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("could not get server version: %v", err)
	}
	return versionInfo.GitVersion, nil
}

func getClientConfig(secret *corev1.Secret) (clientcmd.ClientConfig, error) {
	kubeconfig, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, errors.New("missing kubeconfig field in secret")
	}
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not create client config from kubeconfig: %v", err)
	}
	return clientConfig, nil
}

func encodeProviderID(machineName string) string {
	if machineName == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", ProviderName, machineName)
}

func buildNetworks(networkSpecs []api.NetworkSpec) ([]kubevirtv1.Interface, []kubevirtv1.Network, string) {
	// If no network specs, return empty lists
	if len(networkSpecs) == 0 {
		return nil, nil, ""
	}

	// Determine whether there is a default network
	hasDefault := false
	for _, networkSpec := range networkSpecs {
		if networkSpec.Default {
			hasDefault = true
			break
		}
	}

	// Initialize network counter
	count := 0

	// If no default network was specified, append an interface and a network for the pod network.
	var interfaces []kubevirtv1.Interface
	var networks []kubevirtv1.Network
	if !hasDefault {
		// Append an interface and a network for the pod network
		interfaces = append(interfaces, kubevirtv1.Interface{
			Name: "default",
			InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
				Bridge: &kubevirtv1.InterfaceBridge{},
			},
		})
		networks = append(networks, kubevirtv1.Network{
			Name: "default",
			NetworkSource: kubevirtv1.NetworkSource{
				Pod: &kubevirtv1.PodNetwork{},
			},
		})

		// Increment network counter
		count++
	}

	// Append interfaces and networks for all network specs
	for _, networkSpec := range networkSpecs {
		// Generate a unique name for this network
		name := fmt.Sprintf("net%d", count)

		// Append an interface and a network for this network spec
		interfaces = append(interfaces, kubevirtv1.Interface{
			Name: name,
			InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
				Bridge: &kubevirtv1.InterfaceBridge{},
			},
		})
		networks = append(networks, kubevirtv1.Network{
			Name: name,
			NetworkSource: kubevirtv1.NetworkSource{
				Multus: &kubevirtv1.MultusNetwork{
					NetworkName: networkSpec.Name,
					Default:     networkSpec.Default,
				},
			},
		})

		// Increment network counter
		count++
	}

	// Enable DHCP for all ethernet interfces in networkData
	networkData := `version: 2
ethernets:
  id0:
    match:
      name: "e*"
    dhcp4: true
`

	return interfaces, networks, networkData
}

const (
	// defaultRegion is the name of the default region.
	// VMs using this region are scheduled on nodes for which a region failure domain is not specified.
	defaultRegion = "default"
	// defaultZone is the name of the default zone.
	// VMs using this zone are scheduled on nodes for which a zone failure domain is not specified.
	defaultZone = "default"
)

func buildAffinity(region string, zones []string, k8sVersion string) *corev1.Affinity {
	var affinity *corev1.Affinity
	if region != "" {
		// Get region and zone labels
		regionLabel, zoneLabel := getRegionAndZoneLabels(k8sVersion)

		// Add match expression for the region label
		var matchExpressions []corev1.NodeSelectorRequirement
		if region != defaultRegion {
			matchExpressions = append(matchExpressions, corev1.NodeSelectorRequirement{
				Key:      regionLabel,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{region},
			})
		} else {
			matchExpressions = append(matchExpressions, corev1.NodeSelectorRequirement{
				Key:      regionLabel,
				Operator: corev1.NodeSelectorOpDoesNotExist,
			})
		}

		// If there are zones, add match expression for the zone label
		if len(zones) > 0 {
			if len(zones) > 1 || zones[0] != defaultZone {
				matchExpressions = append(matchExpressions, corev1.NodeSelectorRequirement{
					Key:      zoneLabel,
					Operator: corev1.NodeSelectorOpIn,
					Values:   zones,
				})
			} else {
				matchExpressions = append(matchExpressions, corev1.NodeSelectorRequirement{
					Key:      zoneLabel,
					Operator: corev1.NodeSelectorOpDoesNotExist,
				})
			}
		}

		// Build affinity with the match expressions
		affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: matchExpressions,
						},
					},
				},
			},
		}
	}
	return affinity
}

func getRegionAndZoneLabels(k8sVersion string) (string, string) {
	c, _ := semver.NewConstraint("< 1.17")
	if c.Check(semver.MustParse(normalizeVersion(k8sVersion))) {
		return corev1.LabelZoneRegion, corev1.LabelZoneFailureDomain
	}
	return "topology.kubernetes.io/region", "topology.kubernetes.io/zone"
}

func normalizeVersion(version string) string {
	v := strings.Replace(version, "v", "", -1)
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	return v
}

func addUserSSHKeysToUserData(userData string, sshKeys []string) (string, error) {
	var userDataBuilder strings.Builder
	if strings.Contains(userData, "ssh_authorized_keys:") {
		return "", errors.New("userData already contains key `ssh_authorized_keys`")
	}

	userDataBuilder.WriteString(userData)
	userDataBuilder.WriteString("\nssh_authorized_keys:\n")
	for _, key := range sshKeys {
		userDataBuilder.WriteString("- ")
		userDataBuilder.WriteString(key)
		userDataBuilder.WriteString("\n")
	}

	return userDataBuilder.String(), nil
}
