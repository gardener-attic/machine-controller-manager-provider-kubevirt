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

package core_test

import (
	"context"
	"strconv"
	"time"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	. "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/core"
	mockclient "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/mock/client"
	mockcore "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/mock/kubevirt/core"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace         = "default"
	serverVersion     = "1.18"
	machineName       = "machine-1"
	clusterName       = "shoot--dev--kubevirt"
	machineClassName  = "machine-class-1"
	region            = "local"
	zone              = "local-1"
	storageClassName  = "standard"
	imageSourceURL    = "https://cloud-images.ubuntu.com/bionic/current/bionic-server-cloudimg-amd64.img"
	networkName       = "default/net-conf"
	sshPublicKey      = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3b"
	machineProviderID = ProviderName + "://" + machineName
)

var _ = Describe("PluginSPIImpl", func() {
	var (
		ctrl *gomock.Controller

		c     *mockclient.MockClient
		cf    *mockcore.MockClientFactory
		svf   *mockcore.MockServerVersionFactory
		timer *mockcore.MockTimer

		spi *PluginSPIImpl

		t                  = time.Now()
		userDataSecretName = "userdata-" + machineName + "-" + strconv.Itoa(int(t.Unix()))

		tags = map[string]string{
			"mcm.gardener.cloud/cluster":      clusterName,
			"mcm.gardener.cloud/role":         "node",
			"mcm.gardener.cloud/machineclass": machineClassName,
		}

		providerSpec = &api.KubeVirtProviderSpec{
			Region: region,
			Zone:   zone,
			Resources: kubevirtv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4096Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
			RootVolume: cdicorev1alpha1.DataVolumeSpec{
				PVC: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("8Gi"),
						},
					},
					StorageClassName: pointer.StringPtr(storageClassName),
				},
				Source: cdicorev1alpha1.DataVolumeSource{
					HTTP: &cdicorev1alpha1.DataVolumeSourceHTTP{
						URL: imageSourceURL,
					},
				},
			},
			AdditionalVolumes: []api.AdditionalVolumeSpec{
				{
					DataVolume: &cdicorev1alpha1.DataVolumeSpec{
						PVC: &corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								"ReadWriteOnce",
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
							StorageClassName: pointer.StringPtr(storageClassName),
						},
						Source: cdicorev1alpha1.DataVolumeSource{
							Blank: &cdicorev1alpha1.DataVolumeBlankImage{},
						},
					},
				},
			},
			SSHKeys: []string{
				sshPublicKey,
			},
			Networks: []api.NetworkSpec{
				{
					Name: networkName,
				},
			},
			CPU: &kubevirtv1.CPU{
				Cores:                 uint32(1),
				Sockets:               uint32(2),
				Threads:               uint32(1),
				DedicatedCPUPlacement: true,
			},
			Memory: &kubevirtv1.Memory{
				Hugepages: &kubevirtv1.Hugepages{
					PageSize: "2Mi",
				},
			},
			DNSPolicy: corev1.DNSDefault,
			DNSConfig: &corev1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8"},
			},
			Tags: tags,
		}

		secret = &corev1.Secret{
			Data: map[string][]byte{
				"userData": []byte("#cloud-config\nchpasswd:\nexpire: false\npassword: pass\nuser: test"),
			},
		}

		virtualMachine = &kubevirtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      machineName,
				Namespace: namespace,
				Labels: map[string]string{
					"mcm.gardener.cloud/cluster":      clusterName,
					"mcm.gardener.cloud/role":         "node",
					"mcm.gardener.cloud/machineclass": machineClassName,
					"kubevirt.io/vm":                  machineName,
				},
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
							Resources: providerSpec.Resources,
							CPU:       providerSpec.CPU,
							Memory:    providerSpec.Memory,
							Devices: kubevirtv1.Devices{
								Disks: []kubevirtv1.Disk{
									{
										Name: "rootdisk",
										DiskDevice: kubevirtv1.DiskDevice{
											Disk: &kubevirtv1.DiskTarget{
												Bus: "virtio",
											},
										},
									},
									{
										Name: "cloudinitdisk",
										DiskDevice: kubevirtv1.DiskDevice{
											Disk: &kubevirtv1.DiskTarget{
												Bus: "virtio",
											},
										},
									},
									{
										Name: "disk0",
										DiskDevice: kubevirtv1.DiskDevice{
											Disk: &kubevirtv1.DiskTarget{
												Bus: "virtio",
											},
										},
									},
								},
								Interfaces: []kubevirtv1.Interface{
									{
										Name: "default",
										InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
											Bridge: &kubevirtv1.InterfaceBridge{},
										},
									},
									{
										Name: "net0",
										InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
											Bridge: &kubevirtv1.InterfaceBridge{},
										},
									},
								},
							},
						},
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "topology.kubernetes.io/region",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{region},
												},
												{
													Key:      "topology.kubernetes.io/zone",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{zone},
												},
											},
										},
									},
								},
							},
						},
						TerminationGracePeriodSeconds: pointer.Int64Ptr(30),
						Volumes: []kubevirtv1.Volume{
							{
								Name: "rootdisk",
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
											Name: userDataSecretName,
										},
										NetworkData: `version: 2
ethernets:
  id0:
    match:
      name: "e*"
    dhcp4: true
`,
									},
								},
							},
							{
								Name: "disk0",
								VolumeSource: kubevirtv1.VolumeSource{
									DataVolume: &kubevirtv1.DataVolumeSource{
										Name: machineName + "-0",
									},
								},
							},
						},
						Networks: []kubevirtv1.Network{
							{
								Name: "default",
								NetworkSource: kubevirtv1.NetworkSource{
									Pod: &kubevirtv1.PodNetwork{},
								},
							},
							{
								Name: "net0",
								NetworkSource: kubevirtv1.NetworkSource{
									Multus: &kubevirtv1.MultusNetwork{
										NetworkName: networkName,
									},
								},
							},
						},
						DNSPolicy: providerSpec.DNSPolicy,
						DNSConfig: providerSpec.DNSConfig,
					},
				},
				DataVolumeTemplates: []cdicorev1alpha1.DataVolume{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      machineName,
							Namespace: namespace,
						},
						Spec: providerSpec.RootVolume,
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      machineName + "-0",
							Namespace: namespace,
						},
						Spec: *providerSpec.AdditionalVolumes[0].DataVolume,
					},
				},
			},
		}

		userDataSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userDataSecretName,
				Namespace: namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(virtualMachine, kubevirtv1.VirtualMachineGroupVersionKind),
				},
			},
			Data: map[string][]byte{
				"userdata": []byte("#cloud-config\nchpasswd:\nexpire: false\npassword: pass\nuser: test\nssh_authorized_keys:\n- " + sshPublicKey + "\n"),
			},
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		c = mockclient.NewMockClient(ctrl)
		cf = mockcore.NewMockClientFactory(ctrl)
		svf = mockcore.NewMockServerVersionFactory(ctrl)
		timer = mockcore.NewMockTimer(ctrl)

		cf.EXPECT().GetClient(gomock.AssignableToTypeOf(&corev1.Secret{})).Return(c, namespace, nil)

		spi = NewPluginSPIImpl(cf, svf, timer)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#CreateMachine", func() {
		It("should create the kubevirt virtual machine and the userdata secret", func() {
			svf.EXPECT().GetServerVersion(gomock.AssignableToTypeOf(&corev1.Secret{})).Return(serverVersion, nil)
			timer.EXPECT().Now().Return(t)

			c.EXPECT().Create(context.TODO(), virtualMachine).Return(nil)
			c.EXPECT().Create(context.TODO(), userDataSecret).Return(nil)

			providerID, err := spi.CreateMachine(context.TODO(), machineName, providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerID).To(Equal(machineProviderID))
		})
	})

	Describe("#DeleteMachine", func() {
		It("should delete the kubevirt virtual machine if it exists", func() {
			expectGetVirtualMachine(c, virtualMachine, nil)
			c.EXPECT().Delete(context.TODO(), virtualMachine).Return(nil)

			providerID, err := spi.DeleteMachine(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerID).To(Equal(machineProviderID))
		})

		It("should not fail if the kubevirt virtual machine does not exist", func() {
			expectGetVirtualMachine(c, nil, apierrors.NewNotFound(schema.GroupResource{}, ""))

			providerID, err := spi.DeleteMachine(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerID).To(BeEmpty())
		})
	})

	Describe("#GetMachineStatus", func() {
		It("should return the provider id of the kubevirt virtual machine if it exists", func() {
			expectGetVirtualMachine(c, virtualMachine, nil)

			providerID, err := spi.GetMachineStatus(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerID).To(Equal(machineProviderID))
		})

		It("should return a MachineNotFoundError if the kubevirt virtual machine does not exist", func() {
			expectGetVirtualMachine(c, nil, apierrors.NewNotFound(schema.GroupResource{}, ""))

			providerID, err := spi.GetMachineStatus(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).To(Equal(&MachineNotFoundError{Name: machineName}))
			Expect(providerID).To(BeEmpty())
		})
	})

	Describe("#ListMachines", func() {
		It("should list the provider ids of all kubevirt virtual machines matching the provider spec", func() {
			expectListVirtualMachines(c, virtualMachine, tags)

			providerIDs, err := spi.ListMachines(context.TODO(), providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerIDs).To(Equal(map[string]string{
				machineProviderID: machineName,
			}))
		})

		It("should return an empty map if no kubevirt virtual machines matching the provider spec exist", func() {
			expectListVirtualMachines(c, nil, tags)

			providerIDs, err := spi.ListMachines(context.TODO(), providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerIDs).To(BeEmpty())
		})
	})

	Describe("#ShutDownMachine", func() {
		It("should set the spec.running field of the kubevirt virtual machine to false", func() {
			expectGetVirtualMachine(c, virtualMachine, nil)
			c.EXPECT().Update(context.TODO(), withRunning(virtualMachine, false)).Return(nil)

			providerID, err := spi.ShutDownMachine(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerID).To(Equal(machineProviderID))
		})

		It("should return a MachineNotFoundError if the kubevirt virtual machine does not exist", func() {
			expectGetVirtualMachine(c, nil, apierrors.NewNotFound(schema.GroupResource{}, ""))

			providerID, err := spi.ShutDownMachine(context.TODO(), machineName, machineProviderID, providerSpec, secret)
			Expect(err).To(Equal(&MachineNotFoundError{Name: machineName}))
			Expect(providerID).To(BeEmpty())
		})
	})
})

func expectGetVirtualMachine(c *mockclient.MockClient, virtualMachine *kubevirtv1.VirtualMachine, err error) {
	c.EXPECT().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: machineName}, &kubevirtv1.VirtualMachine{}).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, vm *kubevirtv1.VirtualMachine) error {
			if err != nil {
				return err
			}
			*vm = *virtualMachine.DeepCopy()
			return nil
		})
}

func expectListVirtualMachines(c *mockclient.MockClient, virtualMachine *kubevirtv1.VirtualMachine, labels map[string]string) {
	c.EXPECT().List(context.TODO(), &kubevirtv1.VirtualMachineList{}, client.InNamespace(namespace), client.MatchingLabels(labels)).
		DoAndReturn(func(_ context.Context, vmList *kubevirtv1.VirtualMachineList, _ ...client.ListOption) error {
			if virtualMachine != nil {
				vmList.Items = []kubevirtv1.VirtualMachine{*virtualMachine.DeepCopy()}
			} else {
				vmList.Items = []kubevirtv1.VirtualMachine{}
			}
			return nil
		})
}

func withRunning(virtualMachine *kubevirtv1.VirtualMachine, running bool) *kubevirtv1.VirtualMachine {
	vm := virtualMachine.DeepCopy()
	vm.Spec.Running = pointer.BoolPtr(false)
	return vm
}
