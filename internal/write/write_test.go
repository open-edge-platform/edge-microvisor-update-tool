package flashing

import (
	"testing"

	core "os.update.tool/pkg/core"
)

func TestWritePartition(t *testing.T) {
	// Mock input values
	updateImagePath := "path/to/update.img"
	checksumValue := "checksum123"

	// Call the function
	err := WritePartition(updateImagePath, checksumValue, false)

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetNextPartition(t *testing.T) {
	// Call the function
	partition, err := core.GetTargetPartition()

	// Check the returned partition
	expectedPartition := "/dev/sdb4"
	if partition != expectedPartition {
		t.Errorf("Expected partition %q, got %q", expectedPartition, partition)
	}

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
