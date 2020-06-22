/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes-csi/drivers project
https://github.com/kubernetes-csi/drivers/blob/release-1.0/pkg/nfs/plugin.go

Modifications Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
*/

package kubevirt

import (
	"context"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	api "github.com/moadqassem/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/moadqassem/machine-controller-manager-provider-kubevirt/pkg/kubevirt/core"

	corev1 "k8s.io/api/core/v1"
)

// PluginSPI provides an interface to deal with cloud provider session
// You can optionally enhance this interface to add interface methods here
// You can use it to mock cloud provider calls
type PluginSPI interface {
	CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerID string, err error)
	DeleteMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
	GetMachineStatus(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
	ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerIDList map[string]string, err error)
	ShutDownMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
}

// MachinePlugin implements the cmi.MachineServer
// It also implements the pluginSPI interface
type MachinePlugin struct {
	SPI PluginSPI
}

// NewKubevirtPlugin returns a new Kubevirt p
func NewKubevirtPlugin() driver.Driver {
	return &MachinePlugin{
		SPI: core.PluginSPIImpl{},
	}
}
