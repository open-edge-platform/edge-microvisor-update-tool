package flashing

import (
	"fmt"

	core "os.abupdate.tool/pkg/core"
)

func WritePartition(updateImagePath string, checksumValue string) (int, error) {
	//TODO:
	//1. Decompress the update image file. Use a library like "archive/tar" or "compress/gzip" if the image is compressed.
	//   Ensure the decompression process handles errors and validates the integrity of the decompressed data.
	//2. Get the next available partition for flashing. Implement logic in getNextPartition() to dynamically determine
	//   the partition based on system state, considering features like FDE (Full Disk Encryption) or dm-verity.
	//3. Write the decompressed image to the identified partition. Use system utilities or libraries like "os" or "io"
	//   to perform the write operation. Validate the write process by checking the written data.
	//4. Copy the UKI (Unified Kernel Image) to the boot partition with a ".bak" extension. Ensure the boot partition
	//   is writable and the copy operation does not overwrite critical files. Use "os" or "io" for file operations.

	nxtPartition, err := getNextPartition()
	if err != nil {
		return -1, fmt.Errorf("failed to get next partition: %w", err)
	}

	fmt.Printf("Writing rootfs partition:\n")
	fmt.Printf("  Update Image Path: %s\n", updateImagePath)
	fmt.Printf("  Checksum: %s\n", checksumValue)
	fmt.Printf("  Target Partition: %s\n", nxtPartition)

	return 0, nil
}

func getNextPartition() (string, error) {

	//todo: Determine the next available partition for flashing.
	// High-level approach:
	// 1. Use system utilities like "lsblk" or "blkid" to list available partitions.
	// 2. Parse the output to identify partitions that are not currently in use.
	// 3. Consider system-specific features like FDE (Full Disk Encryption) or dm-verity.
	// 4. Use "os/exec" to execute system commands and process their output.
	// Example: exec.Command("lsblk", "-o", "NAME,MOUNTPOINT").Output()
	//this is just a placeholder
	//this should be implemented to get the next available partition for flashing
	//the security feature like enablement of FDE/dm-verity has to be considered

	// Example of calling the ExecuteCommand function
	targetPartition, err := core.GetTargetPartition()
	if err != nil {
		return "", fmt.Errorf("failed to get target partition: %w", err)
	}

	return targetPartition, nil
}
