package rollback

import (
	"fmt"

	boot "os.update.tool/pkg/bootconfig"
	"os.update.tool/pkg/exec"
	"os.update.tool/pkg/logger"
)

var (
	getActiveUKIFunc   = boot.GetActiveUKI
	executeCommandFunc = exec.ExecuteCommand
)

// RollbackChange restores the previous UKI
// by setting it as the default.
func RollbackChange() error {
	var prevUKI string

	// Get the active UKI
	activeUKI, err := getActiveUKIFunc()
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
	output, err := executeCommandFunc("bootctl", "set-default", prevUKI)
	if err != nil {
		logger.LogError("Failed to restore previous OS. Output: %s", output)
		return err
	}

	logger.LogInfo("Restore boot successfully.")
	return nil
}
