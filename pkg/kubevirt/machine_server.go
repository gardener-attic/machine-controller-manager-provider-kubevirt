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

// Package kubevirt contains the cloud kubevirt specific implementations to manage machines
package kubevirt

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/util"
	"github.com/gardener/machine-controller-manager-provider-kubevirt/pkg/kubevirt/validation"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog"
)

// CreateMachine handles a machine creation request
// REQUIRED METHOD
//
// REQUEST PARAMETERS (driver.CreateMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be created
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.CreateMachineResponse)
// ProviderID            string                   Unique identification of the VM at the cloud kubevirt. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName              string                   Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
// LastKnownState        string                   (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanisms to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
//
func (p *MachinePlugin) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	// Log messages to track request
	klog.V(2).Infof("Machine creation request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine creation request has been processed for %q", req.Machine.Name)

	providerSpec, err := util.DecodeProviderSpecAndSecret(req.MachineClass)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Create machine %q failed on DecodeProviderSpecAndSecret", req.Machine.Name)
	}

	validationErrors := validation.ValidateKubevirtSecret(providerSpec, req.Secret)
	if validationErrors != nil {
		err = fmt.Errorf("error while validating ProviderSpec %v", validationErrors)
		return nil, status.Error(codes.Internal, err.Error())
	}

	providerID, err := p.SPI.CreateMachine(ctx, req.Machine.Name, providerSpec, req.Secret)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Create machine %q failed", req.Machine.Name)
	}

	response := &driver.CreateMachineResponse{
		ProviderID:     providerID,
		NodeName:       req.Machine.Name,
		LastKnownState: fmt.Sprintf("Created %s", providerID),
	}
	return response, nil
}

// DeleteMachine handles a machine deletion request
//
// REQUEST PARAMETERS (driver.DeleteMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be deleted
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.DeleteMachineResponse)
// LastKnownState        bytes(blob)              (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//
func (p *MachinePlugin) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	// Log messages to track delete request
	klog.V(2).Infof("Machine deletion request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine deletion request has been processed for %q", req.Machine.Name)

	providerSpec, err := util.DecodeProviderSpecAndSecret(req.MachineClass)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Create machine %q failed on DecodeProviderSpecAndSecret", req.Machine.Name)
	}

	providerID, err := p.SPI.DeleteMachine(ctx, req.Machine.Name, req.Machine.Spec.ProviderID, providerSpec, req.Secret)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Create machine %q failed", req.Machine.Name)
	}

	response := &driver.DeleteMachineResponse{
		LastKnownState: fmt.Sprintf("Deleted %s", providerID),
	}
	return response, nil
}

// GetMachineStatus handles a machine get status request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.GetMachineStatusRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM status needs to be returned
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.GetMachineStatueResponse)
// ProviderID            string                   Unique identification of the VM at the cloud kubevirt. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName             string                    Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
//
// The request should return a NOT_FOUND (5) status errors code if the machine is not existing
func (p *MachinePlugin) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("Get request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine get request has been processed successfully for %q", req.Machine.Name)

	providerSpec, err := util.DecodeProviderSpecAndSecret(req.MachineClass)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Create machine %q failed on DecodeProviderSpecAndSecret", req.Machine.Name)
	}

	providerID, err := p.SPI.GetMachineStatus(ctx, req.Machine.Name, req.Machine.Spec.ProviderID, providerSpec, req.Secret)
	if err != nil {
		return nil, util.PrepareErrorf(err, "Machine status %q failed", req.Machine.Name)
	}

	response := &driver.GetMachineStatusResponse{
		ProviderID: providerID,
		NodeName:   req.Machine.Name,
	}

	klog.V(2).Infof("Machine status: found VM %q for Machine: %q", response.ProviderID, req.Machine.Name)

	return response, nil
}

// ListMachines lists all the machines possibly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.ListMachinesRequest)
// MachineClass          *v1alpha1.MachineClass   MachineClass based on which VMs created have to be listed
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.ListMachinesResponse)
// MachineList           map<string,string>  A map containing the keys as the MachineID and value as the MachineName
//                                           for all machine's who where possibilly created by this ProviderSpec
//
func (p *MachinePlugin) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)

	providerSpec, err := util.DecodeProviderSpecAndSecret(req.MachineClass)
	if err != nil {
		return nil, util.PrepareErrorf(err, "List machines failed on DecodeProviderSpecAndSecret")
	}

	machineList, err := p.SPI.ListMachines(ctx, providerSpec, req.Secret)
	if err != nil {
		return nil, util.PrepareErrorf(err, "List machines failed")
	}

	klog.V(2).Infof("List machines request for kubevirt cluster, found %d machines", len(machineList))

	return &driver.ListMachinesResponse{
		MachineList: machineList,
	}, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom an kubevirt volume was found
//
// REQUEST PARAMETERS (driver.GetVolumeIDsRequest)
// PVSpecList            []*corev1.PersistentVolumeSpec       PVSpecsList is a list PV specs for whom volume-IDs are required.
//
// RESPONSE PARAMETERS (driver.GetVolumeIDsResponse)
// VolumeIDs             []string                             VolumeIDs is a repeated list of VolumeIDs.
//
func (p *MachinePlugin) GetVolumeIDs(ctx context.Context, req *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetVolumeIDs request has been recieved for %q", req.PVSpecs)
	defer klog.V(2).Infof("GetVolumeIDs request has been processed successfully for %q", req.PVSpecs)

	return &driver.GetVolumeIDsResponse{}, status.Error(codes.Unimplemented, "")
}
