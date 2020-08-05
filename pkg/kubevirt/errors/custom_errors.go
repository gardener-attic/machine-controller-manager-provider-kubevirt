package errors

import (
	"fmt"
)

// MachineNotFoundError is used to indicate not found error in PluginSPI
type MachineNotFoundError struct {
	// Name is the machine name
	Name string
	// MachineID is the machine uuid
	MachineID string
}

func (e *MachineNotFoundError) Error() string {
	return fmt.Sprintf("machine name=%s, uuid=%s not found", e.Name, e.MachineID)
}

func IsMachineNotFoundError(err error) bool {
	switch err.(type) {
	case *MachineNotFoundError:
		return true
	default:
		return false
	}
}

// DataVolumeError is used to indicate not found error in Kubevirt DataVolumeManager
type DataVolumeError struct {
	// Name is the machine name
	Name string
	// Namespace is the datavolume namespace
	Namespace string
}

func (e *DataVolumeError) Error() string {
	return fmt.Sprintf("data volume name=%s in namespace=%s is not found", e.Name, e.Namespace)
}

func IsDataVolumeError(err error) bool {
	switch err.(type) {
	case *DataVolumeError:
		return true
	default:
		return false
	}
}
