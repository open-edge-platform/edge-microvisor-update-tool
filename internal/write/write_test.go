package flashing

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

var (
	updateImagePath = "./../testData/valid.raw.gz"
	checksumValue   string
)

func init() {
	// Compute the checksum once and store it in checksumValue
	file, err := os.Open(updateImagePath)
	if err != nil {
		fmt.Printf("failed to open file: %v\n", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		fmt.Printf("failed to compute hash: %v\n", err)
		return
	}
	hashSum := hasher.Sum(nil)
	checksumValue = fmt.Sprintf("%x", hashSum)
}

func resetMocks() {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda3", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda2", nil }
	verifyChecksumFunc = func(path, checksum string) error { return nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }
	loopUnSetupFunc = func(dev string) error { return nil }
	mountDeviceFunc = func(dev, mount string, ro bool, opts ...string) error { return nil }
	findFirstFileFunc = func(path string) (string, error) { return "/tmp/uki.efi", nil }
	getUUIDfromUKIFunc = func(path string) (string, error) { return "mock-uuid", nil }
	findPartitionByUUIDFunc = func(uuid string) string { return "" }
	getBlockNameFunc = func(dev string) (string, error) { return "sda", nil }
	writeRootfsToPartitionFunc = func(src, dst string) error { return nil }
	copyOnboardingVariableFunc = func(path string) error { return nil }
	copyUKIToBootPartitionFunc = func(path string) error { return nil }
	unmountDeviceFunc = func(path string) error { return nil }
	addLoginFunc = func(workDir, dev string) error { return nil }
	relabelSELinuxFunc = func(workDir, dev string) error { return nil }
	setUUIDFunc = func(dev, uuid string) error { return nil }
	setVerityFunc = func(workDir, dev string) error { return nil }
	deleteDirFunc = func(path string) error { return nil }
	createSecureDirFunc = func(path string) error { return nil }
}

func TestWritePartition_VerifyChecksumFails(t *testing.T) {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	// Remove last two digits from the checksum
	alteredChecksum := checksumValue[:len(checksumValue)-2]

	err := WritePartition(updateImagePath, alteredChecksum, false)
	assert.Error(t, err, "expected checksum error")
}

func TestWritePartition_CreateSecureTempDirFails(t *testing.T) {

	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected error creating secure temp dir")
}

func TestWritePartition_DecompressImageFails(t *testing.T) {

	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected decompress error")
}

func TestWritePartition_LoopSetupFails(t *testing.T) {

	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected loop setup error")
}

func TestWritePartition_CreateSecureDirFails(t *testing.T) {

	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }

	call := 0
	createSecureDirFunc = func(path string) error {
		call++
		if call == 2 {
			return errors.New("cannot create folder")
		}
		return nil
	}
	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected error creating secure dir")
}

func TestWritePartition_MountDeviceFails(t *testing.T) {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }
	mountDeviceFunc = func(dev, mount string, ro bool, opts ...string) error { return nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected mount error")
}

func TestWritePartition_FindFirstFileFails(t *testing.T) {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }
	mountDeviceFunc = func(dev, mount string, ro bool, opts ...string) error { return nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected error finding first file")
}

func TestWritePartition_GetUUIDfromUKIFails(t *testing.T) {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }
	mountDeviceFunc = func(dev, mount string, ro bool, opts ...string) error { return nil }
	findFirstFileFunc = func(path string) (string, error) { return "/tmp/uki.efi", nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected error getting UUID from UKI")
}

func TestWritePartition_DuplicateUUID(t *testing.T) {
	getTargetPartitionFunc = func() (string, error) { return "/dev/sda4", nil }
	getActivePartitionFunc = func() (string, error) { return "/dev/sda6", nil }
	createSecureTempDirFunc = func(base string) (string, error) { return "/tmp/mockdir", nil }
	decompressImageFunc = func(workDir, imgPath string) (string, error) { return "/tmp/mockdir/image", nil }
	loopSetupFunc = func(path string) (string, error) { return "/dev/loop0", nil }
	mountDeviceFunc = func(dev, mount string, ro bool, opts ...string) error { return nil }
	findFirstFileFunc = func(path string) (string, error) { return "/tmp/uki.efi", nil }
	getUUIDfromUKIFunc = func(path string) (string, error) { return "mock-uuid", nil }

	err := WritePartition(updateImagePath, checksumValue, false)
	assert.Error(t, err, "expected duplicate UUID error")
}

func TestWritePartition_Success(t *testing.T) {
	resetMocks()
	err := WritePartition(updateImagePath, checksumValue, false)
	assert.NoError(t, err, "Expected no error")
}
