package apply

import (
	"testing"
)

func TestApplyChange(t *testing.T) {
	// Call the function
	err := ApplyChange()

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
