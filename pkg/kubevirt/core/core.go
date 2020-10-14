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
	"context"
	"fmt"
	"strconv"
	"time"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"

	"github.com/pkg/errors"
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
	// ProviderName is the kubevirt provider name.
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

// PluginSPIImpl is the implementation of PluginSPI interface.
type PluginSPIImpl struct {
	cf  ClientFactory
	svf ServerVersionFactory
}

// NewPluginSPIImpl creates a new PluginSPIImpl with the given ClientFactory and ServerVersionFactory.
func NewPluginSPIImpl(cf ClientFactory, svf ServerVersionFactory) *PluginSPIImpl {
	return &PluginSPIImpl{
		cf:  cf,
		svf: svf,
	}
}

// CreateMachine creates a machine with the given name, using the given provider spec and secret.
// Here it creates a kubevirt virtual machine and a secret containing the userdata (cloud-init).
func (p PluginSPIImpl) CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerID string, err error) {
	// Generate a unique name for the userdata secret
	userDataSecretName := fmt.Sprintf("userdata-%s-%s", machineName, strconv.Itoa(int(time.Now().Unix())))

	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", errors.Wrap(err, "could not create client")
	}

	// Build interfaces and networks
	interfaces, networks, networkData := buildNetworks(providerSpec.Networks)

	// Build disks, volumes, and data volumes
	disks, volumes, dataVolumes := buildVolumes(machineName, namespace, userDataSecretName, networkData, providerSpec.RootVolume, providerSpec.AdditionalVolumes)

	// Get Kubernetes version
	k8sVersion, err := p.svf.GetServerVersion(secret)
	if err != nil {
		return "", errors.Wrap(err, "could not get server version")
	}

	// Build affinity
	affinity := buildAffinity(providerSpec.Region, providerSpec.Zone, k8sVersion)

	// Add SSH keys to user data
	userData, err := addUserSSHKeysToUserData(string(secret.Data["userData"]), providerSpec.SSHKeys)
	if err != nil {
		return "", err
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
		return "", errors.Wrapf(err, "could not create VirtualMachine %q", machineName)
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
		return "", errors.Wrapf(err, "could not create userdata secret %q", userDataSecretName)
	}

	// Return the VM provider ID
	return encodeProviderID(machineName), nil
}

// DeleteMachine deletes the machine with the given name and provider id, using the given provider spec and secret.
// Here it deletes the kubevirt virtual machine with the given name.
func (p PluginSPIImpl) DeleteMachine(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", errors.Wrap(err, "could not create client")
	}

	// Get the VM by name
	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		if IsMachineNotFoundError(err) {
			klog.V(2).Infof("VirtualMachine %s not found", machineName)
			return "", nil
		}
		return "", err
	}

	// Delete the VM
	if err := client.IgnoreNotFound(c.Delete(ctx, virtualMachine)); err != nil {
		return "", errors.Wrapf(err, "could not delete VirtualMachine %q", machineName)
	}

	// Return the VM provider ID
	return encodeProviderID(virtualMachine.Name), nil
}

// GetMachineStatus returns the provider id of the machine with the given name and provider id, using the given provider spec and secret.
// Here it returns the provider id of the kubevirt virtual machine with the given name.
func (p PluginSPIImpl) GetMachineStatus(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", errors.Wrap(err, "could not create client")
	}

	// Get the VM by name
	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		return "", err
	}

	// Return the VM provider ID
	return encodeProviderID(virtualMachine.Name), nil
}

// ListMachines lists all machines matching the given provider spec and secret.
// Here it lists all kubevirt virtual machines matching the tags of the given provider spec.
func (p PluginSPIImpl) ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerIDList map[string]string, err error) {
	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return nil, errors.Wrap(err, "could not create client")
	}

	// Initialize VM labels
	var vmLabels = map[string]string{}
	if len(providerSpec.Tags) > 0 {
		vmLabels = providerSpec.Tags
	}

	// List all VMs matching the labels
	virtualMachineList, err := p.listVMs(ctx, c, namespace, vmLabels)
	if err != nil {
		return nil, err
	}

	// Return a map containing the provider IDs and names of all found VMs
	var providerIDs = make(map[string]string, len(virtualMachineList.Items))
	for _, virtualMachine := range virtualMachineList.Items {
		providerIDs[encodeProviderID(virtualMachine.Name)] = virtualMachine.Name
	}
	return providerIDs, nil
}

// ShutDownMachine shuts down the machine with the given name and provider id, using the given provider spec and secret.
// Here it shuts down the kubevirt virtual machine with the given name by setting its spec.running field to false.
func (p PluginSPIImpl) ShutDownMachine(ctx context.Context, machineName, _ string, _ *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error) {
	// Get client and namespace from secret
	c, namespace, err := p.cf.GetClient(secret)
	if err != nil {
		return "", errors.Wrap(err, "could not create client")
	}

	// Get the VM by name
	virtualMachine, err := p.getVM(ctx, c, machineName, namespace)
	if err != nil {
		return "", err
	}

	// Set the VM spec.running field to false
	virtualMachine.Spec.Running = pointer.BoolPtr(false)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return c.Update(ctx, virtualMachine)
	}); err != nil {
		return "", errors.Wrapf(err, "could not update VirtualMachine %q", machineName)
	}

	// Return the VM provider ID
	return encodeProviderID(virtualMachine.Name), nil
}

func (p PluginSPIImpl) getVM(ctx context.Context, c client.Client, machineName, namespace string) (*kubevirtv1.VirtualMachine, error) {
	virtualMachine := &kubevirtv1.VirtualMachine{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: machineName}, virtualMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, &MachineNotFoundError{
				Name: machineName,
			}
		}
		return nil, errors.Wrapf(err, "could not get VirtualMachine %q", machineName)
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
		return nil, errors.Wrapf(err, "could not list VirtualMachines in namespace %q", namespace)
	}
	return virtualMachineList, nil
}
