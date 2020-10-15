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
	"time"

	api "github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/apis"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/core"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	corev1 "k8s.io/api/core/v1"
)

// PluginSPI is an interface for provider-specific machine operations.
type PluginSPI interface {
	// CreateMachine creates a machine with the given name, using the given provider spec and secret.
	CreateMachine(ctx context.Context, machineName string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerID string, err error)
	// DeleteMachine deletes the machine with the given name and provider id, using the given provider spec and secret.
	DeleteMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error)
	// GetMachineStatus returns the provider id of the machine with the given name and provider id, using the given provider spec and secret.
	GetMachineStatus(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error)
	// ListMachines lists all machines matching the given provider spec and secret.
	ListMachines(ctx context.Context, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (providerIDList map[string]string, err error)
	// ShutDownMachine shuts down the machine with the given name and provider id, using the given provider spec and secret.
	ShutDownMachine(ctx context.Context, machineName, providerID string, providerSpec *api.KubeVirtProviderSpec, secret *corev1.Secret) (foundProviderID string, err error)
}

// MachinePlugin implements cmi.MachineServer by delegating to a PluginSPI implementation.
type MachinePlugin struct {
	// SPI is an implementation of the PluginSPI interface.
	SPI PluginSPI
}

// NewKubevirtPlugin creates a new kubevirt driver.
func NewKubevirtPlugin() driver.Driver {
	return &MachinePlugin{
		SPI: core.NewPluginSPIImpl(core.ClientFactoryFunc(core.GetClient), core.ServerVersionFactoryFunc(core.GetServerVersion), core.TimerFunc(time.Now)),
	}
}
