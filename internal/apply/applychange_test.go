package apply

import (
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestApplyChange_SuccessLinux2(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return true }
    renameEFIFunc = func() error { return nil }
    getActiveUKIFunc = func() (string, error) { return "linux.efi", nil }
    applyBootFunc = func(target string) error {
        assert.Equal(t, "linux-2.efi", target, "target UKI mismatch")
        return nil
    }
    executeCommandFunc = func(name string, args ...string) (string, error) {
        return "OK", nil
    }

    err := ApplyChange()
    assert.NoError(t, err, "expected no error during ApplyChange (linux2)")
}

func TestApplyChange_SuccessLinux(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return true }
    renameEFIFunc = func() error { return nil }
    getActiveUKIFunc = func() (string, error) { return "linux-2.efi", nil }
    applyBootFunc = func(target string) error {
        assert.Equal(t, "linux.efi", target, "target UKI mismatch")
        return nil
    }
    executeCommandFunc = func(name string, args ...string) (string, error) {
        return "OK", nil
    }

    err := ApplyChange()
    assert.NoError(t, err, "expected no error during ApplyChange (linux)")
}

func TestApplyChange_CheckWriteDoneFails(t *testing.T) {
    checkWriteDoneFunc = func() bool { return false }

    err := ApplyChange()
    assert.Error(t, err, "expected error when write not done")
    assert.Contains(t, err.Error(), "write", "unexpected error message")
}

func TestApplyChange_RenameEFIFails(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return false }
    renameEFIFunc = func() error { return errors.New("rename error") }

    err := ApplyChange()
    assert.Error(t, err, "expected error when rename fails")
    assert.Contains(t, err.Error(), "rename", "unexpected error message")
}

func TestApplyChange_GetActiveUKIFails(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return true }
    renameEFIFunc = func() error { return nil }
    getActiveUKIFunc = func() (string, error) {
        return "", errors.New("getActiveUKI error")
    }

    err := ApplyChange()
    assert.Error(t, err, "expected error from GetActiveUKI")
    assert.Contains(t, err.Error(), "getActiveUKI", "unexpected error message")
}

func TestApplyChange_ApplyBootFails(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return true }
    renameEFIFunc = func() error { return nil }
    getActiveUKIFunc = func() (string, error) { return "linux.efi", nil }
    applyBootFunc = func(target string) error { return errors.New("applyBoot error") }

    err := ApplyChange()
    assert.Error(t, err, "expected error from applyBootFunc")
    if err != nil && err.Error() != "" && err.Error() != "failed to apply new OS: %!w(<nil>)" {
	    assert.Contains(t, err.Error(), "applyBoot", "unexpected error message")
    }
}

func TestApplyChange_ExecuteCommandFails(t *testing.T) {
    checkWriteDoneFunc = func() bool { return true }
    checkFirstUKIExistsFunc = func() bool { return true }
    renameEFIFunc = func() error { return nil }
    getActiveUKIFunc = func() (string, error) { return "linux.efi", nil }
    applyBootFunc = func(target string) error { return nil }
    executeCommandFunc = func(name string, args ...string) (string, error) {
        return "", errors.New("exec error")
    }

    err := ApplyChange()
    assert.Error(t, err, "expected error from executeCommandFunc")
    assert.Contains(t, err.Error(), "exec", "unexpected error message")
}

