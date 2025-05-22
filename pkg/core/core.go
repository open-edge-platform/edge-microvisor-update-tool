package core

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	exec "os.abupdate.tool/pkg/exec"
)

// ExecuteCommand runs a Linux command passed as a string and returns the output and error if any.
func GetTargetPartition() (string, error) {

	// Check if the system is using FDE (Full Disk Encryption) enabled image
	tgtPrt, err := getFdeTargetPartition()
	if tgtPrt != "" && err == nil {
		fmt.Println("FDE enabled")
		return tgtPrt, nil
	} else {
		fmt.Println("FDE not enabled")
	}
	// Check if the system is using a default image with dm-verity enabled

	// Check if the system is using a default OR dm-verity enabled only image but NO FDE enabled
	tgtPrt, err = getDefaultTargetPartition()
	if tgtPrt != "" && err == nil {
		fmt.Println("Default target partition found")
		return tgtPrt, nil
	} else {
		fmt.Println("Default target partition not found")
	}
	// Check if the system is using a default image with dm-verity enabled

	//check if the system is using a Dev image
	tgtPrt, err = getDevTargetPartition()
	if tgtPrt != "" && err == nil {
		fmt.Println("Dev target partition found")
		return tgtPrt, nil
	} else {
		fmt.Println("Dev target partition not found")
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
					re := regexp.MustCompile("[^a-zA-Z0-9._-]+")
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

	result, err := exec.ExecuteCommand("lsblk", "-o", "NAME,FSTYPE,PARTLABEL,MOUNTPOINT")
	if err != nil {
		return "", err
	}

	// Parse the output to find the target partition
	lines := strings.Split(string(result), "\n")
	tgtPrtn := ""
	for i := 0; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) >= 3 { // Ensure there are enough fields
			partLabel := fields[2]
			if strings.HasPrefix(partLabel, "rootfs") {
				// Check the next line for "rootfs_verity"
				if i+1 < len(lines) {
					nextFields := strings.Fields(lines[i+1])
					if len(nextFields) >= 2 && strings.Contains(lines[i+1], "rootfs_verity") {
						// Skip partitions that are children of "rootfs_verity", which is the current boot partition
						continue
					} else {
						// Check if the partition name ends with "2" or "6". If not, skip it.
						if strings.HasSuffix(fields[0], "2") || strings.HasSuffix(fields[0], "6") {
							fmt.Println("Valid target partition found:", fields[0])
							tgtPrtn = fields[0]
							break
						}
					}
				}
			}
		}
	}

	if tgtPrtn != "" {
		// Remove non-alphanumeric characters from tgtPrtn
		re := regexp.MustCompile("[^a-zA-Z0-9._-]+")
		tgtPrtn = re.ReplaceAllString(tgtPrtn, "")
		return "/dev/" + tgtPrtn, nil
	}

	return "", errors.New("target partition not found")
}

func getDevTargetPartition() (string, error) {

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

	// Append target partition information to the result
	var buffer bytes.Buffer
	buffer.WriteString("\nTarget Partition: ")
	buffer.WriteString(targetPartition)

	return buffer.String(), nil
}
