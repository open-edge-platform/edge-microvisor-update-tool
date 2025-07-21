package core

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	exec "os.update.tool/pkg/exec"
	"os.update.tool/pkg/logger"
)

const (
	INDEXPARTA = 2 // allowed index for part A
	INDEXPARTB = 6 // allowed index for part B
	UKIPATH    = "/boot/efi/EFI/Linux"
)

var UKIUpdatePath = filepath.Join(UKIPATH, "os-update-tool-data")
var UKIUpdate = filepath.Join(UKIUpdatePath, "linux.bak")

// ExecuteCommand runs a Linux command passed as a string and returns the output and error if any.
func GetTargetPartition() (string, error) {

	// Check if the system is using FDE (Full Disk Encryption) enabled image
	tgtPrt, err := getFdeTargetPartition()
	if tgtPrt != "" && err == nil {
		logger.LogInfo("FDE enabled and target partition found")
		return tgtPrt, nil
	} else {
		logger.LogError(fmt.Sprintf("Error: %v, FDE not enabled", err))
	}
	// Check if the system is using a default image with dm-verity enabled

	// Check if the system is using a default OR dm-verity enabled only image but NO FDE enabled
	tgtPrt, err = getDefaultTargetPartition()
	if tgtPrt != "" && err == nil {
		logger.LogInfo("Dm-verity enabled and target partition found")
		return tgtPrt, nil
	} else {
		logger.LogError(fmt.Sprintf("Error: %v, Dm-verity not enabled", err))
	}
	// Check if the system is using a default image with dm-verity enabled

	//check if the system is using a Dev image
	tgtPrt, err = getDevTargetPartition()
	if tgtPrt != "" && err == nil {
		logger.LogInfo("Dev enabled and target partition found")
		return tgtPrt, nil
	} else {
		logger.LogError(fmt.Sprintf("Error: %v, Dev not enabled", err))
	}

	return "", errors.New("failed to get any target partition (fde/default/dev) for new os")
}

func getRootfsPartition(partitionFields []string) (string, error) {

	tgtPrtn := ""
	for _, field := range partitionFields {
		if strings.Contains(field, "rootfs_") {
			tgtPrtn = field
			break
		}
	}

	return tgtPrtn, nil
}

func getFdeTargetPartition() (string, error) {

	// This is critical function identifies the target partition for systems with Full Disk Encryption (FDE) and dm-verity enabled.
	// Steps:
	// 1. Execute the "lsblk" command to retrieve the block device information, including NAME, FSTYPE, PARTLABEL, and MOUNTPOINT.
	// 2. Parse the output to find partitions with PARTLABEL starting with "rootfs" and names ending with "2" or "6".
	// 3. For each matching partition:
	//    a. Check if it has a child starting with "rootfs_" as name (e.g., "rootfs_a" or "rootfs_b").
	//    b. Ensure the child does not have a further child with "rootfs_verity" as its name, which indicates the current boot partition.
	//    c. If valid, extract the rootfs_a or rootfs_a name and return it as "/dev/mapper/<partition_name>".
	// 4. If no valid partition is found, return an error.
	//
	// Example of partition layout for FDE (sda6 is the target partition since it has no rootfs_verity as its child, and sda2 is the current booted):
	// sda                                                    100M loop
	// ├─sda1                                                  1M part        primary
	// ├─sda2                                                 48M part        rootfs_a
	// │ └─rootfs_a                                           32M crypt
	// │   └─rootfs_verity                                    32M crypt
	// ├─sda3                                                  1M part        primary
	// ├─sda4                                                  1M part        primary
	// ├─sda5                                                  1M part        primary
	// └─sda6                                                 47M part        rootfs_b
	//   └─rootfs_b                                           31M crypt

	result, err := exec.ExecuteCommand("lsblk", "-o", "NAME,FSTYPE,PARTLABEL,MOUNTPOINT")
	if err != nil {
		return "", err
	}

	// Check if the lsblk output contains the "crypt" keyword
	if !strings.Contains(string(result), "crypt") {
		return "", errors.New("FDE partition not found: no crypt partitions")
	}

	// Parse the output to find the target partition
	lines := strings.Split(string(result), "\n")
	for i := 0; i < len(lines); i++ {
		fields := strings.Fields(lines[i])

		// Only consider names ending with 2 or 6 (hardcoded rules by security dracut boot)
		if len(fields) < 3 || (!strings.HasSuffix(fields[0], "2") && !strings.HasSuffix(fields[0], "6")) {
			continue
		}

		partLabel := fields[2]
		if strings.HasPrefix(partLabel, "rootfs") && i+1 < len(lines) {
			nextFields := strings.Fields(lines[i+1])
			if len(nextFields) >= 2 && strings.Contains(lines[i+1], "rootfs_") {
				if i+2 < len(lines) {
					next2Fields := strings.Fields(lines[i+2])
					if len(next2Fields) >= 2 && strings.Contains(next2Fields[1], "rootfs_verity") {
						continue
					}
				}
				tgtPrtn, _ := getRootfsPartition(nextFields)
				if tgtPrtn != "" {
					// Remove non-alphanumeric characters from tgtPrtn
					re := regexp.MustCompile("[^a-zA-Z0-9._]+")
					tgtPrtn = re.ReplaceAllString(tgtPrtn, "")
					return "/dev/mapper/" + tgtPrtn, nil
				}
			}
		}
	}

	return "", errors.New("FDE target partition not found")
}

func getDefaultTargetPartition() (string, error) {

	// This function identifies the target partition for non-dev images without FDE or with only dm-verity enabled.
	// Steps:
	// 1. Search all partitions with a PARTLABEL starting with "rootfs" that are not mounted by "rootfs_verity".
	// 2. For each matching partition:
	//    a. Check if it is a child of "rootfs_verity". If yes, skip it as it is the current boot partition.
	//    b. Ensure the partition name ends with either "2" or "6". If not, skip it as it is not a valid target partition.
	// 3. Return the first valid partition found.
	//
	// Example of partition layout (sda2 is the target partition as sda6 is current booted):
	// root [ /home/user ]# lsblk -o NAME,FSTYPE,PARTLABEL,MOUNTPOINT
	// NAME                 FSTYPE   PARTLABEL        MOUNTPOINT
	// sda
	// |-sda1               vfat     esp              /boot/efi
	// |-sda2               ext4     rootfs
	// |-sda3               ext4     tiber_persistent /opt
	// |-sda4               ext4
	// |-sda5               ext4
	// `-sda6               ext4     rootfs
	//   `-rootfs_verity    ext4                      /

	result, err := exec.ExecuteCommand("lsblk", "-n", "-o", "NAME,FSTYPE,PARTLABEL,MOUNTPOINT")
	if err != nil {
		return "", err
	}

	// Check if the lsblk output contains the "rootfs_verity" keyword
	if !strings.Contains(string(result), "rootfs_verity") {
		return "", errors.New("DM-verity partition not found: no rootfs_verity device")
	}

	// Parse the output to find the target partition
	lines := strings.Split(string(result), "\n")
	tgtPrtn := ""
	for i := 0; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		mountPoint := ""
		if len(fields) >= 4 { // Ensure there are enough fields, including mount point
			mountPoint = fields[3]
		}
		partLabel := ""
		if len(fields) >= 3 {
			partLabel = fields[2]
		}
		if strings.HasPrefix(partLabel, "rootfs") {
			// Check if the partition is mounted at "/" or is a child of "rootfs_verity"
			if mountPoint == "/" || (i+1 < len(lines) && strings.Contains(lines[i+1], "rootfs_verity")) {
				// Skip partitions that are the current boot partition
				continue
			} else {
				// Check if the partition name ends with "2" or "6". If not, skip it.
				if strings.HasSuffix(fields[0], "2") || strings.HasSuffix(fields[0], "6") {
					logger.LogInfo("Valid target partition found:", fields[0])
					tgtPrtn = fields[0]
					break
				}
			}
		}
	}

	if tgtPrtn != "" {
		// Remove non-alphanumeric characters from tgtPrtn
		re := regexp.MustCompile("[^a-zA-Z0-9._]+")
		tgtPrtn = re.ReplaceAllString(tgtPrtn, "")
		return "/dev/" + tgtPrtn, nil
	}

	return "", errors.New("target partition not found")
}

func getDevTargetPartition() (string, error) {
	// Example of partition layout (sda6 is the target partition as sda2 is current booted):
	// root [ /home/user ]# lsblk -o NAME,FSTYPE,PARTLABEL,MOUNTPOINT
	// NAME                 FSTYPE   PARTLABEL        MOUNTPOINT
	// sda
	// |-sda1               vfat     esp              /boot/efi
	// |-sda2               ext4     rootfs			  /
	// |-sda3               ext4     tiber_persistent /opt
	// |-sda4               ext4
	// |-sda5               ext4
	// └─sda6				ext4   	 rootfs

	result, err := exec.ExecuteCommand("lsblk", "-o", "NAME,FSTYPE,PARTLABEL,MOUNTPOINT")
	if err != nil {
		return "", err
	}

	// Parse the output to find the target partition
	lines := strings.Split(string(result), "\n")
	var targetPartition string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 { // Ensure there are enough fields
			partLabel := fields[2]
			mountPoint := ""
			if len(fields) > 3 {
				mountPoint = fields[3]
			}
			if strings.HasPrefix(partLabel, "rootfs") && mountPoint == "" {
				// Ensure PARTLABEL starts with "rootfs" and MOUNTPOINT is empty
				targetPartition = fields[0]
				break
			}
		}
	}

	if targetPartition == "" {
		return "", errors.New("target partition not found")
	}

	// Remove non-alphanumeric characters from target partition
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	targetPartition = re.ReplaceAllString(targetPartition, "")
	targetPartition = "/dev/" + targetPartition

	return targetPartition, nil
}

// VerifyChecksum verifies the checksum of a given file against the provided checksum value.
func VerifyChecksum(updateImagePath, checksumValue string) error {
	// Open the file
	file, err := os.Open(updateImagePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a SHA256 hash
	hasher := sha256.New()

	// Copy the file content into the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	// Compute the checksum
	computedChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Compare the computed checksum with the provided checksum
	if computedChecksum != checksumValue {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", checksumValue, computedChecksum)
	}

	return nil
}

// DecompressImage decompresses a given image file and returns the path to the decompressed file.
func DecompressImage(outputDir, updateImagePath string) (string, error) {
	// Determine the file type based on the extension
	var outputFile string
	if strings.HasSuffix(updateImagePath, ".gz") {
		outputFile = fmt.Sprintf("%s/%s", outputDir, strings.TrimSuffix(filepath.Base(updateImagePath), ".gz"))
	} else if strings.HasSuffix(updateImagePath, ".xz") {
		outputFile = fmt.Sprintf("%s/%s", outputDir, strings.TrimSuffix(filepath.Base(updateImagePath), ".xz"))
	} else {
		return "", fmt.Errorf("unsupported file extension")
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if strings.HasSuffix(updateImagePath, ".gz") {
		// Handle .gz files
		logger.LogInfo("Decompress gz file")
		return decompressGzip(updateImagePath, outputFile)
	} else if strings.HasSuffix(updateImagePath, ".xz") {
		logger.LogInfo("Copy xz file to %s", outputDir)
		input, err := os.Open(updateImagePath)
		if err != nil {
			return "", fmt.Errorf("failed to open update image: %w", err)
		}
		defer input.Close()

		copiedPath := filepath.Join(outputDir, filepath.Base(updateImagePath))
		output, err := os.Create(copiedPath)
		if err != nil {
			return "", fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()

		if _, err := io.Copy(output, input); err != nil {
			return "", fmt.Errorf("failed to copy update image: %w", err)
		}
		updateImagePath = copiedPath
		logger.LogInfo("Decompress xz file")
		// Handle .xz files using ExecuteCommand
		return decompressXz(updateImagePath, outputFile)
	}

	return "", fmt.Errorf("unsupported file extension")
}

// decompressGzip handles the decompression of .gz files
func decompressGzip(inputPath, outputPath string) (string, error) {
	// Open the input .gz file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open update image: %w", err)
	}
	defer inputFile.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create the output file
	outputFileHandle, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create decompressed image file: %w", err)
	}
	defer outputFileHandle.Close()

	// Decompress the input file into the output file
	if err := decompress(gzipReader, outputFileHandle); err != nil {
		return "", fmt.Errorf("failed to decompress image: %w", err)
	}

	return outputPath, nil
}

// decompressXz handles the decompression of .xz files using ExecuteCommand
func decompressXz(inputPath, outputPath string) (string, error) {
	// Use the xz command to decompress the file
	logger.LogInfo("Decompressing .xz file: %s", inputPath+" to "+outputPath)
	_, err := exec.ExecuteCommand("xz", "-d", inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to decompress .xz image: %w", err)
	}

	return outputPath, nil
}

// decompress is a helper function to handle the decompression logic
func decompress(input io.Reader, output io.Writer) error {
	// Copy the content from the input reader to the output writer
	_, err := io.Copy(output, input)
	if err != nil {
		return fmt.Errorf("error during decompression: %w", err)
	}
	return nil
}

// GetPartitionUUID retrieves the PARTUUID of a current partition.
func GetPartitionUUID(partition string) (string, error) {
	// Execute the "blkid" command to get the PARTUUID of the partition
	result, err := exec.ExecuteCommand("blkid", "-s", "PARTUUID", "-o", "value", partition)
	if err != nil {
		return "", fmt.Errorf("failed to get current partition PARTUUID: %w", err)
	}

	// Trim any whitespace from the result
	return strings.TrimSpace(string(result)), nil
}

// LoopSetup mounts the update image to a loop device.
func LoopSetup(imagePath string) (string, error) {
	// Execute the "losetup" command to create a loop device for the image
	result, err := exec.ExecuteCommand("losetup", "--find", "--show", "-P", imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create loop device: %w", err)
	}

	// Trim any whitespace from the result
	loopDevice := strings.TrimSpace(string(result))

	return loopDevice, nil
}

// LoopUnSetup deletes the specified loop device.
func LoopUnSetup(loopDevice string) error {
	// Execute the "losetup" command to detach the loop device
	_, err := exec.ExecuteCommand("losetup", "-d", loopDevice)
	if err != nil {
		return fmt.Errorf("failed to delete loop device %s: %w", loopDevice, err)
	}

	return nil
}

// MountDevice mounts a device to a directory with optional read-only mode.
// If readOnly is true, it performs "mount -o ro $SOURCEDEV $SOURCECOPY".
// Otherwise, it performs "mount $TARGETDEV $DESTCOPY".
func MountDevice(device, mountPoint string, readOnly bool, option ...string) error {
	// Construct the mount command
	var mountCmd []string
	if readOnly {
		mountCmd = []string{"mount", "-o", "ro"}
	} else {
		mountCmd = []string{"mount"}
	}

	// Append optional parameters if provided
	if len(option) > 0 {
		mountCmd = append(mountCmd, option...)
	}

	// Append device and mount point
	mountCmd = append(mountCmd, device, mountPoint)

	logger.LogInfo("Mount command:", strings.Join(mountCmd, " "))
	// Execute the mount command
	_, err := exec.ExecuteCommand(mountCmd[0], mountCmd[1:]...)
	if err != nil {
		return fmt.Errorf("failed to mount device %s to %s: %w", device, mountPoint, err)
	}

	return nil
}

// UnmountDevice unmounts a device from a directory.
func UnmountDevice(mountPoint string) error {
	// Execute the unmount command
	_, err := exec.ExecuteCommand("umount", "-l", mountPoint)
	if err != nil {
		return fmt.Errorf("failed to unmount %s: %w", mountPoint, err)
	}

	return nil
}

// GetImageUUID retrieves the PARTUUID from the update image.
func GetImageUUID(imagePath string) (string, error) {
	// Execute the "blkid" command to get the PARTUUID of the image
	result, err := exec.ExecuteCommand("blkid", "-s", "PARTUUID", "-o", "value", imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to get PARTUUID from image: %w", err)
	}

	// Trim any whitespace from the result
	return strings.TrimSpace(string(result)), nil
}

// WriteRootfsToPartition writes the rootfs to the target partition.
func WriteRootfsToPartition(source, target string) error {
	// Delete old rootfs and copy new
	err := os.RemoveAll(filepath.Join(target, "*"))
	if err != nil {
		return fmt.Errorf("failed to remove old OS: %w", err)
	}

	// Sync to ensure changes are written to disk
	_, err = exec.ExecuteCommand("sync")
	if err != nil {
		return fmt.Errorf("failed to sync after removing old OS: %w", err)
	}

	// Copy new rootfs
	logger.LogInfo("Executing command: cp -rp %s %s\n", filepath.Join(source, "*"), filepath.Join(target, "/"))
	_, err = exec.ExecuteCommand("sh", "-c", fmt.Sprintf("cp -rp %s/* %s/", filepath.Clean(source), filepath.Clean(target)))
	if err != nil {
		// Cleanup on failure
		_ = os.RemoveAll(filepath.Join(target, "*"))
		return fmt.Errorf("failed to copy latest OS: %w", err)
	}
	return nil
}

// CreateSecureDir creates a secure directory with restrictive permissions.
// If the directory already exists, it removes it first.
func CreateSecureDir(dirPath string) error {
	// Remove the existing directory if it exists
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to remove existing directory %s: %w", dirPath, err)
	}

	// Create the directory with restrictive permissions
	err = os.MkdirAll(dirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Set restrictive permissions
	err = os.Chmod(dirPath, 0700)
	if err != nil {
		// Cleanup on failure
		_ = os.RemoveAll(dirPath)
		return fmt.Errorf("failed to set permissions for directory %s: %w", dirPath, err)
	}

	return nil
}

func DeleteDir(dirPath string) error {
	// Remove the directory and all its contents
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to delete directory %s: %w", dirPath, err)
	}

	return nil
}

// CreateSecureTempDir creates a secure temporary directory inside a given directory.
func CreateSecureTempDir(secureTempDir string) (string, error) {
	// Remove existing directory if it exists
	if _, err := os.Stat(secureTempDir); err == nil {
		err = os.RemoveAll(secureTempDir)
		if err != nil {
			return "", fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Create the secure directory
	err := CreateSecureDir(secureTempDir)
	if err != nil {
		return "", fmt.Errorf("failed to create secure directory: %w", err)
	}

	// Create a temporary directory inside the secure directory
	tempFolder, err := os.MkdirTemp(secureTempDir, "temp-*")
	if err != nil {
		// Cleanup on failure
		_ = os.RemoveAll(secureTempDir)
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	return tempFolder, nil
}

// CopyOnboardingVariable copies onboarding variables (e.g., hostname) to the target device.
func CopyOnboardingVariable(target string) error {
	// Copy the hostname file to the destination
	srcHostname := "/etc/hostname"
	destHostname := filepath.Join(target, "etc", "hostname")
	_, err := exec.ExecuteCommand("cp", "-r", srcHostname, destHostname)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcHostname, destHostname, err)
	}

	// Log success
	logger.LogInfo("Success: Copy onboarding var.")
	return nil
}

// CopyUKIToBootPartition copies the UKI to the boot partition with a ".bak" extension.
func CopyUKIToBootPartition(source string) error {
	err := CreateSecureDir(UKIUpdatePath)
	if err != nil {
		return fmt.Errorf("failed to create secure directory %s: %w", UKIUpdatePath, err)
	}

	// Validate inputs
	if strings.TrimSpace(source) == "" || strings.TrimSpace(UKIUpdate) == "" {
		return fmt.Errorf("source and UKIUpdate paths cannot be empty")
	}

	// Construct the command arguments
	args := []string{"if=" + source, "of=" + UKIUpdate, "bs=4M"}

	// Execute the command
	_, err = exec.ExecuteCommand("dd", args...)
	if err != nil {
		// Cleanup on failure
		_ = os.RemoveAll(UKIUpdatePath)
		return fmt.Errorf("failed to write EFI source file to destination: %w", err)
	}

	// Run sync to ensure data is written to disk
	_, err = exec.ExecuteCommand("sync")
	if err != nil {
		return fmt.Errorf("failed to sync data to disk: %w", err)
	}

	// Log success
	logger.LogInfo("Success: Copy UKI.")
	return nil
}

// GetActivePartition retrieves the active partition.
func GetActivePartition() (string, error) {
	logger.LogInfo("Get Active Partition.")

	var currentName, currentSubname string

	// Execute the lsblk command to get partition details
	output, err := exec.ExecuteCommand("lsblk", "-n", "-o", "NAME,FSTYPE,MOUNTPOINT")
	if err != nil {
		return "", fmt.Errorf("failed to execute lsblk command: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		// Parse the line into name, subname, fstype, and mountpoint
		fields := strings.Fields(line)

		name := cleanName(fields[0])
		subname := ""
		// subname dont exist in dev image
		if len(fields) > 3 {
			subname = cleanName(fields[1])
		}
		mountpoint := fields[len(fields)-1]

		// Check if the mountpoint is "/"
		if mountpoint == "/" {
			logger.LogInfo("Name: %s, Subname: %s, Mountpoint: %s\n", name, subname, mountpoint)
			// check active partition for fde+dm verity
			if strings.HasPrefix(currentSubname, "rootfs") {
				logger.LogInfo("Active FDE partition: /dev/%s\n", currentName)
				return "/dev/" + currentName, nil
			}

			// check active partition for dm verity
			if strings.Contains(name, "verity") || strings.Contains(subname, "verity") {
				if !IsPartIndexAllowed(currentName) {
					continue
				}
				logger.LogInfo("Active partition: /dev/%s\n", currentName)
				return "/dev/" + currentName, nil
			} else {
				// check active partition for dev
				logger.LogInfo("Active Dev partition: /dev/%s\n", name)
				return "/dev/" + name, nil
			}
		}

		// Update currentName and currentSubname for the next iteration
		if name != "" {
			currentName = name
		}
		if subname != "" {
			currentSubname = subname
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading lsblk output: %w", err)
	}

	return "", errors.New("failed to find active partition")
}

// cleanName removes all leading non-alphanumeric characters and spaces from the input name.
func cleanName(name string) string {
	// Use a regular expression to remove all leading non-alphanumeric characters and spaces
	re := regexp.MustCompile(`^[^a-zA-Z0-9]*`)
	name = re.ReplaceAllString(name, "")
	return name
}

// CheckWriteDone checks if the write operation is done by looking for a specific file.
// It returns true if the file exists, otherwise false.
func CheckWriteDone() bool {
	if _, err := os.Stat(UKIUpdate); os.IsNotExist(err) {
		return false
	}
	return true
}

func ResetWriteDone() bool {
	// Remove the .bak file
	err := DeleteDir(UKIUpdatePath)
	return err == nil
}

// ApplyBoot applies the UKI based on the active UKI
func ApplyBoot(nextUKI string) error {
	logger.LogInfo("Apply UKI command")
	fullNextUKI := filepath.Join(UKIPATH, nextUKI)

	// Execute the dd command
	ddCmd, err := exec.ExecuteCommand("dd", fmt.Sprintf("if=%s", UKIUpdate), fmt.Sprintf("of=%s", fullNextUKI), "bs=4M")
	if err != nil {
		return fmt.Errorf("failed to execute lsblk command: %w", err)
	}

	logger.LogInfo("Write result:", string(ddCmd))

	// Execute the sync command
	_, err = exec.ExecuteCommand("sync")
	if err != nil {
		return fmt.Errorf("failed to execute sync command: %w", err)
	}

	logger.LogInfo("Success: Apply UKI.")
	return nil
}

// GetUUIDfromUKI retrieves the UUID from the UKI file.
func GetUUIDfromUKI(ukiFile string) (string, error) {
	logger.LogInfo("Get UUID from UKI file.")
	boot_uuid, err := extractUUIDfromUKI(ukiFile, "boot_uuid")
	if err != nil {
		return "", fmt.Errorf("failed to get Boot UUID: %w", err)
	}

	if boot_uuid == "" {
		boot_uuid, err = extractUUIDfromUKI(ukiFile, "PARTUUID")
		if err != nil {
			return "", fmt.Errorf("failed to get Part UUID: %w", err)
		}

		if boot_uuid == "" {
			return "", fmt.Errorf("failed to get Boot/Part UUID: %w", err)
		}

	}

	return boot_uuid, nil
}

// extractSubstring extracts characters from the 11th to the 46th position of the input string.
func extractSubstring(input string, start, end int) string {
	if len(input) < end {
		return ""
	}
	substring := input[start:end] // Extract characters from the 11th to the 46th position.

	if !ValidateUUID(substring) {
		return ""
	}

	return substring
}

func ValidateUUID(input string) bool {
	// Validate if the input is in UUID format.
	uuidRegex := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	return uuidRegex.MatchString(input)
}

// extractUUIDfromUKI extracts the UUID from the UKI file based on the provided identifier.
func extractUUIDfromUKI(ukiFile string, identifier string) (string, error) {
	logger.LogInfo("Get UUID from UKI file.")
	var start, end int
	// Determine the start and end positions based on the identifier
	if identifier == "PARTUUID" {
		start = 9
		end = 45
	} else if identifier == "boot_uuid" {
		start = 10
		end = 46
	} else {
		return "", fmt.Errorf("invalid identifier: %s", identifier)
	}

	// Extract the UUID from the UKI file based on the identifier
	grepPattern := fmt.Sprintf("%s=.* ", identifier)
	output, err := exec.ExecuteCommand("grep", "-a", "-h", "-o", grepPattern, ukiFile)
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %w", identifier, err)
	}
	cleanOutput := strings.TrimSpace(string(output))
	extractedUUID := extractSubstring(cleanOutput, start, end)

	return extractedUUID, nil
}

// FindFirstFile searches for the first file in the specified directory and returns its path.
func FindFirstFile(directory string) (string, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			return filepath.Join(directory, file.Name()), nil
		}
	}

	return "", fmt.Errorf("no files found in directory: %s", directory)
}

// FindPartitionByUUID searches for a partition by its UUID.
func FindPartitionByUUID(uuid string) string {
	var output string
	var err error
	result := ""

	// Check for UUID
	output, err = exec.ExecuteCommand("blkid", "-o", "device", "-t", "UUID="+uuid)
	if err == nil && strings.TrimSpace(output) != "" {
		result = strings.TrimSpace(output)
	}

	// Check for PARTUUID
	output, err = exec.ExecuteCommand("blkid", "-o", "device", "-t", "PARTUUID="+uuid)
	if err == nil && strings.TrimSpace(output) != "" {
		result = strings.TrimSpace(output)
	}

	// Check for TYPE=crypto_LUKS and UUID
	output, err = exec.ExecuteCommand("blkid", "-o", "device", "-t", "TYPE=crypto_LUKS,UUID="+uuid)
	if err == nil && strings.TrimSpace(output) != "" {
		result = strings.TrimSpace(output)
	}

	lines := strings.Split(string(result), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "/dev/loop") {
			return strings.TrimSpace(line)
		}
	}

	return ""
}

// IsLoopDevice checks if the given device is a loop device.
func IsLoopDevice(device string) bool {
	return strings.HasPrefix(device, "/dev/loop")
}

// AddLogin adds a login user in a chroot environment.
func AddLogin(workDir, targetDev string) error {
	chrootDir := filepath.Join(workDir, "abchroot")
	username := "user"

	// Create a secure directory
	err := CreateSecureDir(chrootDir)
	if err != nil {
		return fmt.Errorf("failed to create secure directory: %w", err)
	}

	// Mount necessary filesystems
	mounts := []struct {
		source string
		target string
		flags  []string
	}{
		{targetDev, chrootDir, nil},
		{"/dev", filepath.Join(chrootDir, "dev"), []string{"--bind"}},
		{"/dev/pts", filepath.Join(chrootDir, "dev/pts"), []string{"--bind"}},
		{"/proc", filepath.Join(chrootDir, "proc"), []string{"--bind"}},
		{"/sys", filepath.Join(chrootDir, "sys"), []string{"--bind"}},
	}

	for _, m := range mounts {
		err = MountDevice(m.source, m.target, false, m.flags...)
		if err != nil {
			// Cleanup on failure
			for _, mount := range mounts {
				UnmountDevice(mount.target)
			}
			if chrootDir != "" {
				DeleteDir(chrootDir)
			}
			return fmt.Errorf("failed to mount %s to %s: %w", m.source, m.target, err)
		}
	}

	// Execute chroot commands
	chrootCmd := fmt.Sprintf(`
		# Create the user
		useradd -m -s /bin/bash "%s"

		# Set the password
		echo '%s:$6$BTZupwxuptVcnJ2q$aKz3z0XxjPW0EI7r90/xfgMH.2J5dNB9V2jPbFPu0.NwioQh66VmyjVrG2uQuJnUu2d3MSvHqUiqGdU0VxFKA0' | chpasswd -e

		# Add the user to the sudo group
		usermod -aG sudo "%s"
	`, username, username, username)

	_, err = exec.ExecuteCommand("chroot", chrootDir, "/bin/bash", "-c", chrootCmd)
	if err != nil {
		// Cleanup on failure
		for _, mount := range mounts {
			UnmountDevice(mount.target)
		}
		if chrootDir != "" {
			DeleteDir(chrootDir)
		}
		return fmt.Errorf("failed to add user %s in chroot environment %s: %w", username, chrootDir, err)
	}

	// Unmount filesystems
	for i := len(mounts) - 1; i >= 0; i-- {
		err = UnmountDevice(mounts[i].target)
		if err != nil {
			return fmt.Errorf("failed to unmount %s: %w", mounts[i].target, err)
		}
	}

	// Remove the chroot directory
	err = DeleteDir(chrootDir)
	if err != nil {
		return fmt.Errorf("failed to remove chroot directory %s: %w", chrootDir, err)
	}

	logger.LogInfo("User %s added successfully in chroot environment %s.\n", username, chrootDir)
	return nil
}

// RelabelSELinux relabels SELinux contexts in a chroot environment.
func RelabelSELinux(workDir, targetDev string) error {
	updateMountPoint := filepath.Join(workDir, "abchroot")
	// Create a secure directory
	err := CreateSecureDir(updateMountPoint)
	if err != nil {
		return fmt.Errorf("failed to create secure directory: %w", err)
	}

	// Mount necessary filesystems
	mounts := []struct {
		source string
		target string
		flags  []string
	}{
		{targetDev, updateMountPoint, nil},
		{"/dev", filepath.Join(updateMountPoint, "dev"), []string{"--bind"}},
		{"/dev/pts", filepath.Join(updateMountPoint, "dev/pts"), []string{"--bind"}},
		{"/proc", filepath.Join(updateMountPoint, "proc"), []string{"--bind"}},
		{"/sys", filepath.Join(updateMountPoint, "sys"), []string{"--bind"}},
	}

	for _, m := range mounts {
		err = MountDevice(m.source, m.target, false, m.flags...)
		if err != nil {
			// Cleanup on failure
			for _, mount := range mounts {
				UnmountDevice(mount.target)
			}
			if updateMountPoint != "" {
				DeleteDir(updateMountPoint)
			}
			return fmt.Errorf("failed to mount %s to %s: %w", m.source, m.target, err)
		}
	}

	// Execute the SELinux relabeling command in the chroot environment
	chrootCmd := `
		setfiles -m -v /etc/selinux/targeted/contexts/files/file_contexts /
	`
	_, err = exec.ExecuteCommand("chroot", updateMountPoint, "/bin/bash", "-c", chrootCmd)
	if err != nil {
		// Cleanup on failure
		for _, mount := range mounts {
			UnmountDevice(mount.target)
		}
		if updateMountPoint != "" {
			DeleteDir(updateMountPoint)
		}
		return fmt.Errorf("failed to relabel SELinux contexts in chroot environment %s: %w", updateMountPoint, err)
	}

	// Unmount filesystems
	for i := len(mounts) - 1; i >= 0; i-- {
		err = UnmountDevice(mounts[i].target)
		if err != nil {
			return fmt.Errorf("failed to unmount %s: %w", mounts[i].target, err)
		}
	}

	// Remove the update mount point directory
	err = DeleteDir(updateMountPoint)
	if err != nil {
		return fmt.Errorf("failed to remove update mount point %s: %w", updateMountPoint, err)
	}

	logger.LogInfo("SELinux relabeling completed successfully.")
	return nil
}

// CheckFDEOn checks if Full Disk Encryption (FDE) is enabled.
func CheckFDEOn() bool {
	// Execute the "dmsetup ls --target crypt" command to check for crypt devices
	output, err := exec.ExecuteCommand("dmsetup", "ls", "--target", "crypt")
	if err != nil {
		return false // Return false if the command fails
	}

	// Check if the output contains "No devices found"
	if strings.Contains(string(output), "No devices found") {
		return false
	}

	// Check if the output is empty
	if strings.TrimSpace(string(output)) == "" {
		return false
	}

	// Check if the output contains both "rootfs_a" and "rootfs_b"
	if strings.Contains(string(output), "rootfs_a") && strings.Contains(string(output), "rootfs_b") {
		return true // FDE is enabled
	}

	return false // Return false if the devices are not recognized
}

// CheckDMVerityOn checks if dm-verity is enabled.
func CheckDMVerityOn() bool {
	// Execute the "lsblk -nr -o NAME,FSTYPE,MOUNTPOINT" command to check mountpoints
	output, err := exec.ExecuteCommand("lsblk", "-nr", "-o", "NAME,FSTYPE,MOUNTPOINT")
	if err != nil {
		return false // Return false if the command fails
	}

	// Count the number of root ("/") mountpoints
	rootCount := 0
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == "/" {
			rootCount++
		}
	}

	// Return true if there are 2 or more "/" mountpoints
	return rootCount >= 2
}

// SetUUID sets a new UUID for a given partition.
// Usage: SetUUID("/dev/sda3", "NEW-UUID", true, false)
func SetUUID(devicePartition, newUUID string) error {

	dmv := CheckDMVerityOn()
	luks := CheckFDEOn()

	if luks {
		// Handle LUKS UUID
		blockDevice, err := GetBlockName(devicePartition)
		if err != nil {
			return fmt.Errorf("failed to get block device %w", err)
		}

		logger.LogInfo("Change LUKS UUID for %s\n", blockDevice)

		_, err = exec.ExecuteCommand("cryptsetup", "luksUUID", "--batch-mode", blockDevice, "--uuid", newUUID)
		if err != nil {
			return fmt.Errorf("failed to set LUKS UUID: %w", err)
		}
	} else if dmv {
		// Handle partition UUID with FDE enabled
		logger.LogInfo("Change partition UUID with DM-verity enabled for %s\n", devicePartition)
		_, err := exec.ExecuteCommand("e2fsck", "-f", devicePartition, "-y")
		if err != nil {
			return fmt.Errorf("failed to run e2fsck: %w", err)
		}
		_, err = exec.ExecuteCommand("tune2fs", "-U", newUUID, devicePartition)
		if err != nil {
			return fmt.Errorf("failed to set partition UUID: %w", err)
		}
	} else {
		// Handle partition UUID without FDE
		logger.LogInfo("Change partition UUID without FDE for %s\n", devicePartition)
		baseDevice := strings.TrimRight(devicePartition, "0123456789")
		partitionNumber := strings.TrimPrefix(devicePartition, baseDevice)
		_, err := exec.ExecuteCommand("sfdisk", "--part-uuid", baseDevice, partitionNumber, newUUID)
		if err != nil {
			return fmt.Errorf("failed to set partition UUID: %w", err)
		}
	}

	return nil
}

func IsPartIndexAllowed(targetDev string) bool {
	// Extract the last character of targetDev as the index
	fsIndex := targetDev[len(targetDev)-1:]
	fsIndexInt, err := strconv.Atoi(fsIndex)
	if err != nil {
		return false
	}

	// Check if the index matches INDEXPARTA or INDEXPARTB
	if fsIndexInt == INDEXPARTA || fsIndexInt == INDEXPARTB {
		return true
	}

	return false
}

// SetVerity sets up dm-verity for a given partition.
func SetVerity(workDir, targetDev string) error {
	logger.LogInfo("Set verity for %s.\n", targetDev)

	dmv := CheckDMVerityOn()
	fde := CheckFDEOn()

	if !dmv && !fde {
		logger.LogInfo("skipping set verity : no dm-verity or FDE enabled for %s", targetDev)
		return nil
	}

	tempDir := filepath.Join(workDir, "abroothash")
	err := CreateSecureDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to create secure directory: %w", err)
	}
	defer func() {
		_ = UnmountDevice(tempDir)
		_ = os.RemoveAll(tempDir)
	}()

	if fde {
		logger.LogInfo("FDE is enabled.")
		device := filepath.Base(targetDev)

		err = MountDevice("/dev/mapper/ver_roothash", tempDir, false)
		if err != nil {
			return fmt.Errorf("failed to mount ver_roothash: %w", err)
		}

		if device == "rootfs_b" {
			logger.LogInfo("Verity Setup on rootfs_b")
			output, err := exec.ExecuteCommand("veritysetup", "format", "/dev/mapper/rootfs_b", "/dev/mapper/root_b_ver_hash_map")
			if err != nil {
				return fmt.Errorf("failed to set up verity for rootfs_b: %w", err)
			}
			rootHash := extractRootHash(output)
			err = os.WriteFile(filepath.Join(tempDir, "part_b_roothash"), []byte(rootHash), 0644)
			if err != nil {
				return fmt.Errorf("failed to write root hash for rootfs_b: %w", err)
			}
		} else {
			logger.LogInfo("Verity Setup on rootfs_a")
			output, err := exec.ExecuteCommand("veritysetup", "format", "/dev/mapper/rootfs_a", "/dev/mapper/root_a_ver_hash_map")
			if err != nil {
				return fmt.Errorf("failed to set up verity for rootfs_a: %w", err)
			}
			rootHash := extractRootHash(output)
			err = os.WriteFile(filepath.Join(tempDir, "part_a_roothash"), []byte(rootHash), 0644)
			if err != nil {
				return fmt.Errorf("failed to write root hash for rootfs_a: %w", err)
			}
		}
	} else if dmv {
		logger.LogInfo("DM Verity is enabled.")
		// index mapping for DM verity only
		INDEXHASHPART := "7"
		INDEXMAPPARTA := "4"
		INDEXMAPPARTB := "5"

		fnReturn, err := ReplaceDeviceIndex(targetDev, INDEXHASHPART)
		if err != nil {
			return fmt.Errorf("failed to replace device index: %w", err)
		}

		err = MountDevice(fnReturn, tempDir, false)
		if err != nil {
			return fmt.Errorf("failed to mount device: %w", err)
		}
		defer func() {
			_ = UnmountDevice(tempDir)
		}()

		fsIndex := targetDev[len(targetDev)-1:]
		fsIndexInt, err := strconv.Atoi(fsIndex)
		if err != nil {
			return fmt.Errorf("failed to convert fsIndex to integer: %w", err)
		}
		if fsIndexInt == INDEXPARTA {
			logger.LogInfo("Verity Setup on %s\n", targetDev)
			fnReturn, err = ReplaceDeviceIndex(targetDev, INDEXMAPPARTA)
			if err != nil {
				return fmt.Errorf("failed to replace device index for part A: %w", err)
			}
			output, err := exec.ExecuteCommand("veritysetup", "format", targetDev, fnReturn)
			if err != nil {
				return fmt.Errorf("failed to set up verity for part A: %w", err)
			}
			rootHash := extractRootHash(output)
			err = os.WriteFile(filepath.Join(tempDir, "part_a_roothash"), []byte(rootHash), 0644)
			if err != nil {
				return fmt.Errorf("failed to write root hash for part A: %w", err)
			}
		} else if fsIndexInt == INDEXPARTB {
			logger.LogInfo("Verity Setup on %s\n", targetDev)
			fnReturn, err = ReplaceDeviceIndex(targetDev, INDEXMAPPARTB)
			if err != nil {
				return fmt.Errorf("failed to replace device index for part B: %w", err)
			}
			output, err := exec.ExecuteCommand("veritysetup", "format", targetDev, fnReturn)
			if err != nil {
				return fmt.Errorf("failed to set up verity for part B: %w", err)
			}
			rootHash := extractRootHash(output)
			err = os.WriteFile(filepath.Join(tempDir, "part_b_roothash"), []byte(rootHash), 0644)
			if err != nil {
				return fmt.Errorf("failed to write root hash for part B: %w", err)
			}
		}
	}

	_, err = exec.ExecuteCommand("sync")
	if err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	return nil
}

// extractRootHash extracts the root hash from the veritysetup output.
func extractRootHash(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Root hash") {
			parts := strings.Fields(line)
			if len(parts) > 2 {
				return parts[2]
			}
		}
	}
	return ""
}

// ReplaceDeviceIndex replaces the index of a device with a new index.
func ReplaceDeviceIndex(device string, newIndex string) (string, error) {
	if device == "" || newIndex == "" {
		return "", errors.New("device or new index cannot be empty")
	}
	return fmt.Sprintf("%s%s", strings.TrimRight(device, "0123456789"), newIndex), nil
}

// CheckFileExists checks if a file exists at the given path.
func CheckFirstUKIExists() bool {
	_, err := os.Stat(UKIPATH + "/linux.efi")
	return err == nil
}

// RenameEFI renames the first EFI file found in UKIPath to linux.efi.
func RenameEFI() error {
	// Find the first file in the UKIPath directory
	source, err := FindFirstFile(UKIPATH)
	if err != nil {
		return fmt.Errorf("failed to find EFI file in %s: %w", UKIPATH, err)
	}

	// Define the destination path
	destination := filepath.Join(UKIPATH, "linux.efi")

	// Rename the file
	err = os.Rename(source, destination)
	if err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", source, destination, err)
	}

	logger.LogInfo("Successfully renamed %s to %s\n", source, destination)
	return nil
}

// GetBlockName extracts the block name for a given rootfs device.
func GetBlockName(device string) (string, error) {
	// check if not contain rootfs_a or rootfs_b
	if !strings.Contains(device, "rootfs_a") && !strings.Contains(device, "rootfs_b") {
		return device, nil
	}
	// Extract the base name of the device
	baseDevice := filepath.Base(device)
	var blockName string

	logger.LogInfo("Get block name for %s.\n", baseDevice)

	// Execute the lsblk command to get partition details
	output, err := exec.ExecuteCommand("lsblk", "-n", "-o", "NAME,FSTYPE,MOUNTPOINT")
	if err != nil {
		return "", fmt.Errorf("failed to execute lsblk command: %w", err)
	}

	// Parse the output line by line
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()

		// Parse the line into name, subname, and mountpoint
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := cleanName(fields[0])
		subname := ""
		if len(fields) > 2 {
			subname = cleanName(fields[1])
		}

		// Check if the subname matches the base device
		if subname == baseDevice {
			return "/dev/" + blockName, nil
		}

		// Accumulate values for blockName
		if name != "" {
			blockName = name
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading lsblk output: %w", err)
	}

	return "", errors.New("block name not found")
}
