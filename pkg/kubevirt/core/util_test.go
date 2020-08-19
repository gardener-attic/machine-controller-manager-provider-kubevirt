package core

import (
	"strings"
	"testing"
)

var (
	testCases = []struct {
		name             string
		userData         string
		sshKeys          []string
		expectedUserData string
		expectedError    bool
	}{
		{
			name:             "`ssh_authorized_keys` key already exists error",
			userData:         "#cloud-config\nchpasswd:\nexpire: false\npassword: pass\nuser: test\nssh_authorized_keys:\n- ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu",
			sshKeys:          []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3b"},
			expectedUserData: "",
			expectedError:    true,
		},
		{
			name:             "add user ssh key to userdata successfully",
			userData:         "#cloud-config\nchpasswd:\nexpire: false\npassword: pass\nuser: test",
			sshKeys:          []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3b"},
			expectedUserData: "#cloud-config\nchpasswd:\nexpire: false\npassword: pass\nuser: test\nssh_authorized_keys:\n- ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDdOIhYmzCK5DSVLu3b",
			expectedError:    false,
		},
	}
)

func TestAddUserSSHKeysToUserData(t *testing.T) {
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			u, err := addUserSSHKeysToUserData(testCase.userData, testCase.sshKeys)
			if testCase.expectedError && err == nil {
				t.Fatal("expected an error but got error: nil")
			}

			if err != nil && !testCase.expectedError {
				t.Fatalf("unexpected error was encoutred: %v", err)
			}

			if strings.TrimSpace(testCase.expectedUserData) != strings.TrimSpace(u) {
				t.Fatalf("expecting userdata: %v and got: %v", testCase.expectedUserData, u)
			}

		})
	}
}
