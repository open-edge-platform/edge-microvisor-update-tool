package commit

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommitChange_Success(t *testing.T) {
	checkWriteDoneFunc = func() bool { return true }
	getActiveUKIFunc = func() (string, error) { return "/boot/efi/linux.efi", nil }
	executeCommandFunc = func(name string, args ...string) (string, error) {
		assert.Equal(t, "bootctl", name, "unexpected command name")
		assert.Equal(t, []string{"set-default", "linux.efi"}, args, "unexpected command args")
		return "OK", nil
	}
	resetWriteDoneFunc = func() bool { return true }

	err := CommitChange()
	assert.NoError(t, err, "unexpected error")
}

func TestCommitChange_CheckWriteDoneFails(t *testing.T) {
	checkWriteDoneFunc = func() bool { return false }

	err := CommitChange()
	assert.Error(t, err, "expected error but got nil")
}

func TestCommitChange_GetActiveUKIFails(t *testing.T) {
	checkWriteDoneFunc = func() bool { return true }
	getActiveUKIFunc = func() (string, error) { return "", errors.New("getActiveUKI error") }

	err := CommitChange()
	assert.Error(t, err, "expected error but got nil")
}

func TestCommitChange_ExecuteCommandFails(t *testing.T) {
	checkWriteDoneFunc = func() bool { return true }
	getActiveUKIFunc = func() (string, error) { return "/boot/efi/linux.efi", nil }
	executeCommandFunc = func(name string, args ...string) (string, error) {
		return "", errors.New("exec error")
	}
	resetWriteDoneFunc = func() bool { return true }

	err := CommitChange()
	assert.Error(t, err, "expected error but got nil")
}

func TestCommitChange_ResetWriteDoneFails(t *testing.T) {
	checkWriteDoneFunc = func() bool { return true }
	getActiveUKIFunc = func() (string, error) { return "/boot/efi/linux.efi", nil }
	executeCommandFunc = func(name string, args ...string) (string, error) {
		return "OK", nil
	}
	resetWriteDoneFunc = func() bool { return false }

	err := CommitChange()
	assert.Error(t, err, "expected error but got nil")
}
