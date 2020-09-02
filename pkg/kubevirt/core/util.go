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
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClient creates a client from the kubeconfig saved in the "kubeconfig" field of the given secret.
// It also returns the namespace of the kubeconfig's current context.
func GetClient(secret *corev1.Secret) (client.Client, string, error) {
	kubeconfig, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, "", errors.New("missing kubeconfig field in secret")
	}
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("could not create client config from kubeconfig: %v", err)
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("could not get REST config from client config: %v", err)
	}
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, "", fmt.Errorf("could not create client from REST config: %v", err)
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("could not get namespace from client config: %v", err)
	}
	return c, namespace, nil
}

func encodeProviderID(machineID string) string {
	if machineID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", ProviderName, machineID)
}

func addUserSSHKeysToUserData(userData string, sshKeys []string) (string, error) {
	var userDataBuilder strings.Builder
	if strings.Contains(userData, "ssh_authorized_keys:") {
		return "", errors.New("userdata already contains key `ssh_authorized_keys`")
	}

	userDataBuilder.WriteString(userData)
	userDataBuilder.WriteString("\nssh_authorized_keys:\n")
	for _, key := range sshKeys {
		userDataBuilder.WriteString("- ")
		userDataBuilder.WriteString(key)
		userDataBuilder.WriteString("\n")
	}

	return userDataBuilder.String(), nil
}
