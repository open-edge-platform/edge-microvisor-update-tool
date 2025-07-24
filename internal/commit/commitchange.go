package commit

import (
	"fmt"
	"path/filepath"

	boot "os.update.tool/pkg/bootconfig"
	core "os.update.tool/pkg/core"
	"os.update.tool/pkg/exec"
	"os.update.tool/pkg/logger"
)

var (
	checkWriteDoneFunc = core.CheckWriteDone
	getActiveUKIFunc   = boot.GetActiveUKI
	executeCommandFunc = exec.ExecuteCommand
	resetWriteDoneFunc = core.ResetWriteDone
)

// CommitChange sets the next boot entry to the new OS
// by executing the bootctl set-default command.
func CommitChange() error {
	var curUKI string

	// Check if write is done
	if !checkWriteDoneFunc() {
		logger.LogError("Nothing to commit: write is not done")
		return fmt.Errorf("nothing to commit: write is not done")
	}

	// Get the active UKI
	activeUKI, err := getActiveUKIFunc()
	if err != nil {
		logger.LogError("Failed to get active UKI: %v", err)
		return fmt.Errorf("failed to get active uki: %w", err)
	}

	// Get current UKI
	curUKI = filepath.Base(activeUKI)

	// Log the next default UKI
	logger.LogInfo("Next Default: %s", curUKI)

	// Execute the bootctl set-default command
	output, err := executeCommandFunc("bootctl", "set-default", curUKI)
	if err != nil {
		logger.LogError("Failed to commit new OS. Output: %s", output)
		return err
	}

	// Remove the .bak file
	if !resetWriteDoneFunc() {
		logger.LogError("Failed to remove temp UKI file: %v", err)
		return fmt.Errorf("failed to remove temp uki file: %w", err)
	}

	logger.LogInfo("Set default boot successfully.")
	return nil
}
