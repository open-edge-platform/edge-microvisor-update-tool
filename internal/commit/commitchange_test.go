package apply

import (
	"testing"
)

func TestCommitChange(t *testing.T) {
	// Call the function
	err := CommitChange()

	// Check for errors
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
