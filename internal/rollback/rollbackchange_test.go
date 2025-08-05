package rollback

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRollbackChange_SwitchesToLinux2_Success(t *testing.T) {
	// Simulate active UKI is "linux.efi"
	getActiveUKIFunc = func() (string, error) {
		return "linux.efi", nil
	}

	executeCommandFunc = func(name string, args ...string) (string, error) {
		assert.Len(t, args, 2, "expected at least 2 arguments")
		assert.Equal(t, "linux-2.efi", args[1], "expected linux-2.efi as target")
		return "OK", nil
	}

	err := RollbackChange()
	assert.NoError(t, err, "unexpected error")
}

func TestRollbackChange_SwitchesToLinux_Success(t *testing.T) {
	// Simulate active UKI is "linux-2.efi"
	getActiveUKIFunc = func() (string, error) {
		return "linux-2.efi", nil
	}

	executeCommandFunc = func(name string, args ...string) (string, error) {
		assert.Len(t, args, 2, "expected at least 2 arguments")
		assert.Equal(t, "linux.efi", args[1], "expected linux.efi as target")
		return "OK", nil
	}

	err := RollbackChange()
	assert.NoError(t, err, "unexpected error")
}

func TestRollbackChange_GetActiveUKIFails(t *testing.T) {
	// Simulate failure in GetActiveUKI
	getActiveUKIFunc = func() (string, error) {
		return "", errors.New("mock error getting active UKI")
	}

	executeCommandFunc = func(name string, args ...string) (string, error) {
		return "", nil
	}

	err := RollbackChange()
	assert.Error(t, err, "expected error but got nil")
}

func TestRollbackChange_ExecuteCommandFails(t *testing.T) {
	// Simulate successful GetActiveUKI but failing ExecuteCommand
	getActiveUKIFunc = func() (string, error) {
		return "linux.efi", nil
	}

	executeCommandFunc = func(name string, args ...string) (string, error) {
		return "", errors.New("mock exec error")
	}

	err := RollbackChange()
	assert.Error(t, err, "expected error but got nil")
}
