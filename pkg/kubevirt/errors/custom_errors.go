package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrInstanceNotFound tells that the requested instance was not found on the cloud provider
	ErrInstanceNotFound = errors.New("instance not found")
)

func IsNotFound(err error) bool {
	return err == ErrInstanceNotFound
}

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
