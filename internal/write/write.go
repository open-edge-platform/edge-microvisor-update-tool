package flashing

import (
	"fmt"

	core "os.update.tool/pkg/core"
	"os.update.tool/pkg/logger"
)

// Injectable function variables for testing
var (
	getTargetPartitionFunc     = core.GetTargetPartition
	getActivePartitionFunc     = core.GetActivePartition
	verifyChecksumFunc         = core.VerifyChecksum
	createSecureTempDirFunc    = core.CreateSecureTempDir
	decompressImageFunc        = core.DecompressImage
	loopSetupFunc              = core.LoopSetup
	mountDeviceFunc            = core.MountDevice
	findFirstFileFunc          = core.FindFirstFile
	getUUIDfromUKIFunc         = core.GetUUIDfromUKI
	findPartitionByUUIDFunc    = core.FindPartitionByUUID
	getBlockNameFunc           = core.GetBlockName
	writeRootfsToPartitionFunc = core.WriteRootfsToPartition
	copyOnboardingVariableFunc = core.CopyOnboardingVariable
	copyUKIToBootPartitionFunc = core.CopyUKIToBootPartition
	unmountDeviceFunc          = core.UnmountDevice
	addLoginFunc               = core.AddLogin
	relabelSELinuxFunc         = core.RelabelSELinux
	setUUIDFunc                = core.SetUUID
	setVerityFunc              = core.SetVerity
	deleteDirFunc              = core.DeleteDir
	createSecureDirFunc        = core.CreateSecureDir
	loopUnSetupFunc            = core.LoopUnSetup
)

// Helper function to clean up resources
//
//nolint:unparam
func cleanupResources(tempDir string, loopDev string, mounts ...string) {
	for _, mount := range mounts {
		if err := unmountDeviceFunc(mount); err != nil {
			fmt.Printf("Failed to unmount device %s: %v\n", mount, err)
		}
	}
	if tempDir != "" {
		if err := deleteDirFunc(tempDir); err != nil {
			fmt.Printf("Failed to delete directory %s: %v\n", tempDir, err)
		}
	}
	if loopDev != "" {
		if err := loopUnSetupFunc(loopDev); err != nil {
			fmt.Printf("Failed to loop device %s: %v\n", loopDev, err)
		}
	}
}

func WritePartition(updateImagePath string, checksumValue string, devMode bool) error {
	if devMode {
		logger.LogInfo("Development mode is active.")
	}

	nxtPartition, err := getTargetPartitionFunc()
	if err != nil {
		logger.LogError("Failed to get next partition: %v", err)
		return fmt.Errorf("failed to get next partition: %w", err)
	}

	currPartition, err := getActivePartitionFunc()
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
	verified := verifyChecksumFunc(updateImagePath, checksumValue)
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
	workDir, err := createSecureTempDirFunc(secureTempDir)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create work dir error: %v", err)
		return fmt.Errorf("failed to create work dir error: %w", err)
	}

	// Decompress the update image file
	decompressedImagePath, err := decompressImageFunc(workDir, updateImagePath)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to decompress image: %s, error: %v", updateImagePath, err)
		return fmt.Errorf("failed to decompress image: %s, error: %w", updateImagePath, err)
	}
	logger.LogInfo("Decompressed image path: %s", decompressedImagePath)

	// Mount the update image to loop device
	loopDev, err = loopSetupFunc(decompressedImagePath)
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

	if err := createSecureDirFunc(imageBootMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", imageBootMount, err)
	}
	if err := createSecureDirFunc(imageRootfsMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", imageRootfsMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", imageRootfsMount, err)
	}
	if err := createSecureDirFunc(nextRootfsMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to create folder: %s, error: %v", nextRootfsMount, err)
		return fmt.Errorf("failed to create folder: %s, error: %w", nextRootfsMount, err)
	}

	// Mount the source image to the created folders
	if err := mountDeviceFunc(sourceBootDev, imageBootMount, true); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", imageBootMount, err)
	}
	mounts = append(mounts, imageBootMount)

	logger.LogInfo("Check PARTUUID if the partition is same as the one in the update image")
	efiSourceFolder := fmt.Sprintf("%s/EFI/Linux", imageBootMount)

	efiSource, err := findFirstFileFunc(efiSourceFolder)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to find EFI source file in: %s, error: %v", imageBootMount, err)
		return fmt.Errorf("failed to find EFI source file in: %s, error: %w", imageBootMount, err)
	}

	bootUUID, err := getUUIDfromUKIFunc(efiSource)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to find boot uuid from source file in: %s, error: %v", efiSource, err)
		return fmt.Errorf("failed to find boot uuid from source file in: %s, error: %w", efiSource, err)
	}

	rootfsMatch := findPartitionByUUIDFunc(bootUUID)
	blockDevice, err := getBlockNameFunc(nxtPartition)
	if err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to get block device %v", err)
		return fmt.Errorf("failed to get block device %w", err)
	}
	if rootfsMatch != "" && rootfsMatch != blockDevice {
		logger.LogInfo("Exist Partition: \"%s\", uuid: \"%s\"", rootfsMatch, bootUUID)
		cleanupResources(secureTempDir, loopDev, mounts...)
		return fmt.Errorf("duplicated UUID detected, please check image source")
	}

	logger.LogInfo("update image is different compared to current partition, update needed")
	if err := mountDeviceFunc(sourceRootfsDev, imageRootfsMount, true); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", imageRootfsMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", imageRootfsMount, err)
	}
	mounts = append(mounts, imageRootfsMount)

	if err := mountDeviceFunc(nxtPartition, nextRootfsMount, false); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to mount folder: %s, error: %v", nextRootfsMount, err)
		return fmt.Errorf("failed to mount folder: %s, error: %w", nextRootfsMount, err)
	}
	mounts = append(mounts, nextRootfsMount)

	if err := writeRootfsToPartitionFunc(imageRootfsMount, nextRootfsMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to write rootfs to target partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to write rootfs to target partition: %s, error: %w", nxtPartition, err)
	}

	logger.LogInfo("Successfully wrote the rootfs to the target partition: %s", nxtPartition)

	if err := copyOnboardingVariableFunc(nextRootfsMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to copy onboarding variable: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to copy onboarding variable: %s, error: %w", nxtPartition, err)
	}
	logger.LogInfo("Successfully copied onboarding variable to the target partition: %s", nxtPartition)

	if err := copyUKIToBootPartitionFunc(efiSource); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to copy UKI to boot partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to copy UKI to boot partition: %s, error: %w", nxtPartition, err)
	}
	logger.LogInfo("Successfully copied UKI to the boot partition: %s", nxtPartition)

	if err := unmountDeviceFunc(nextRootfsMount); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to unmount target partition: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to unmount target partition: %s, error: %w", nxtPartition, err)
	}

	if devMode {
		if err := addLoginFunc(workDir, nxtPartition); err != nil {
			cleanupResources(secureTempDir, loopDev, mounts...)
			logger.LogError("Failed to add login: %s, error: %v", nxtPartition, err)
			return fmt.Errorf("failed to add login: %s, error: %w", nxtPartition, err)
		}
	}

	if err := relabelSELinuxFunc(workDir, nxtPartition); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to relabel SELinux: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to relabel SELinux: %s, error: %w", nxtPartition, err)
	}

	if err := setUUIDFunc(nxtPartition, bootUUID); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to set UUID: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to set UUID: %s, error: %w", nxtPartition, err)
	}

	if err := setVerityFunc(workDir, nxtPartition); err != nil {
		cleanupResources(secureTempDir, loopDev, mounts...)
		logger.LogError("Failed to set verity: %s, error: %v", nxtPartition, err)
		return fmt.Errorf("failed to set verity: %s, error: %w", nxtPartition, err)
	}

	cleanupResources(secureTempDir, loopDev, mounts...)
	logger.LogInfo("Write operation completed successfully for partition: %s", nxtPartition)
	return nil
}
