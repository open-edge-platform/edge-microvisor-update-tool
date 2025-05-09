package rollback

import (
	"testing"
)

func TestRollbackChange(t *testing.T) {
	// Call the function
	err := RollbackChange()

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
