package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func executeCommand(cmd *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err = cmd.Execute()
	return buf.String(), err
}

func assertCommandFailure(t *testing.T, cmd string, args ...string) {
	t.Helper()
	output, err := executeCommand(rootCmd, append([]string{cmd}, args...)...)
	if err != nil {
		assert.Contains(t, output, "Error", "Expected 'Error' in output, but not found")
	} else {
		t.Fatalf("Expected failure for '%s', but got success\nOutput: %s", cmd, output)
	}
}

func TestWriteCmdFailure(t *testing.T) {
	assertCommandFailure(t, "write", "../internal/testData/valid.raw.gz", "fbdd36650f73f3afdbf34b9c887e192a3c20d1591741cb79a253075ac192dbba")
}

func TestApplyCmdFailure(t *testing.T) {
	assertCommandFailure(t, "apply")
}

func TestCommitCmdFailure(t *testing.T) {
	assertCommandFailure(t, "commit")
}

func TestRollbackCmdFailure(t *testing.T) {
	assertCommandFailure(t, "rollback")
}

func TestDisplayCmd(t *testing.T) {
	output, err := executeCommand(rootCmd, "display")
	if err != nil {
		t.Fatalf("displayCmd failed: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty output for success, got:\n%s", output)
	}
}
