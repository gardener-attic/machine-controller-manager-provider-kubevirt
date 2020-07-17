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
