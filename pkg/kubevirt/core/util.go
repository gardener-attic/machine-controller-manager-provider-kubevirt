package core

import (
	"fmt"
)

func encodeProviderID(machineID string) string {
	if machineID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", ProviderName, machineID)
}
