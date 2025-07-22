package apply

import (
	"fmt"
	"path/filepath"

	boot "os.update.tool/pkg/bootconfig"
	core "os.update.tool/pkg/core"
	"os.update.tool/pkg/exec"
	"os.update.tool/pkg/logger"
)

// CommitChange sets the next boot entry to the new OS
// by executing the bootctl set-default command.
func CommitChange() error {
	var curUKI string

	// Check if write is done
	if !core.CheckWriteDone() {
		logger.LogError("Nothing to commit: write is not done")
		return fmt.Errorf("nothing to commit: write is not done")
	}

	// Get the active UKI
	activeUKI, err := boot.GetActiveUKI()
	if err != nil {
		logger.LogError("Failed to get active UKI: %v", err)
		return fmt.Errorf("failed to get active uki: %w", err)
	}

	// Get current UKI
	curUKI = filepath.Base(activeUKI)

	// Log the next default UKI
	logger.LogInfo("Next Default: %s", curUKI)

	// Execute the bootctl set-default command
	output, err := exec.ExecuteCommand("bootctl", "set-default", curUKI)
	if err != nil {
		logger.LogError("Failed to commit new OS. Output: %s", string(output))
		return err
	}

	// Remove the .bak file
	if !core.ResetWriteDone() {
		logger.LogError("Failed to remove temp UKI file: %v", err)
		return fmt.Errorf("failed to remove temp uki file: %w", err)
	}

	logger.LogInfo("Set default boot successfully.")
	return nil
}
