/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package kubevirt contains the cloud kubevirt specific implementations to manage machines
package core

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	clouderrors "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/errors"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/util"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ProviderName = "kubevirt"

// PluginSPIImpl is the real implementation of PluginSPI interface
// that makes the calls to the provider SDK
type PluginSPIImpl struct {
	client client.Client
}

func (p PluginSPIImpl) CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerID string, err error) {
	requestsAndLimits, err := util.ParseResources(providerSpec.CPUs, providerSpec.Memory)
	if err != nil {
		return "", err
	}

	pvcSize, err := resource.ParseQuantity(providerSpec.PVCSize)
	if err != nil {
		return "", fmt.Errorf("failed to parse value of pvcSize field: %v", err)
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
			return "", fmt.Errorf("invalid dns policy: %v", err)
		}
	}

	if providerSpec.DNSConfig != "" {
		if err := yaml.Unmarshal([]byte(providerSpec.DNSConfig), dnsConfig); err != nil {
			return "", fmt.Errorf(`failed to unmarshal "dnsConfig" field: %v`, err)
		}
	}

	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: providerSpec.Namespace,
			Labels: map[string]string{
				"kubevirt.io/vm": machineName,
			},
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
								},
							},
						},
					},
					DNSPolicy: dnsPolicy,
					DNSConfig: dnsConfig,
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

	if p.client == nil {
		kubevirtClient, err := kubevirtClient(secrets)
		if err != nil {
			return "", fmt.Errorf("failed to get kubevirt client: %v", err)
		}
		p.client = kubevirtClient
	}

	if err := p.client.Create(ctx, virtualMachine); err != nil {
		return "", fmt.Errorf("failed to create vmi: %v", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            userdataSecretName,
			Namespace:       virtualMachine.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(virtualMachine, kubevirtv1.VirtualMachineGroupVersionKind)},
		},
		Data: map[string][]byte{"userdata": []byte(secrets.Data["userData"])},
	}

	if err := p.client.Create(ctx, secret); err != nil {
		return "", fmt.Errorf("failed to create secret for userdata: %v", err)
	}

	// TODO(MQ): do we really need this? another approach is to return an empty provider id and leave it to the
	// Get
	return p.machineProviderID(secrets, machineName, providerSpec.Namespace)
}

func (p PluginSPIImpl) DeleteMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error) {
	vm, err := p.getVM(secrets, machineName, providerSpec.Namespace)
	if err != nil {
		if clouderrors.IsMachineNotFoundError(err) {
			klog.V(2).Infof("skip vm evicting, vm instance %s is not found", machineName)
			return "", nil
		}
		return "", err
	}

	if vm != nil {
		if p.client == nil {
			p.client, err = kubevirtClient(secrets)
			if err != nil {
				return "", fmt.Errorf("failed to create kubevirt kubevirtClient: %v", err)
			}
		}

		if err := p.client.Delete(context.Background(), vm); err != nil {
			return "", fmt.Errorf("failed to delete vm %v: %v", machineName, err)
		}
		return encodeProviderID(string(vm.UID)), nil
	}

	return "", nil
}

func (p PluginSPIImpl) GetMachineStatus(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error) {
	return p.machineProviderID(secrets, machineName, providerSpec.Namespace)
}

func (p PluginSPIImpl) ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerIDList map[string]string, err error) {
	return p.listVMs(ctx, secrets)
}

func (p PluginSPIImpl) ShutDownMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error) {
	virtualMachine, err := p.getVM(secrets, machineName, providerSpec.Namespace)
	if err != nil {
		return "", err
	}
	if virtualMachine != nil {
		virtualMachine.Spec.Running = utilpointer.BoolPtr(false)
		if err := p.client.Update(ctx, virtualMachine); err != nil {
			return "", fmt.Errorf("failed to update machine running state: %v", err)
		}
		return encodeProviderID(string(virtualMachine.UID)), nil
	}

	return "", nil
}

func (p PluginSPIImpl) getVM(secret *corev1.Secret, machineName, namespace string) (*kubevirtv1.VirtualMachine, error) {
	var err error
	if p.client == nil {
		p.client, err = kubevirtClient(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubevirt kubevirtClient: %v", err)
		}
	}

	virtualMachine := &kubevirtv1.VirtualMachine{}
	if err := p.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: machineName}, virtualMachine); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, &clouderrors.MachineNotFoundError{
				Name: machineName,
			}
		}
		return nil, fmt.Errorf("failed to find kubevirt vm: %v", err)
	}

	return virtualMachine, nil
}

func (p PluginSPIImpl) listVMs(ctx context.Context, secret *corev1.Secret) (map[string]string, error) {
	var err error
	if p.client == nil {
		p.client, err = kubevirtClient(secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubevirt kubevirtClient: %v", err)
		}
	}

	virtualMachineList := &kubevirtv1.VirtualMachineList{}
	if err := p.client.List(ctx, virtualMachineList, &client.ListOptions{}); err != nil {
		return nil, fmt.Errorf("failed to list kubevirt virtual machines: %v", err)
	}

	var providerIDs = make(map[string]string, len(virtualMachineList.Items))
	for _, vm := range virtualMachineList.Items {
		providerID := encodeProviderID(string(vm.UID))
		providerIDs[providerID] = vm.Name
	}

	return providerIDs, nil
}

func kubevirtClient(secret *corev1.Secret) (client.Client, error) {
	kubeconfig, kubevirtKubeconifgCheck := secret.Data["kubeconfig"]
	if !kubevirtKubeconifgCheck {
		return nil, fmt.Errorf("kubevirt kubeconfig is not found")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decode kubeconfig: %v", err)
	}

	return client.New(config, client.Options{})
}

func (p PluginSPIImpl) machineProviderID(secret *corev1.Secret, vmName, namespace string) (string, error) {
	vm, err := p.getVM(secret, vmName, namespace)
	if err != nil || vm == nil {
		return "", err
	}

	return encodeProviderID(string(vm.UID)), nil
}
