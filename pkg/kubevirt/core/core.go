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

// Package core contains the cloud kubevirt specific implementations to manage machines
package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	clouderrors "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/util"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	utilpointer "k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProviderName specifies the machine controller for kubevirt cloud provider
const ProviderName = "kubevirt"

// ClientFactory creates a client from the kubeconfig saved in the "kubeconfig" field of the given secret.
type ClientFactory interface {
	// GetClient creates a client from the kubeconfig saved in the "kubeconfig" field of the given secret.
	// It also returns the namespace of the kubeconfig's current context.
	GetClient(secret *corev1.Secret) (client.Client, string, error)
}

// ClientFactoryFunc is a function that implements ClientFactory.
type ClientFactoryFunc func(secret *corev1.Secret) (client.Client, string, error)

// GetClient creates a client from the kubeconfig saved in the "kubeconfig" field of the given secret.
// It also returns the namespace of the kubeconfig's current context.
func (f ClientFactoryFunc) GetClient(secret *corev1.Secret) (client.Client, string, error) {
	return f(secret)
}

// PluginSPIImpl is the real implementation of PluginSPI interface
// that makes the calls to the provider SDK
type PluginSPIImpl struct {
	cf ClientFactory
}

// NewPluginSPIImpl creates a new PluginSPIImpl with the given ClientFactory.
func NewPluginSPIImpl(cf ClientFactory) (*PluginSPIImpl, error) {
	return &PluginSPIImpl{
		cf: cf,
	}, nil
}

// CreateMachine creates a Kubevirt virtual machine with the given name and an associated data volume based on the
// DataVolumeTemplate, using the given provider spec. It also creates a secret where the userdata(cloud-init) are saved and mounted on the VM.
func (p PluginSPIImpl) CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerID string, err error) {
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	requestsAndLimits, err := util.ParseResources(providerSpec.CPUs, providerSpec.Memory)
	if err != nil {
		return "", fmt.Errorf("failed to parse resources fields: %v", err)
	}

	pvcSize, err := resource.ParseQuantity(providerSpec.PVCSize)
	if err != nil {
		return "", fmt.Errorf("failed to parse pvcSize field: %v", err)
	}

	var (
		pvcRequest                    = corev1.ResourceList{corev1.ResourceStorage: pvcSize}
		terminationGracePeriodSeconds = int64(30)
		userdataSecretName            = fmt.Sprintf("userdata-%s-%s", machineName, strconv.Itoa(int(time.Now().Unix())))

		dnsPolicy corev1.DNSPolicy
		dnsConfig *corev1.PodDNSConfig
	)

	if providerSpec.DNSPolicy != "" {
		dnsPolicy, err = util.DNSPolicy(providerSpec.DNSPolicy)
		if err != nil {
			return "", fmt.Errorf("invalid DNS policy: %v", err)
		}
	}

	if providerSpec.DNSConfig != "" {
		if err := yaml.Unmarshal([]byte(providerSpec.DNSConfig), dnsConfig); err != nil {
			return "", fmt.Errorf(`failed to unmarshal "dnsConfig" field: %v`, err)
		}
	}

	interfaces, networks, networkData := buildNetworks(providerSpec.Networks)

	userData := string(secret.Data["userData"])
	if len(providerSpec.SSHKeys) > 0 {
		var userSSHKeys []string
		for _, sshKey := range providerSpec.SSHKeys {
			userSSHKeys = append(userSSHKeys, strings.TrimSpace(sshKey))
		}

		userData, err = addUserSSHKeysToUserData(userData, userSSHKeys)
		if err != nil {
			return "", fmt.Errorf("failed to add ssh keys to cloud-init: %v", err)
		}
	}

	var vmLabels = map[string]string{}
	if len(providerSpec.Tags) > 0 {
		vmLabels = providerSpec.Tags
	}
	vmLabels["kubevirt.io/vm"] = machineName

	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: namespace,
			Labels:    vmLabels,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Running: utilpointer.BoolPtr(true),
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/vm": machineName,
					},
				},
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						Devices: kubevirtv1.Devices{
							Disks: []kubevirtv1.Disk{
								{
									Name:       "datavolumedisk",
									DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}},
								},
								{
									Name:       "cloudinitdisk",
									DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}},
								},
							},
							Interfaces: interfaces,
						},
						Resources: kubevirtv1.ResourceRequirements{
							Requests: *requestsAndLimits,
							Limits:   *requestsAndLimits,
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Volumes: []kubevirtv1.Volume{
						{
							Name: "datavolumedisk",
							VolumeSource: kubevirtv1.VolumeSource{
								DataVolume: &kubevirtv1.DataVolumeSource{
									Name: machineName,
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
									UserDataSecretRef: &corev1.LocalObjectReference{
										Name: userdataSecretName,
									},
									NetworkData: networkData,
								},
							},
						},
					},
					DNSPolicy: dnsPolicy,
					DNSConfig: dnsConfig,
					Networks:  networks,
				},
			},
			DataVolumeTemplates: []cdi.DataVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: machineName,
					},
					Spec: cdi.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							StorageClassName: utilpointer.StringPtr(providerSpec.StorageClassName),
							AccessModes: []corev1.PersistentVolumeAccessMode{
								"ReadWriteOnce",
							},
							Resources: corev1.ResourceRequirements{
								Requests: pvcRequest,
							},
						},
						Source: cdi.DataVolumeSource{
							HTTP: &cdi.DataVolumeSourceHTTP{
								URL: providerSpec.SourceURL,
							},
						},
					},
				},
			},
		},
	}

	if err := c.Create(ctx, virtualMachine); err != nil {
		return "", fmt.Errorf("failed to create VirtualMachine: %v", err)
	}

	userDataSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            userdataSecretName,
			Namespace:       virtualMachine.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(virtualMachine, kubevirtv1.VirtualMachineGroupVersionKind)},
		},
		Data: map[string][]byte{"userdata": []byte(userData)},
	}

	if err := c.Create(ctx, userDataSecret); err != nil {
		return "", fmt.Errorf("failed to create secret for userdata: %v", err)
	}

	return encodeProviderID(machineName), nil
}

// DeleteMachine deletes the Kubevirt virtual machine with the given name.
func (p PluginSPIImpl) DeleteMachine(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		if clouderrors.IsMachineNotFoundError(err) {
			klog.V(2).Infof("skip VirtualMachine evicting, VirtualMachine instance %s is not found", machineName)
			return "", nil
		}
		return "", err
	}

	if err := client.IgnoreNotFound(c.Delete(ctx, virtualMachine)); err != nil {
		return "", fmt.Errorf("failed to delete VirtualMachine %v: %v", machineName, err)
	}
	return encodeProviderID(virtualMachine.Name), nil
}

// GetMachineStatus fetches the provider id of the Kubevirt virtual machine with the given name.
func (p PluginSPIImpl) GetMachineStatus(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		return "", err
	}

	return encodeProviderID(virtualMachine.Name), nil
}

// ListMachines lists the provider ids of all Kubevirt virtual machines.
func (p PluginSPIImpl) ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerIDList map[string]string, err error) {
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	var vmLabels = map[string]string{}
	if len(providerSpec.Tags) > 0 {
		vmLabels = providerSpec.Tags
	}

	virtualMachineList, err := p.listVMs(ctx, c, namespace, vmLabels)
	if err != nil {
		return nil, err
	}

	var providerIDs = make(map[string]string, len(virtualMachineList.Items))
	for _, virtualMachine := range virtualMachineList.Items {
		providerIDs[encodeProviderID(virtualMachine.Name)] = virtualMachine.Name
	}

	return providerIDs, nil
}

// ShutDownMachine shuts down the Kubevirt virtual machine with the given name by setting its spec.running field to false.
func (p PluginSPIImpl) ShutDownMachine(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		return "", err
	}

	virtualMachine.Spec.Running = utilpointer.BoolPtr(false)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return c.Update(ctx, virtualMachine)
	}); err != nil {
		return "", fmt.Errorf("failed to update VirtualMachine running state: %v", err)
	}

	return encodeProviderID(virtualMachine.Name), nil
}

func (p PluginSPIImpl) getVM(ctx context.Context, c client.Client, machineName, namespace string) (*kubevirtv1.VirtualMachine, error) {
	virtualMachine := &kubevirtv1.VirtualMachine{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: machineName}, virtualMachine); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, &clouderrors.MachineNotFoundError{
				Name: machineName,
			}
		}
		return nil, fmt.Errorf("failed to get VirtualMachine: %v", err)
	}
	return virtualMachine, nil
}

func (p PluginSPIImpl) listVMs(ctx context.Context, c client.Client, namespace string, vmLabels map[string]string) (*kubevirtv1.VirtualMachineList, error) {
	virtualMachineList := &kubevirtv1.VirtualMachineList{}
	opts := []client.ListOption{client.InNamespace(namespace)}
	if len(vmLabels) > 0 {
		opts = append(opts, client.MatchingLabels(vmLabels))
	}
	if err := c.List(ctx, virtualMachineList, opts...); err != nil {
		return nil, fmt.Errorf("failed to list VirtualMachines: %v", err)
	}
	return virtualMachineList, nil
}
