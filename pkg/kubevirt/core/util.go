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
	"fmt"
	"strings"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
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
		return nil, "", errors.Wrap(err, "could not get REST config from client config")
	}
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, "", errors.Wrap(err, "could not create client from REST config")
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, "", errors.Wrap(err, "could not get namespace from client config")
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
		return "", errors.Wrap(err, "could not get REST config from client config")
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "could not create clientset from REST config")
	}
	versionInfo, err := cs.ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "could not get server version")
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
		return nil, errors.Wrap(err, "could not create client config from kubeconfig")
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

	var interfaces []kubevirtv1.Interface
	var networks []kubevirtv1.Network

	// Determine whether there is a default network
	hasDefault := false
	for _, networkSpec := range networkSpecs {
		if networkSpec.Default {
			hasDefault = true
			break
		}
	}

	// If no default network was specified, append an interface and a network for the pod network.
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
	}

	// Append interfaces and networks for all network specs
	for i, networkSpec := range networkSpecs {
		// Generate a unique name for this network
		name := fmt.Sprintf("net%d", i)

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

func buildVolumes(
	machineName, namespace, userDataSecretName, networkData string,
	rootVolume cdicorev1alpha1.DataVolumeSpec,
	additionalVolumes []api.AdditionalVolumeSpec,
	configuredDisks []kubevirtv1.Disk,
) ([]kubevirtv1.Disk, []kubevirtv1.Volume, []cdicorev1alpha1.DataVolume) {
	var disks []kubevirtv1.Disk
	var volumes []kubevirtv1.Volume
	var dataVolumes []cdicorev1alpha1.DataVolume

	// Append a disk, a volume, and a data volume for the root disk
	var rootDisk kubevirtv1.Disk
	if d := findDiskByName(api.RootDiskName, configuredDisks); d != nil {
		rootDisk = *d
	} else {
		rootDisk = buildDefaultDisk(api.RootDiskName)
	}

	disks = append(disks, rootDisk)
	volumes = append(volumes, kubevirtv1.Volume{
		Name: api.RootDiskName,
		VolumeSource: kubevirtv1.VolumeSource{
			DataVolume: &kubevirtv1.DataVolumeSource{
				Name: machineName,
			},
		},
	})
	dataVolumes = append(dataVolumes, cdicorev1alpha1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: namespace,
		},
		Spec: rootVolume,
	})

	// Append a disk and a volume for the cloud-init disk
	disks = append(disks, kubevirtv1.Disk{
		Name: "cloudinitdisk",
		DiskDevice: kubevirtv1.DiskDevice{
			Disk: &kubevirtv1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	volumes = append(volumes, kubevirtv1.Volume{
		Name: "cloudinitdisk",
		VolumeSource: kubevirtv1.VolumeSource{
			CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
				UserDataSecretRef: &corev1.LocalObjectReference{
					Name: userDataSecretName,
				},
				NetworkData: networkData,
			},
		},
	})

	// Append disks, volumes, and data volumes for all additional disks
	for i, volume := range additionalVolumes {
		// Generate a unique name for this disk
		diskName := fmt.Sprintf("disk%d", i)

		var disk kubevirtv1.Disk
		if d := findDiskByName(volume.Name, configuredDisks); d != nil {
			disk = *d
			disk.Name = diskName
		} else {
			disk = buildDefaultDisk(diskName)
		}
		disks = append(disks, disk)

		switch {
		case volume.DataVolume != nil:
			// Generate a unique name for this data volume
			dataVolumeName := fmt.Sprintf("%s-%d", machineName, i)

			// Append a volume and a data volume for this additional disk
			volumes = append(volumes, kubevirtv1.Volume{
				Name: diskName,
				VolumeSource: kubevirtv1.VolumeSource{
					DataVolume: &kubevirtv1.DataVolumeSource{
						Name: dataVolumeName,
					},
				},
			})
			dataVolumes = append(dataVolumes, cdicorev1alpha1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataVolumeName,
					Namespace: namespace,
				},
				Spec: *volume.DataVolume,
			})

		case volume.VolumeSource != nil:
			// Append a volume for this additional disk
			volumes = append(volumes, kubevirtv1.Volume{
				Name: diskName,
				VolumeSource: kubevirtv1.VolumeSource{
					PersistentVolumeClaim: volume.VolumeSource.PersistentVolumeClaim,
					ConfigMap:             volume.VolumeSource.ConfigMap,
					Secret:                volume.VolumeSource.Secret,
				},
			})
		}
	}

	return disks, volumes, dataVolumes
}

func findDiskByName(name string, disks []kubevirtv1.Disk) *kubevirtv1.Disk {
	for _, disk := range disks {
		if name == disk.Name {
			return &disk
		}
	}
	return nil
}

func buildDefaultDisk(name string) kubevirtv1.Disk {
	return kubevirtv1.Disk{
		Name: name,
		DiskDevice: kubevirtv1.DiskDevice{
			Disk: &kubevirtv1.DiskTarget{
				Bus: "virtio",
			},
		},
	}
}

const (
	// defaultRegion is the name of the default region.
	// VMs using this region are scheduled on nodes for which a region failure domain is not specified.
	defaultRegion = "default"
	// defaultZone is the name of the default zone.
	// VMs using this zone are scheduled on nodes for which a zone failure domain is not specified.
	defaultZone = "default"
)

func buildAffinity(region, zone, k8sVersion string) *corev1.Affinity {
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

		// If there is a zone, add match expression for the zone label
		if zone != "" {
			if zone != defaultZone {
				matchExpressions = append(matchExpressions, corev1.NodeSelectorRequirement{
					Key:      zoneLabel,
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{zone},
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
	if len(sshKeys) == 0 {
		return userData, nil
	}

	if strings.Contains(userData, "ssh_authorized_keys:") {
		return "", errors.New("userData already contains key `ssh_authorized_keys`")
	}

	var userDataBuilder strings.Builder
	userDataBuilder.WriteString(userData)
	userDataBuilder.WriteString("\nssh_authorized_keys:\n")
	for _, sshKey := range sshKeys {
		userDataBuilder.WriteString("- ")
		userDataBuilder.WriteString(strings.TrimSpace(sshKey))
		userDataBuilder.WriteString("\n")
	}

	return userDataBuilder.String(), nil
}
