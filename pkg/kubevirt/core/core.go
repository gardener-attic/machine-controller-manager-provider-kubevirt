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
	"time"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ProviderName specifies the machine controller for kubevirt cloud provider
	ProviderName = "kubevirt"
)

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

// ServerVersionFactory gets the server version from the kubeconfig saved in the "kubeconfig" field of the given secret.
type ServerVersionFactory interface {
	// GetServerVersion gets the server version from the kubeconfig saved in the "kubeconfig" field of the given secret.
	GetServerVersion(secret *corev1.Secret) (string, error)
}

// ServerVersionFactoryFunc is a function that implements ServerVersionFactory.
type ServerVersionFactoryFunc func(secret *corev1.Secret) (string, error)

// GetServerVersion gets the server version from the kubeconfig saved in the "kubeconfig" field of the given secret.
func (f ServerVersionFactoryFunc) GetServerVersion(secret *corev1.Secret) (string, error) {
	return f(secret)
}

// PluginSPIImpl is the real implementation of PluginSPI interface
// that makes the calls to the provider SDK
type PluginSPIImpl struct {
	cf  ClientFactory
	svf ServerVersionFactory
}

// NewPluginSPIImpl creates a new PluginSPIImpl with the given ClientFactory and ServerVersionFactory.
func NewPluginSPIImpl(cf ClientFactory, svf ServerVersionFactory) (*PluginSPIImpl, error) {
	return &PluginSPIImpl{
		cf:  cf,
		svf: svf,
	}, nil
}

// CreateMachine creates a Kubevirt virtual machine with the given name and an associated data volume based on the
// DataVolumeTemplate, using the given provider spec. It also creates a secret where the userdata(cloud-init) are saved and mounted on the VM.
func (p PluginSPIImpl) CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerID string, err error) {
	// Generate a unique name for the userdata secret
	userDataSecretName := fmt.Sprintf("userdata-%s-%s", machineName, strconv.Itoa(int(time.Now().Unix())))

	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	// Build interfaces and networks
	interfaces, networks, networkData := buildNetworks(providerSpec.Networks)

	// Build disks, volumes, and data volumes
	disks, volumes, dataVolumes := buildVolumes(machineName, namespace, userDataSecretName, networkData, providerSpec.RootVolume, providerSpec.AdditionalVolumes)

	// Get Kubernetes version
	k8sVersion, err := p.svf.GetServerVersion(secret)
	if err != nil {
		return "", fmt.Errorf("failed to get server version: %v", err)
	}

	// Build affinity
	affinity := buildAffinity(providerSpec.Region, providerSpec.Zone, k8sVersion)

	// Add SSH keys to user data
	userData, err := addUserSSHKeysToUserData(string(secret.Data["userData"]), providerSpec.SSHKeys)
	if err != nil {
		return "", fmt.Errorf("failed to add ssh keys to cloud-init: %v", err)
	}

	// Initialize VM labels
	vmLabels := make(map[string]string)
	if len(providerSpec.Tags) > 0 {
		vmLabels = providerSpec.Tags
	}
	vmLabels["kubevirt.io/vm"] = machineName

	// Build the VM
	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: namespace,
			Labels:    vmLabels,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Running: pointer.BoolPtr(true),
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/vm": machineName,
					},
				},
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Domain: kubevirtv1.DomainSpec{
						CPU:    providerSpec.CPU,
						Memory: providerSpec.Memory,
						Devices: kubevirtv1.Devices{
							Disks:      disks,
							Interfaces: interfaces,
						},
						Resources: providerSpec.Resources,
					},
					Affinity:                      affinity,
					TerminationGracePeriodSeconds: pointer.Int64Ptr(30),
					Volumes:                       volumes,
					Networks:                      networks,
					DNSPolicy:                     providerSpec.DNSPolicy,
					DNSConfig:                     providerSpec.DNSConfig,
				},
			},
			DataVolumeTemplates: dataVolumes,
		},
	}

	// Create the VM
	if err := c.Create(ctx, virtualMachine); err != nil {
		return "", fmt.Errorf("failed to create VirtualMachine: %v", err)
	}

	// Build the userdata secret
	userDataSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userDataSecretName,
			Namespace: virtualMachine.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(virtualMachine, kubevirtv1.VirtualMachineGroupVersionKind),
			},
		},
		Data: map[string][]byte{
			"userdata": []byte(userData),
		},
	}

	// Create the userdata secret
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
		if errors.IsMachineNotFoundError(err) {
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

	virtualMachine.Spec.Running = pointer.BoolPtr(false)
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
		if apierrors.IsNotFound(err) {
			return nil, &errors.MachineNotFoundError{
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
