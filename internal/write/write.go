package flashing

import (
	"fmt"

	core "os.abupdate.tool/pkg/core"
	"os.abupdate.tool/pkg/logger"
)

// Helper function to clean up resources
func cleanupResources(tempDir string, loopDev string, mounts ...string) {
	for _, mount := range mounts {
		core.UnmountDevice(mount)
	}
	if tempDir != "" {
		core.DeleteDir(tempDir)
	}
	if loopDev != "" {
		core.LoopUnSetup(loopDev)
	}
}

func WritePartition(updateImagePath string, checksumValue string, devMode bool) error {
	if devMode {
		logger.LogInfo("Development mode is active.")
	}

	nxtPartition, err := core.GetTargetPartition()
	if err != nil {
		logger.LogError("Failed to get next partition: %v", err)
		return fmt.Errorf("failed to get next partition: %w", err)
	}

	currPartition, err := core.GetActivePartition()
	if err != nil {
		logger.LogError("Failed to get current boot partition: %v", err)
		return fmt.Errorf("failed to get current boot partition: %w", err)
	}
	logger.LogInfo("Current Boot Partition: %s", currPartition)

	logger.LogInfo("Writing rootfs partition:")
	logger.LogInfo("  Update Image Path: %s", updateImagePath)
	logger.LogInfo("  Checksum: %s", checksumValue)
	logger.LogInfo("  Target Partition: %s", nxtPartition)

	logger.LogInfo("Verifying Image with checksum")

	// Verify the update image path with the provided checksum
	verified := core.VerifyChecksum(updateImagePath, checksumValue)
	if verified != nil {
		logger.LogError("Checksum verification failed for image: %s, error: %v", updateImagePath, verified)
		return fmt.Errorf("checksum verification failed for image: %s, error: %w", updateImagePath, verified)
	}

	logger.LogInfo("Checksum verification successful")

	logger.LogInfo("Decompressing update image file")

	// Ensure cleanup is performed when the function exits
	var loopDev string
	var mounts []string
	secureTempDir := "/opt/OS/abupdate/"

	// creating secure folder
	workDir, err := core.CreateSecureTempDir(secureTempDir)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create work dir error: %v", err)
		return fmt.Errorf("failed to create work dir error: %w", err)
	}

	// Decompress the update image file
	decompressedImagePath, err := core.DecompressImage(workDir, updateImagePath)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to decompress image: %s, error: %v", updateImagePath, err)
		return fmt.Errorf("failed to decompress image: %s, error: %w", updateImagePath, err)
	}
	logger.LogInfo("Decompressed image path: %s", decompressedImagePath)
	// Mount the update image to loop device
	loopDev, err = core.LoopSetup(decompressedImagePath)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount image: %s, error: %v", decompressedImagePath, err)
		return fmt.Errorf("failed to mount image: %s, error: %w", decompressedImagePath, err)
	}
	logger.LogInfo("Setup image to loop device: %s", loopDev)

	// Get the source device from the loop device
	sourceBootDev := fmt.Sprintf("%sp1", loopDev)
	sourceRootfsDev := fmt.Sprintf("%sp2", loopDev)

	// create all needed folders
	imageBootMount := workDir + "/sourceBoot"
	imageRootfsMount := workDir + "/sourceRootfs"
	nextRootfsMount := workDir + "/destRootfs"
	err = core.CreateSecureDir(imageBootMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", imageBootMount, err)
	}
	err = core.CreateSecureDir(imageRootfsMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", imageRootfsMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", imageRootfsMount, err)
	}
	err = core.CreateSecureDir(nextRootfsMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", nextRootfsMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", nextRootfsMount, err)
	}

	// Mount the source image to the created folders
	err = core.MountDevice(sourceBootDev, imageBootMount, true)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", imageBootMount, err)
	}
	mounts = append(mounts, imageBootMount)

	logger.LogInfo("Check PARTUUID if the partition is same as the one in the update image")

	efiSourceFolder := fmt.Sprintf("%s/EFI/Linux", imageBootMount)
	efiSource, err := core.FindFirstFile(efiSourceFolder)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to find EFI source file in: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to find EFI source file in: %s, error: %w", imageBootMount, err)
	}

	boot_uuid, err := core.GetUUIDfromUKI(efiSource)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to find boot uuid from source file in: %s, error: %v", efiSource, err)
		return fmt.Errorf("failed to find boot uuid from source file in: %s, error: %w", efiSource, err)
	}

	// Check to make sure image update is not same image
	rootfsMatch := core.FindPartitionByUUID(boot_uuid)
	// convert to block device
	blockDevice, err := core.GetBlockName(nxtPartition)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to get block device %v", err)
		return fmt.Errorf("failed to get block device %w", err)
	}

	if rootfsMatch != "" && rootfsMatch != blockDevice {
		logger.LogInfo("Exist Partition: \"%s\", uuid: \"%s\"", rootfsMatch, boot_uuid)
		cleanupResources(secureTempDir, loopDev, mounts...)
		return fmt.Errorf("duplicated UUID detected, please check image source")
	}

	logger.LogInfo("update image is different compared to current partition, update needed")

	err = core.MountDevice(sourceRootfsDev, imageRootfsMount, true)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", imageRootfsMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", imageRootfsMount, err)
	}
	mounts = append(mounts, imageRootfsMount)

	err = core.MountDevice(nxtPartition, nextRootfsMount, false)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", nextRootfsMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", nextRootfsMount, err)
	}
	mounts = append(mounts, nextRootfsMount)

	// Proceed with writing the rootfs to the target partition
	err = core.WriteRootfsToPartition(imageRootfsMount, nextRootfsMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to write rootfs to target partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to write rootfs to target partition: %s, error: %w", nxtPartition, err)
	}

	logger.LogInfo("Successfully wrote the rootfs to the target partition: %s", nxtPartition)

	// Copy onboarding variable to the second boot partition
	err = core.CopyOnboardingVariable(nextRootfsMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to copy onboarding variable: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to copy onboarding variable: %s, error: %w", nxtPartition, err)
	}

	logger.LogInfo("Successfully copied onboarding variable to the target partition: %s", nxtPartition)
	// Copy the UKI to the boot partition with a ".bak" extension
	err = core.CopyUKIToBootPartition(efiSource)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to copy UKI to boot partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to copy UKI to boot partition: %s, error: %w", nxtPartition, err)
	}
	logger.LogInfo("Successfully copied UKI to the boot partition: %s", nxtPartition)

	// umount target partition
	err = core.UnmountDevice(nextRootfsMount)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to unmount target partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to unmount target partition: %s, error: %w", nxtPartition, err)
	}

	if devMode {
		err = core.AddLogin(workDir, nxtPartition)
		if err != nil {
			cleanupResources(secureTempDir, loopDev, mounts...)
			logger.LogError("Failed to add login: %s, error: %v", nxtPartition, err)
			return fmt.Errorf("failed to add login: %s, error: %w", nxtPartition, err)
		}
	}

	err = core.RelabelSELinux(workDir, nxtPartition)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to relabel SELinux: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to relabel SELinux: %s, error: %w", nxtPartition, err)
	}

	err = core.SetUUID(nxtPartition, boot_uuid)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to set UUID: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to set UUID: %s, error: %w", nxtPartition, err)
	}

	err = core.SetVerity(workDir, nxtPartition)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to set verity: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to set verity: %s, error: %w", nxtPartition, err)
	}

	cleanupResources(secureTempDir, loopDev, mounts...)
	logger.LogInfo("Write operation completed successfully for partition: %s", nxtPartition)

	return nil
}
