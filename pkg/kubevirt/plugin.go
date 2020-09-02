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

package kubevirt

import (
	"context"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/core"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// PluginSPI provides an interface to deal with cloud provider session
// You can optionally enhance this interface to add interface methods here
// You can use it to mock cloud provider calls
type PluginSPI interface {
	// CreateMachine handles a machine creation request
	CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerID string, err error)
	// DeleteMachine handles a machine deletion request
	DeleteMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
	// GetMachineStatus handles a machine get status request
	GetMachineStatus(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
	// ListMachines lists all the machines possibly created by a providerSpec
	ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (providerIDList map[string]string, err error)
	// ShutDownMachine shuts down a machine by name
	ShutDownMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secrets *corev1.Secret) (foundProviderID string, err error)
}

// MachinePlugin implements the cmi.MachineServer
// It also implements the pluginSPI interface
type MachinePlugin struct {
	// SPI provides an interface to deal with cloud provider session.
	SPI PluginSPI
}

// NewKubevirtPlugin returns a new Kubevirt cloud provider driver.
func NewKubevirtPlugin() driver.Driver {
	plugin, err := core.NewPluginSPIImpl(core.ClientFactoryFunc(core.GetClient))
	if err != nil {
		klog.Errorf("failed to create Kubevirt plugin")
		return nil
	}

	return &MachinePlugin{
		SPI: plugin,
	}
}
