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

// KubeVirtProviderSpec is the spec to be used while parsing the calls.
type KubeVirtProviderSpec struct {
	// SourceURL is the http url of the source image which will be imported by the cdi.
	SourceURL string `json:"sourceURL,omitempty"`
	// StorageClassName specifies the name which cdi uses to in order to create claims.
	StorageClassName string `json:"storageClassName,omitempty"`
	// PVCSize specifies the size of the PersistentVolumeClaim that's created during the image import by the cdi.
	PVCSize string `json:"pvcSize,omitempty"`
	// CPUs number of cpus that the vm will use.
	CPUs string `json:"cpus,omitempty"`
	// Memory specifies how much memroy the vm will request.
	Memory string `json:"memory,omitempty"`
	// Namespace specifies the namespace where the vm should be created.
	Namespace string `json:"namespace,omitempty"`
	// DNSConfig Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS
	// configuration based on DNSPolicy.
	// +optional
	DNSConfig string `json:"dnsConfig,omitempty"`
	// DNSPolicy Set DNS policy for the pod. Defaults to "ClusterFirst" and valid values are 'ClusterFirstWithHostNet', 'ClusterFirst',
	//'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have
	// DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.
	// +optional
	DNSPolicy string `json:"dnsPolicy,omitempty"`
}
