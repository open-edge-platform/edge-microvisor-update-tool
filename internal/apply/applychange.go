package apply

import (
	"fmt"

	"os.abupdate.tool/pkg/logger"
	boot "os.abupdate.tool/pkg/bootconfig"
	"os.abupdate.tool/pkg/core"
	"os.abupdate.tool/pkg/exec"
)

func ApplyChange() error {
	var nextUKI string

	// Check if write is done
	if !core.CheckWriteDone() {
		logger.LogError("Nothing to apply: write is not done")
		return fmt.Errorf("nothing to apply: write is not done")
	}

	// Check if linux.efi exists
	if !core.CheckFirstUKIExists() {
		// Rename current UKI to linux.efi
		if err := core.RenameEFI(); err != nil {
			logger.LogError("Failed to create linux.efi: %v", err)
			return fmt.Errorf("failed to create linux.efi: %w", err)
		}
	}

	// Get the active UKI
	activeUKI, err := boot.GetActiveUKI()
	if err != nil {
		logger.LogError("Failed to get active UKI: %v", err)
		return fmt.Errorf("failed to get active uki: %w", err)
	}

	// Determine the next UKI based on the active UKI
	if len(activeUKI) >= 6 && activeUKI[len(activeUKI)-6:] == "-2.efi" {
		nextUKI = "linux.efi"
	} else {
		nextUKI = "linux-2.efi"
	}

	// Apply new UKI
	if core.ApplyBoot(nextUKI) != nil {
		logger.LogError("Failed to apply new OS: %v", err)
		return fmt.Errorf("failed to apply new OS: %w", err)
	}

	// Execute the bootctl command to set the default boot entry
	output, err := exec.ExecuteCommand("bootctl", "set-oneshot", nextUKI)
	if err != nil {
		logger.LogError("Failed to apply new OS. Output: %s", string(output))
		return err
	}

	logger.LogInfo("Apply new OS successfully.")
	return nil
}
