package rollback

import (
	"fmt"

	boot "os.abupdate.tool/pkg/bootconfig"
	"os.abupdate.tool/pkg/exec"
	"os.abupdate.tool/pkg/logger"
)

// RollbackChange restores the previous UKI
// by setting it as the default.
func RollbackChange() error {
	var prevUKI string

	// Get the active UKI
	activeUKI, err := boot.GetActiveUKI()
	if err != nil {
		logger.LogError("Failed to get active UKI: %v", err)
		return fmt.Errorf("failed to get active uki: %w", err)
	}

	// Determine the previous UKI based on the active UKI
	if len(activeUKI) >= 6 && activeUKI[len(activeUKI)-6:] == "-2.efi" {
		prevUKI = "linux.efi"
	} else {
		prevUKI = "linux-2.efi"
	}

	// Log the active and previous UKI
	logger.LogInfo("Active UKI: %s", activeUKI)
	logger.LogInfo("Previous UKI: %s", prevUKI)

	// Execute the bootctl command to set the default boot entry
	output, err := exec.ExecuteCommand("bootctl", "set-default", prevUKI)
	if err != nil {
		logger.LogError("Failed to restore previous OS. Output: %s", string(output))
		return err
	}

	logger.LogInfo("Restore boot successfully.")
	return nil
}
