package apply

import (
	"fmt"

	boot "os.update.tool/pkg/bootconfig"
	"os.update.tool/pkg/core"
	"os.update.tool/pkg/exec"
	"os.update.tool/pkg/logger"
)

var (
	checkWriteDoneFunc      = core.CheckWriteDone
	checkFirstUKIExistsFunc = core.CheckFirstUKIExists
	renameEFIFunc           = core.RenameEFI
	getActiveUKIFunc        = boot.GetActiveUKI
	applyBootFunc           = core.ApplyBoot
	executeCommandFunc      = exec.ExecuteCommand
)

func ApplyChange() error {
	var nextUKI string

	// Check if write is done
	if !checkWriteDoneFunc() {
		logger.LogError("Nothing to apply: write is not done")
		return fmt.Errorf("nothing to apply: write is not done")
	}

	// Check if linux.efi exists
	if !checkFirstUKIExistsFunc() {
		// Rename current UKI to linux.efi
		if err := renameEFIFunc(); err != nil {
			logger.LogError("Failed to create linux.efi: %v", err)
			return fmt.Errorf("failed to create linux.efi: %w", err)
		}
	}

	// Get the active UKI
	activeUKI, err := getActiveUKIFunc()
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
	if applyBootFunc(nextUKI) != nil {
		logger.LogError("Failed to apply new OS: %v", err)
		return fmt.Errorf("failed to apply new OS: %w", err)
	}

	// Execute the bootctl command to set the default boot entry
	output, err := executeCommandFunc("bootctl", "set-oneshot", nextUKI)
	if err != nil {
		logger.LogError("Failed to apply new OS. Output: %s", output)
		return err
	}

	logger.LogInfo("Apply new OS successfully.")
	return nil
}
