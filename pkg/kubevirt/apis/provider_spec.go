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

package api

import (
	corev1 "k8s.io/api/core/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

// KubeVirtProviderSpec is the spec to be used while parsing the calls.
type KubeVirtProviderSpec struct {
	// Resources defines requests and limits resources of VMI
	Resources kubevirtv1.ResourceRequirements `json:"resources"`
	// RootVolume is the specification for the root volume of the VM.
	RootVolume cdicorev1alpha1.DataVolumeSpec `json:"rootVolume"`
	// AdditionalVolumes is an optional list of additional volumes attached to the VM.
	// +optional
	AdditionalVolumes []AdditionalVolumeSpec `json:"additionalVolumes,omitempty"`
	// Region is the name of the region for the VM.
	Region string `json:"region"`
	// Zone is the name of the zone for the VM.
	Zone string `json:"zone"`
	// DNSConfig is the DNS configuration of the VM pod.
	// The parameters specified here will be merged with the generated DNS configuration based on DNSPolicy.
	// +optional
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
	// DNSPolicy is the DNS policy for the VM pod.
	// Defaults to "ClusterFirst" and valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.
	// To have DNS options set along with hostNetwork, specify DNS policy as 'ClusterFirstWithHostNet'.
	// +optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// SSHKeys is an optional list of SSH public keys added to the VM (may already be included in UserData)
	// +optional
	SSHKeys []string `json:"sshKeys,omitempty"`
	// Networks is an optional list of networks for the VM. If any of the networks is specified as "default"
	// the pod network won't be added, otherwise it will be added as default.
	// +optional
	Networks []NetworkSpec `json:"networks,omitempty"`
	// Tags is an optional map of tags that is added to the VM as labels.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
	// CPU allows specifying the CPU topology of KubeVirt VM.
	// +optional
	CPU *kubevirtv1.CPU `json:"cpu,omitempty"`
	// Memory allows specifying the VirtualMachineInstance memory features like huge pages and guest memory settings.
	// Each feature might require appropriate FeatureGate enabled.
	// For hugepages take a look at:
	// k8s - https://kubernetes.io/docs/tasks/manage-hugepages/scheduling-hugepages/
	// okd - https://docs.okd.io/3.9/scaling_performance/managing_hugepages.html#huge-pages-prerequisites
	// +optional
	Memory *kubevirtv1.Memory `json:"memory,omitempty"`
}

// AdditionalVolumeSpec represents an additional volume attached to a VM.
// Only one of its members may be specified.
type AdditionalVolumeSpec struct {
	// DataVolume is an optional specification of an additional data volume.
	// +optional
	DataVolume *cdicorev1alpha1.DataVolumeSpec `json:"dataVolume,omitempty"`
	// VolumeSource is an optional reference to an additional volume source.
	// +optional
	VolumeSource *VolumeSource `json:"volumeSource,omitempty"`
}

// VolumeSource represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// ConfigMapSource represents a reference to a ConfigMap in the same namespace.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	// +optional
	ConfigMap *kubevirtv1.ConfigMapVolumeSource `json:"configMap,omitempty"`
	// SecretVolumeSource represents a reference to a secret data in the same namespace.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret/
	// +optional
	Secret *kubevirtv1.SecretVolumeSource `json:"secret,omitempty"`
}

// NetworkSpec contains information about a network.
type NetworkSpec struct {
	// Name is the name (in the format <name> or <namespace>/<name>) of the network.
	Name string `json:"name"`
	// Default is whether the network is the default or not.
	// +optional
	Default bool `json:"default,omitempty"`
}
