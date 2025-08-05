package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	core "os.update.tool/pkg/core"
	exec "os.update.tool/pkg/exec"
)

// getActiveUKI retrieves the active UKI file based on
// the active partition and boot configuration.
func GetActiveUKI() (string, error) {
	activePartition, err := core.GetActivePartition()
	if err != nil {
		return "", fmt.Errorf("failed to get active partition: %w", err)
	}
	efiLocation := core.UKIPATH
	files, err := filepath.Glob(filepath.Join(efiLocation, "*.efi"))
	if err != nil {
		return "", fmt.Errorf("failed to list UKI files: %w", err)
	}

	for _, file := range files {
		var match bool
		match, err := isUKIMatch(activePartition, filepath.Base(file))

		if err != nil {
			return "", err
		}

		if match {
			return file, nil
		}
	}

	return "", errors.New("no matching UKI file found")
}

// isUKIMatch is a helper function that
// checks if the given partition matches the UKI file
// based on LUKS UUID, Filesystem UUID, or PARTUUID.
func isUKIMatch(partition, ukiFile string) (bool, error) {
	// Retrieve PARTUUID or boot UUID from bootctl list
	ukiUUID, err := getBootctlUUID(ukiFile)
	if err != nil {
		return false, fmt.Errorf("could not retrieve PARTUUID or boot UUID for UKI file %s: %w", ukiFile, err)
	}

	// Check LUKS UUID match
	luksUUID, err := getLUKSUUID(partition)
	if err == nil && luksUUID == ukiUUID {
		return true, nil
	}

	// Check Filesystem UUID match
	fsUUID, err := getFilesystemUUID(partition)
	if err == nil && fsUUID == ukiUUID {
		return true, nil
	}

	// Check PARTUUID match
	partUUID, err := getPARTUUID(partition)
	if err == nil && partUUID == ukiUUID {
		return true, nil
	}

	// If no match is found
	return false, nil
}

// extractValue is a helper function that
// extracts the value for a given key from a string.
func extractValue(line, key string) string {
	parts := strings.Split(line, key)
	if len(parts) > 1 {
		value := strings.Fields(parts[1])[0]
		if core.ValidateUUID(value) {
			return value
		}
	}
	return ""
}

// getBootctlUUID is a helper function that retrieves
// the PARTUUID or boot UUID for the given UKI file from bootctl list.
func getBootctlUUID(ukiFile string) (string, error) {
	result, err := exec.ExecuteCommand("bootctl", "list")
	if err != nil {
		return "", fmt.Errorf("failed to execute bootctl list: %w", err)
	}

	output := result
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, ukiFile) {
			// search for options
			for j := i; j < len(lines) && j < i+5; j++ {
				sline := lines[j]
				// Check for PARTUUID
				if strings.Contains(sline, "root=PARTUUID=") {
					return extractValue(sline, "root=PARTUUID="), nil
				}
				// Check for boot UUID
				if strings.Contains(sline, "boot_uuid=") {
					return extractValue(sline, "boot_uuid="), nil
				}
			}
		}
	}

	return "", errors.New("could not retrieve PARTUUID or boot UUID from bootctl")
}

// getLUKSUUID is a helper function to retrieves
// the LUKS UUID for the given partition.
func getLUKSUUID(partition string) (string, error) {
	// Handle LUKS UUID
	blockDevice, err := core.GetBlockName(partition)
	if err != nil {
		return "", fmt.Errorf("failed to get block device %w", err)
	}
	result, err := exec.ExecuteCommand("cryptsetup", "luksUUID", blockDevice)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve LUKS UUID for partition %s: %w", blockDevice, err)
	}
	return strings.TrimSpace(result), nil
}

// getFilesystemUUID is a helper function to retrieves
// the Filesystem UUID for the given partition.
func getFilesystemUUID(partition string) (string, error) {
	result, err := exec.ExecuteCommand("tune2fs", "-l", partition)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve Filesystem UUID for partition %s: %w", partition, err)
	}

	output := result
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Filesystem UUID") {
			return strings.Fields(line)[2], nil
		}
	}

	return "", errors.New("could not retrieve Filesystem UUID")
}

// getPARTUUID is a helper function to retrieves
// the PARTUUID for the given partition.
func getPARTUUID(partition string) (string, error) {
	result, err := exec.ExecuteCommand("blkid", "-s", "PARTUUID", "-o", "value", partition)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve PARTUUID for partition %s: %w", partition, err)
	}
	return strings.TrimSpace(result), nil
}
